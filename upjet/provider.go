// Package upjet exposes the internal PingOne providers to the Upjet based
// Crossplane provider, which cannot import internal packages.
package upjet

import (
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/pingidentity/terraform-provider-pingone/internal/provider/framework"
	"github.com/pingidentity/terraform-provider-pingone/internal/provider/frameworklegacysdk"
	"github.com/pingidentity/terraform-provider-pingone/internal/provider/sdkv2"
)

// SDKv2 returns the Terraform Plugin SDKv2 provider.
func SDKv2(version string) *schema.Provider {
	return sdkv2.New(version)()
}

// Framework returns the Terraform Plugin Framework provider.
func Framework(version string) provider.Provider {
	return framework.New(version)()
}

// FrameworkLegacySDK returns the Terraform Plugin Framework provider whose
// resources are backed by the legacy PingOne SDK.
func FrameworkLegacySDK(version string) provider.Provider {
	return frameworklegacysdk.New(version)()
}
