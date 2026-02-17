package zero_trust_access_mtls_hostname_settings

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the resource name (same in v4 and v5)
	internal.RegisterMigrator("cloudflare_zero_trust_access_mtls_hostname_settings", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource was not renamed between v4 and v5.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_zero_trust_access_mtls_hostname_settings", "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert settings blocks to array attribute
	// v4: settings { hostname = "..." } (ListNestedBlock - multiple blocks)
	// v5: settings = [{ hostname = "..." }] (ListNestedAttribute - array of objects)
	tfhcl.ConvertBlocksToArrayAttribute(body, "settings", false)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op - state transformation is handled by the provider's StateUpgraders.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
