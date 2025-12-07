// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"os"

	goshopify "github.com/bold-commerce/go-shopify/v4"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zero-clor/terraform-provider-shopify/internal/shopify"
	"github.com/zero-clor/terraform-provider-shopify/internal/utils"
)

// Ensure ShopifyProvider satisfies various provider interfaces.
var _ provider.Provider = &ShopifyProvider{}
var _ provider.ProviderWithFunctions = &ShopifyProvider{}

// ShopifyProvider defines the provider implementation.
type ShopifyProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ShopifyProviderModel describes the provider data model.
type ShopifyProviderModel struct {
	Shop                types.String `tfsdk:"shop"`
	APIVersion          types.String `tfsdk:"api_version"`
	APIKey              types.String `tfsdk:"api_key"`
	APISecretKey        types.String `tfsdk:"api_secret_key"`
	AdminAPIAccessToken types.String `tfsdk:"admin_api_access_token"`
}

func (p *ShopifyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "shopify"
	resp.Version = p.version
}

func (p *ShopifyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"shop": schema.StringAttribute{
				MarkdownDescription: "The shopName parameter is the shop's myshopify domain, e.g. `theshop.myshopify.com`, or simply `theshop`. Defaults to the env variable `SHOPIFY_SHOP`.",
				Optional:            true,
			},
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Shopify API version. Defaults to the env variable `SHOPIFY_API_VERSION`.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Shopify app API key. Defaults to the env variable `SHOPIFY_API_KEY`.",
				Optional:            true,
			},
			"api_secret_key": schema.StringAttribute{
				MarkdownDescription: "Shopify app API secret key. Defaults to the env variable `SHOPIFY_API_SECRET_KEY`.",
				Optional:            true,
				Sensitive:           true,
			},
			"admin_api_access_token": schema.StringAttribute{
				MarkdownDescription: "Shopify Admin API access token.  Defaults to the env variable `SHOPIFY_ADMIN_API_ACCESS_TOKEN`.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *ShopifyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ShopifyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	shop := readOrEnvDefault(data.Shop, "SHOPIFY_SHOP")
	if shop == "" {
		resp.Diagnostics.AddError("Unable to find shop", "shop cannot be an empty string")
	}
	apiVersion := readOrEnvDefault(data.APIVersion, "SHOPIFY_API_VERSION")
	if apiVersion == "" {
		resp.Diagnostics.AddError("Unable to find api_version", "api_version cannot be an empty string")
	}
	apiKey := readOrEnvDefault(data.APIKey, "SHOPIFY_API_KEY")
	if apiKey == "" {
		resp.Diagnostics.AddError("Unable to find api_key", "api_key cannot be an empty string")
	}
	apiSecretKey := readOrEnvDefault(data.APISecretKey, "SHOPIFY_API_SECRET_KEY")
	if apiSecretKey == "" {
		resp.Diagnostics.AddError("Unable to find api_secret_key", "api_secret_key cannot be an empty string")
	}
	adminAPIAccessToken := readOrEnvDefault(data.AdminAPIAccessToken, "SHOPIFY_ADMIN_API_ACCESS_TOKEN")
	if adminAPIAccessToken == "" {
		resp.Diagnostics.AddError("Unable to find admin_api_access_token", "admin_api_access_token cannot be an empty string")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	opts := []goshopify.Option{goshopify.WithVersion(apiVersion)}
	httpClient := http.DefaultClient
	httpClient.Transport = utils.NewDebugTransport(http.DefaultTransport)
	opts = append(opts, goshopify.WithHTTPClient(httpClient))

	app := goshopify.App{
		ApiKey:    apiKey,
		ApiSecret: apiSecretKey,
	}
	shopifyRawClient, err := goshopify.NewClient(
		app,
		shop,
		adminAPIAccessToken,
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Shopify client",
			err.Error(),
		)
		return
	}

	shopifyClient := shopify.NewClient(shopifyRawClient)
	resp.DataSourceData = shopifyClient
	resp.ResourceData = shopifyClient
}

func (p *ShopifyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMetafieldDefinitionResource,
		NewMetaobjectDefinitionResource,
		NewPageResource,
	}
}

func (p *ShopifyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *ShopifyProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ShopifyProvider{
			version: version,
		}
	}
}

func readOrEnvDefault(str types.String, envVarKey string) string {
	if !str.IsNull() {
		return str.ValueString()
	}
	return os.Getenv(envVarKey)
}
