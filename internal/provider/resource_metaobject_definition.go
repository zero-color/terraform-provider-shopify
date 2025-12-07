package provider

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/zero-clor/terraform-provider-shopify/internal/shopify"
	"github.com/zero-clor/terraform-provider-shopify/internal/utils"
	"github.com/zero-clor/terraform-provider-shopify/pkg/xslice"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MetaobjectDefinitionResource{}
var _ resource.ResourceWithImportState = &MetaobjectDefinitionResource{}

// MetaobjectDefinitionResource defines the resource implementation.
type MetaobjectDefinitionResource struct {
	client *shopify.Client
}

func NewMetaobjectDefinitionResource() resource.Resource {
	return &MetaobjectDefinitionResource{}
}

type MetaobjectFieldDefinitionResourceModel struct {
}

// MetaobjectDefinitionResourceModel describes the resource data model.
type MetaobjectDefinitionResourceModel struct {
	ID                types.String                      `tfsdk:"id"`
	Name              types.String                      `tfsdk:"name"`
	Type              types.String                      `tfsdk:"type"`
	Description       types.String                      `tfsdk:"description"`
	DisplayNameKey    types.String                      `tfsdk:"display_name_key"`
	FieldDefinitions  []*MetaobjectFieldDefinitionModel `tfsdk:"field_definitions"`
	HasThumbnailField types.Bool                        `tfsdk:"has_thumbnail_field"`
	Access            types.Object                      `tfsdk:"access"`
}

type MetaobjectDefinitionAccessModel struct {
	Admin      types.String `tfsdk:"admin"`
	Storefront types.String `tfsdk:"storefront"`
}

func (m *MetaobjectDefinitionAccessModel) toTerraformObject(ctx context.Context) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, map[string]attr.Type{
		"admin":      types.StringType,
		"storefront": types.StringType,
	}, m)
}

func (m *MetaobjectDefinitionAccessModel) toShopifyModel() *shopify.MetaobjectAccess {
	storefront := m.Storefront.ValueString()
	if storefront == "LEGACY_LIQUID_ONLY" {
		storefront = ""
	}
	return &shopify.MetaobjectAccess{
		Admin:      m.Admin.ValueString(),
		Storefront: storefront,
	}
}

// MetaobjectFieldDefinitionModel describes the metaobject field definition data model.
type MetaobjectFieldDefinitionModel struct {
	Key         types.String                          `tfsdk:"key"`
	Name        types.String                          `tfsdk:"name"`
	Description types.String                          `tfsdk:"description"`
	Type        types.String                          `tfsdk:"type"`
	Required    types.Bool                            `tfsdk:"required"`
	Validations []*MetafieldDefinitionValidationModel `tfsdk:"validations"`
}

func (r *MetaobjectDefinitionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metaobject_definition"
}

func (r *MetaobjectDefinitionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides the definition of a generic object structure composed of metafields.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique ID of the metaobject.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The human-readable name for the metaobject definition.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: `The type of the object definition. Defines the namespace of associated metafields.`,
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description for the metaobject definition.",
				Optional:            true,
			},
			"display_name_key": schema.StringAttribute{
				MarkdownDescription: "The key of a field to reference as the display name for each object.",
				Optional:            true,
			},
			"field_definitions": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: `The key of the new field definition. This can't be changed.
Must be 3-64 characters long and only contain alphanumeric, hyphen, and underscore characters.
`,
							Required: true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "A human-readable name for the field. This can be changed at any time.",
							Optional:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "An administrative description of the field.",
							Optional:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The metafield type applied to values of the field. If the type is changed, the field will be recreated.",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								utils.LogAttributeChangeModifier(func(ctx context.Context, req planmodifier.StringRequest) diag.Diagnostics {
									return diag.Diagnostics{diag.NewWarningDiagnostic(
										"Changing the type will recreate the field.",
										"Changing the type of the field definition will recreate the field. It will delete the existing data associated with the field.",
									)}
								},
									"Changing the type will recreate the field.",
									"Changing the type will recreate the field.",
								),
							},
						},
						"required": schema.BoolAttribute{
							MarkdownDescription: "Whether metaobjects require a saved value for the field.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"validations": schema.ListNestedAttribute{
							MarkdownDescription: "Custom validations that apply to values assigned to the field. Refer to the list of [supported validations](https://shopify.dev/docs/apps/build/custom-data/metafields/definitions/list-of-validation-options).",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name for the metafield definition validation.",
										Required:            true,
									},
									"value": schema.StringAttribute{
										MarkdownDescription: "The value for the metafield definition validation.",
										Required:            true,
									},
								},
							},
							Optional: true,
						},
					},
				},
				Required: true,
			},
			"has_thumbnail_field": schema.BoolAttribute{
				MarkdownDescription: "Whether this metaobject definition has field whose type can visually represent a metaobject with the thumbnailField.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.SingleNestedAttribute{
				MarkdownDescription: "The access settings associated with the metafield definition.",
				Attributes: map[string]schema.Attribute{
					"admin": schema.StringAttribute{
						MarkdownDescription: "The default admin access setting used for the metafields under this definition.",
						Optional:            true,
						Computed:            true,
					},
					"storefront": schema.StringAttribute{
						MarkdownDescription: "The storefront access setting used for the metafields under this definition.",
						Optional:            true,
						Computed:            true,
					},
				},
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *MetaobjectDefinitionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.client, _ = req.ProviderData.(*shopify.Client)
}

func (r *MetaobjectDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MetaobjectDefinitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var shopifyFieldDefinitions []*shopify.MetaobjectFieldDefinitionCreateInput
	for _, fieldDefinitionModel := range data.FieldDefinitions {
		shopifyFieldDefinitions = append(shopifyFieldDefinitions, convertMetaobjectFieldDefinitionModelToCreateInput(fieldDefinitionModel))
	}

	var displayNameKey *string
	if data.DisplayNameKey.ValueString() != "" {
		displayNameKey = data.DisplayNameKey.ValueStringPointer()
	}
	input := shopify.MetaobjectDefinitionCreateInput{
		Type:             data.Type.ValueString(),
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueStringPointer(),
		DisplayNameKey:   displayNameKey,
		FieldDefinitions: shopifyFieldDefinitions,
	}
	if !data.Access.IsNull() && !data.Access.IsUnknown() {
		var access MetaobjectDefinitionAccessModel
		resp.Diagnostics.Append(data.Access.As(ctx, &access, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		input.Access = access.toShopifyModel()
	}
	createdMetaobjectDefinition, err := r.client.CreateMetaobjectDefinition(ctx, &input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create metaobject definition, got error: %s", err))
		return
	}

	createdData, diag := convertMetaobjectDefinitionToResourceModel(ctx, createdMetaobjectDefinition, &data)
	tflog.Trace(ctx, "created a metaobject definition", map[string]interface{}{
		"id": createdData.ID,
	})
	if resp.Diagnostics.Append(diag...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, createdData)...)
}

func (r *MetaobjectDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MetaobjectDefinitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metaobjectDefinition, err := r.client.GetMetaobjectDefinition(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read metaobject definition, got error: %s", err))
		return
	}
	metaobjectDefinitionModel, diags := convertMetaobjectDefinitionToResourceModel(ctx, metaobjectDefinition, &data)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, metaobjectDefinitionModel)...)
}

func (r *MetaobjectDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MetaobjectDefinitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var oldFieldDefinitions []*MetaobjectFieldDefinitionModel
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("field_definitions"), &oldFieldDefinitions)...)
	if resp.Diagnostics.HasError() {
		return
	}
	oldFieldDefinitionMap := make(map[string]*MetaobjectFieldDefinitionModel, len(oldFieldDefinitions))
	for _, fieldDefinition := range oldFieldDefinitions {
		oldFieldDefinitionMap[fieldDefinition.Key.ValueString()] = fieldDefinition
	}

	var fieldDefinitions1stReq []*shopify.MetaobjectFieldDefinitionOperationInput
	var fieldDefinitions2ndReq []*shopify.MetaobjectFieldDefinitionOperationInput
	var recreateFieldDefinitions []string
	for _, newFieldDef := range data.FieldDefinitions {
		if oldFieldDef, ok := oldFieldDefinitionMap[newFieldDef.Key.ValueString()]; ok {
			delete(oldFieldDefinitionMap, newFieldDef.Key.ValueString())
			if reflect.DeepEqual(oldFieldDef, newFieldDef) {
				continue
			}
			if !newFieldDef.Type.Equal(oldFieldDef.Type) {
				fieldDefinitions1stReq = append(fieldDefinitions1stReq, &shopify.MetaobjectFieldDefinitionOperationInput{
					Delete: &shopify.MetaobjectFieldDefinitionDeleteInput{
						Key: oldFieldDef.Key.ValueString(),
					},
				})
				fieldDefinitions2ndReq = append(fieldDefinitions2ndReq, &shopify.MetaobjectFieldDefinitionOperationInput{
					Create: convertMetaobjectFieldDefinitionModelToCreateInput(newFieldDef),
				})
				recreateFieldDefinitions = append(recreateFieldDefinitions, newFieldDef.Key.ValueString())
			} else {
				fieldDefinitions1stReq = append(fieldDefinitions1stReq, &shopify.MetaobjectFieldDefinitionOperationInput{
					Update: &shopify.MetaobjectFieldDefinitionUpdateInput{
						Key:         newFieldDef.Key.ValueString(),
						Name:        newFieldDef.Name.ValueStringPointer(),
						Description: newFieldDef.Description.ValueStringPointer(),
						Required:    newFieldDef.Required.ValueBool(),
						Validations: convertValidationModelsToValidations(newFieldDef.Validations),
					},
				})
			}
		} else {
			fieldDefinitions1stReq = append(fieldDefinitions1stReq, &shopify.MetaobjectFieldDefinitionOperationInput{
				Create: convertMetaobjectFieldDefinitionModelToCreateInput(newFieldDef),
			})
		}
	}
	if len(recreateFieldDefinitions) > 0 {
		tflog.Warn(ctx, "")
	}

	for _, oldFieldDef := range oldFieldDefinitionMap {
		fieldDefinitions1stReq = append(fieldDefinitions1stReq, &shopify.MetaobjectFieldDefinitionOperationInput{
			Delete: &shopify.MetaobjectFieldDefinitionDeleteInput{
				Key: oldFieldDef.Key.ValueString(),
			},
		})
	}

	var displayNameKey *string
	if data.DisplayNameKey.ValueString() != "" {
		displayNameKey = data.DisplayNameKey.ValueStringPointer()
	}
	input1stReq := shopify.MetaobjectDefinitionUpdateInput{
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueStringPointer(),
		DisplayNameKey:   displayNameKey,
		FieldDefinitions: fieldDefinitions1stReq,
	}
	if !data.Access.IsNull() && !data.Access.IsUnknown() {
		var access MetaobjectDefinitionAccessModel
		resp.Diagnostics.Append(data.Access.As(ctx, &access, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		input1stReq.Access = access.toShopifyModel()
	}
	updatedMetaobjectDefinition, err := r.client.UpdateMetaobjectDefinition(ctx, data.ID.ValueString(), &input1stReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update metaobject definition, got error: %s", err))
		return
	}
	updateData, diags := convertMetaobjectDefinitionToResourceModel(ctx, updatedMetaobjectDefinition, &data)
	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	if len(fieldDefinitions2ndReq) > 0 {
		input2ndReq := shopify.MetaobjectDefinitionUpdateInput{
			Name:             data.Name.ValueString(),
			Description:      data.Description.ValueStringPointer(),
			DisplayNameKey:   displayNameKey,
			FieldDefinitions: fieldDefinitions2ndReq,
		}
		updatedMetaobjectDefinition, err := r.client.UpdateMetaobjectDefinition(ctx, data.ID.ValueString(), &input2ndReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update metaobject definition, got error: %s", err))
			return
		}
		updateData, diags = convertMetaobjectDefinitionToResourceModel(ctx, updatedMetaobjectDefinition, &data)
		if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updateData)...)
}

func (r *MetaobjectDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MetaobjectDefinitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMetaobjectDefinition(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete metaobject definition, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "deleted a metaobject definition", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *MetaobjectDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func convertMetaobjectDefinitionToResourceModel(ctx context.Context, definition *shopify.MetaobjectDefinition, data *MetaobjectDefinitionResourceModel) (*MetaobjectDefinitionResourceModel, diag.Diagnostics) {
	access, diags := convertAccessToModel(definition.Access).toTerraformObject(ctx)
	if diags.HasError() {
		return nil, diags
	}
	fieldDefinitionModels := make([]*MetaobjectFieldDefinitionModel, 0, len(definition.FieldDefinitions))
	for _, fieldDefinition := range definition.FieldDefinitions {
		fieldDefinitionData, _ := xslice.FindBy(data.FieldDefinitions, func(v *MetaobjectFieldDefinitionModel) bool {
			return v.Key.ValueString() == fieldDefinition.Key
		})
		fieldDefinitionModels = append(fieldDefinitionModels, convertMetaobjectFieldDefinitionToModel(fieldDefinition, fieldDefinitionData))
	}

	// Sort field definitions by order in the original data not to produce unnecessary diffs
	fieldDefinitionOrderMap := make(map[string]int, len(data.FieldDefinitions))
	for i, fieldDefinition := range data.FieldDefinitions {
		fieldDefinitionOrderMap[fieldDefinition.Key.ValueString()] = i
	}
	sort.Slice(fieldDefinitionModels, func(i, j int) bool {
		return fieldDefinitionOrderMap[fieldDefinitionModels[i].Key.ValueString()] < fieldDefinitionOrderMap[fieldDefinitionModels[j].Key.ValueString()]
	})

	// Shopify API handles empty string and null as the same value
	// So not to produce inconsistency after apply, we'll set the same value as the plan if it's empoty
	description := types.StringValue(definition.Description)
	if definition.Description == "" && data.Description.IsNull() {
		description = types.StringNull()
	}

	return &MetaobjectDefinitionResourceModel{
		ID:                types.StringValue(definition.ID),
		Name:              types.StringValue(definition.Name),
		Type:              types.StringValue(definition.Type),
		Description:       description,
		DisplayNameKey:    types.StringPointerValue(definition.DisplayNameKey),
		FieldDefinitions:  fieldDefinitionModels,
		HasThumbnailField: types.BoolValue(definition.HasThumbnailField),
		Access:            access,
	}, nil
}

func convertAccessToModel(access *shopify.MetaobjectAccess) *MetaobjectDefinitionAccessModel {
	return &MetaobjectDefinitionAccessModel{
		Admin:      types.StringValue(access.Admin),
		Storefront: types.StringValue(access.Storefront),
	}
}

func convertMetaobjectFieldDefinitionToModel(definition *shopify.MetaobjectFieldDefinition, model *MetaobjectFieldDefinitionModel) *MetaobjectFieldDefinitionModel {
	description := types.StringValue(definition.Description)
	if definition.Description == "" && model != nil && model.Description.IsNull() {
		description = types.StringNull()
	}
	return &MetaobjectFieldDefinitionModel{
		Key:         types.StringValue(definition.Key),
		Name:        types.StringValue(definition.Name),
		Description: description,
		Type:        types.StringValue(definition.Type.Name),
		Required:    types.BoolValue(definition.Required),
		Validations: convertValidationsToModels(definition.Validations),
	}
}

func convertMetaobjectFieldDefinitionModelToCreateInput(model *MetaobjectFieldDefinitionModel) *shopify.MetaobjectFieldDefinitionCreateInput {
	return &shopify.MetaobjectFieldDefinitionCreateInput{
		Key:         model.Key.ValueString(),
		Name:        model.Name.ValueStringPointer(),
		Description: model.Description.ValueStringPointer(),
		Type:        model.Type.ValueString(),
		Required:    model.Required.ValueBool(),
		Validations: convertValidationModelsToValidations(model.Validations),
	}
}
