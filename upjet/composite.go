package upjet

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Composite returns a single Terraform Plugin Framework provider serving the
// resources of both framework based PingOne providers, which are configured
// with different client types. Each client is configured lazily, the first
// time a resource that needs it is configured.
func Composite(version string) provider.Provider {
	return &compositeProvider{
		legacy: FrameworkLegacySDK(version),
		modern: Framework(version),
	}
}

type compositeProvider struct {
	legacy provider.Provider
	modern provider.Provider
}

var _ provider.Provider = &compositeProvider{}

func (p *compositeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	p.legacy.Metadata(ctx, req, resp)
}

// Schema returns the schema of the legacy provider. The Terraform mux server
// requires both providers to have the same provider schema.
func (p *compositeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	p.legacy.Schema(ctx, req, resp)
}

func (p *compositeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	ctx = context.WithoutCancel(ctx)
	resp.ResourceData = &compositeData{
		legacy: newLazyClient(ctx, p.legacy, req),
		modern: newLazyClient(ctx, p.modern, req),
	}
}

func (p *compositeProvider) Resources(_ context.Context) []func() resource.Resource {
	ctx := context.Background()
	var out []func() resource.Resource
	for _, r := range p.legacy.Resources(ctx) {
		out = append(out, wrap(r, func(d *compositeData) *lazyClient { return d.legacy }))
	}
	for _, r := range p.modern.Resources(ctx) {
		out = append(out, wrap(r, func(d *compositeData) *lazyClient { return d.modern }))
	}
	return out
}

// DataSources returns none, the Upjet based provider does not use them.
func (p *compositeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

type compositeData struct {
	legacy *lazyClient
	modern *lazyClient
}

type lazyClient struct {
	once     sync.Once
	provider provider.Provider
	ctx      context.Context
	req      provider.ConfigureRequest
	data     any
	diags    diag.Diagnostics
}

func newLazyClient(ctx context.Context, p provider.Provider, req provider.ConfigureRequest) *lazyClient {
	return &lazyClient{provider: p, ctx: ctx, req: req}
}

func (l *lazyClient) get() (any, diag.Diagnostics) {
	l.once.Do(func() {
		var resp provider.ConfigureResponse
		l.provider.Configure(l.ctx, l.req, &resp)
		l.data, l.diags = resp.ResourceData, resp.Diagnostics
	})
	return l.data, l.diags
}

func wrap(newResource func() resource.Resource, side func(*compositeData) *lazyClient) func() resource.Resource {
	return func() resource.Resource {
		return &compositeResource{Resource: newResource(), side: side}
	}
}

type compositeResource struct {
	resource.Resource
	side func(*compositeData) *lazyClient
}

var (
	_ resource.ResourceWithConfigure      = &compositeResource{}
	_ resource.ResourceWithImportState    = &compositeResource{}
	_ resource.ResourceWithModifyPlan     = &compositeResource{}
	_ resource.ResourceWithValidateConfig = &compositeResource{}
	_ resource.ResourceWithUpgradeState   = &compositeResource{}
)

func (r *compositeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	inner, ok := r.Resource.(resource.ResourceWithConfigure)
	if req.ProviderData == nil || !ok {
		return
	}
	data, ok := req.ProviderData.(*compositeData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected the composite provider data.")
		return
	}
	client, diags := r.side(data).get()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	req.ProviderData = client
	inner.Configure(ctx, req, resp)
}

func (r *compositeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if inner, ok := r.Resource.(resource.ResourceWithImportState); ok {
		inner.ImportState(ctx, req, resp)
		return
	}
	resp.Diagnostics.AddError("Resource Import Not Implemented", "This resource does not support import.")
}

func (r *compositeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if inner, ok := r.Resource.(resource.ResourceWithModifyPlan); ok {
		inner.ModifyPlan(ctx, req, resp)
	}
}

func (r *compositeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if inner, ok := r.Resource.(resource.ResourceWithValidateConfig); ok {
		inner.ValidateConfig(ctx, req, resp)
	}
}

func (r *compositeResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	if inner, ok := r.Resource.(resource.ResourceWithUpgradeState); ok {
		return inner.UpgradeState(ctx)
	}
	return nil
}
