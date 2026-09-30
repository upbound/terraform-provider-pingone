package upjet

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestCompositeResources(t *testing.T) {
	ctx := context.Background()
	p := Composite("test")

	want := len(FrameworkLegacySDK("test").Resources(ctx)) + len(Framework("test").Resources(ctx))
	resources := p.Resources(ctx)
	if len(resources) != want {
		t.Fatalf("want %d resources, got %d", want, len(resources))
	}

	seen := map[string]bool{}
	for _, newResource := range resources {
		var resp resource.MetadataResponse
		newResource().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pingone"}, &resp)
		if seen[resp.TypeName] {
			t.Errorf("duplicate resource type %s", resp.TypeName)
		}
		seen[resp.TypeName] = true
	}
	if !seen["pingone_davinci_flow"] || !seen["pingone_language_translation"] {
		t.Errorf("missing expected resource types in %v", seen)
	}
}
