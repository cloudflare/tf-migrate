package logpush_ownership_challenge

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name - same as v5 in this case
	internal.RegisterMigrator("cloudflare_logpush_ownership_challenge", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_logpush_ownership_challenge"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_logpush_ownership_challenge"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - config stays identical
	return content
}

// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_logpush_ownership_challenge"}, "cloudflare_logpush_ownership_challenge"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// NO TRANSFORMATIONS NEEDED
	// The HCL configuration is identical in v4 and v5:
	// - Same resource name
	// - Same field names
	// - Same field types
	// - Same validation behavior

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op for logpush_ownership_challenge migration.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
//
// Provider StateUpgraders handle:
// - Direct copies: account_id, zone_id, destination_conf
// - Remove: ownership_challenge_filename (v4 computed field)
// - Set to Null: filename, message, valid (new v5 computed fields, API populates on create)
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// This function is a no-op for logpush_ownership_challenge migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider handles state transformation, not tf-migrate.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
