package access_rule

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
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_access_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_access_rule"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_access_rule"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_rule", "cloudflare_access_rule"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For access_rule, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert configuration block to attribute (MaxItems:1 → SingleNestedAttribute)
	// v4: configuration { target = "ip" value = "1.2.3.4" }
	// v5: configuration = { target = "ip" value = "1.2.3.4" }
	tfhcl.ConvertBlocksToAttribute(body, "configuration", "configuration", func(block *hclwrite.Block) {})

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op for access_rule migration.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
// The provider will migrate:
// - configuration: array[0] → object (v4 SDKv2 TypeList MaxItems:1 → v5 SingleNestedAttribute)
// - schema_version: 1 → 500 (with controlled rollout via TF_MIG_TEST flag)
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider handles state transformation, not tf-migrate.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
