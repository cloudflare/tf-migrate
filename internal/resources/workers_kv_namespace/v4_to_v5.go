package workers_kv_namespace

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_workers_kv_namespace", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_kv_namespace"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_kv_namespace"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For workers_kv_namespace, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_workers_kv_namespace", "cloudflare_workers_kv_namespace"
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For workers_kv_namespace, the config is identical between v4 and v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - config is identical
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is now delegated to provider StateUpgraders
// This function is a no-op as the provider handles all state transformations
// The provider's StateUpgraders will:
// 1. Read v4 state using v4 schema definition
// 2. Transform to v5 state (pass-through for this resource)
// 3. Provider will populate supports_url_encoding on first refresh
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// Pass through state unchanged - provider StateUpgraders handle migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
// This tells tf-migrate that the provider handles state transformation via StateUpgraders
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
