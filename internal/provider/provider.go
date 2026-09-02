package provider

import (
	"context"
	"os"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/Metronome-Industries/metronome-go/v3/option"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &MetronomeProvider{}

type MetronomeProvider struct {
	version string
}

type MetronomeProviderModel struct {
	BearerToken types.String `tfsdk:"bearer_token"`
	BaseURL     types.String `tfsdk:"base_url"`
}

// ProviderData is passed to every resource and data source via req.ProviderData.
type ProviderData struct {
	Client *metronome.Client
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MetronomeProvider{version: version}
	}
}

func (p *MetronomeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "metronome"
	resp.Version = p.version
}

func (p *MetronomeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Metronome (usage-based billing) configuration as code: billable metrics, products, rate cards and rates, packages, alerts, notifications, customers and contracts. Authenticate with an API token created in the Metronome app under Developer → API tokens. Metronome exposes one API host; tenants (production, Sandbox) are distinguished by the token you use.",
		Attributes: map[string]schema.Attribute{
			"bearer_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Metronome API token. Can also be set via the METRONOME_BEARER_TOKEN environment variable. Use a token minted in your Sandbox tenant for testing.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Metronome API base URL. Defaults to https://api.metronome.com. Can also be set via the METRONOME_BASE_URL environment variable; mainly useful for pointing the provider at a mock server in tests.",
			},
		},
	}
}

func (p *MetronomeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Metronome provider")

	var config MetronomeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.BearerToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("bearer_token"),
			"Unknown Metronome API token",
			"The provider cannot create the Metronome client as there is an unknown configuration value for bearer_token.",
		)
		return
	}
	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Metronome base URL",
			"The provider cannot create the Metronome client as there is an unknown configuration value for base_url.",
		)
		return
	}

	token := os.Getenv("METRONOME_BEARER_TOKEN")
	if !config.BearerToken.IsNull() {
		token = config.BearerToken.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("bearer_token"),
			"Missing Metronome API token",
			"Set bearer_token in the provider configuration or the METRONOME_BEARER_TOKEN environment variable.",
		)
		return
	}

	baseURL := os.Getenv("METRONOME_BASE_URL")
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	opts := []option.RequestOption{option.WithBearerToken(token)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := metronome.NewClient(opts...)

	data := ProviderData{Client: &client}
	resp.DataSourceData = data
	resp.ResourceData = data

	tflog.Info(ctx, "Configured Metronome provider")
}

func (p *MetronomeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *MetronomeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
