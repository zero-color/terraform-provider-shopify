package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"

	goshopify "github.com/bold-commerce/go-shopify/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zero-clor/terraform-provider-shopify/internal/shopify"
	"github.com/zero-clor/terraform-provider-shopify/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PageResource{}
var _ resource.ResourceWithImportState = &PageResource{}

// PageResource defines the resource implementation.
type PageResource struct {
	client *shopify.Client
}

func NewPageResource() resource.Resource {
	return &PageResource{}
}

// PageResourceModel describes the resource data model.
type PageResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Handle         types.String `tfsdk:"handle"`
	Author         types.String `tfsdk:"author"`
	Title          types.String `tfsdk:"title"`
	BodyHTML       types.String `tfsdk:"body_html"`
	TemplateSuffix types.String `tfsdk:"template_suffix"`
	Published      types.Bool   `tfsdk:"published"`
	PublishedAt    types.String `tfsdk:"published_at"`
}

func (r *PageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_page"
}

func (r *PageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Page definitions enable you to define additional validation constraints for metafields, and enable the merchant to edit metafield values in context.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique numeric identifier for the page.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"handle": schema.StringAttribute{
				MarkdownDescription: "A unique, human-friendly string for the page, generated automatically from its title. In themes, the Liquid templating language refers to a page by its handle.",
				Required:            true,
			},
			"author": schema.StringAttribute{
				MarkdownDescription: "The name of the person who created the page.",
				Required:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the page.",
				Required:            true,
			},
			"body_html": schema.StringAttribute{
				MarkdownDescription: "The text content of the page, complete with HTML markup.",
				Required:            true,
			},
			"template_suffix": schema.StringAttribute{
				MarkdownDescription: "he suffix of the template that is used to render the page. If the value is an empty string or null, then the default page template is used.",
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Computed:            true,
			},
			"published": schema.BoolAttribute{
				MarkdownDescription: "Whether the page is published. If true, the page is visible to customers. If false, the page is hidden from customers.",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Computed:            true,
			},
			"published_at": schema.StringAttribute{
				MarkdownDescription: "The date and time (ISO 8601 format) when the page was published.",
				Computed:            true,
			},
		},
	}
}

func (r *PageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.client, _ = req.ProviderData.(*shopify.Client)
}

func (r *PageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	page := goshopify.Page{
		Author:         data.Author.ValueString(),
		Handle:         data.Handle.ValueString(),
		Title:          data.Title.ValueString(),
		BodyHTML:       data.BodyHTML.ValueString(),
		TemplateSuffix: data.TemplateSuffix.ValueString(),
		Published:      utils.Ptr(data.Published.ValueBool()),
	}
	createdPage, err := r.client.Page().Create(ctx, page)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to create a page", err.Error()))
		return
	}

	createdData := convertPageToResourceModel(createdPage)
	resp.Diagnostics.Append(resp.State.Set(ctx, createdData)...)
}

func (r *PageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseUint(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to parse ID", err.Error()))
		return
	}
	page, err := r.client.Page().Get(ctx, id, nil)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to get page", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, convertPageToResourceModel(page))...)
}

func (r *PageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseUint(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to parse ID", err.Error()))
		return
	}
	page := goshopify.Page{
		Id:             id,
		Author:         data.Author.ValueString(),
		Handle:         data.Handle.ValueString(),
		Title:          data.Title.ValueString(),
		BodyHTML:       data.BodyHTML.ValueString(),
		TemplateSuffix: data.TemplateSuffix.ValueString(),
		Published:      utils.Ptr(data.Published.ValueBool()),
	}
	updatedPage, err := r.client.Page().Update(ctx, page)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to update page", err.Error()))
		return
	}

	updatedData := convertPageToResourceModel(updatedPage)
	resp.Diagnostics.Append(resp.State.Set(ctx, updatedData)...)
}

func (r *PageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseUint(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to parse ID", err.Error()))
		return
	}
	if err := r.client.Page().Delete(ctx, id); err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Failed to delete page", err.Error()))
		return
	}
}

func (r *PageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func convertPageToResourceModel(page *goshopify.Page) *PageResourceModel {
	var publishedAt *string
	if page.PublishedAt != nil {
		publishedAtStr := page.PublishedAt.String()
		publishedAt = &publishedAtStr
	}
	return &PageResourceModel{
		ID:             types.StringValue(strconv.FormatUint(page.Id, 10)),
		Handle:         types.StringValue(page.Handle),
		Author:         types.StringValue(page.Author),
		Title:          types.StringValue(page.Title),
		BodyHTML:       types.StringValue(page.BodyHTML),
		TemplateSuffix: types.StringValue(page.TemplateSuffix),
		Published:      types.BoolValue(publishedAt != nil),
		PublishedAt:    types.StringPointerValue(publishedAt),
	}
}
