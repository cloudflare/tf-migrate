package authenticated_origin_pulls

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Authenticated Origin Pulls from v4 to v5
// The v4 cloudflare_authenticated_origin_pulls resource is renamed to cloudflare_authenticated_origin_pulls_settings in v5
// and simplified to only handle zone-wide authenticated origin pulls settings.
//
// v4 fields removed in v5:
// - hostname: Per-hostname AOP moved to separate resource/functionality
// - authenticated_origin_pulls_certificate: Certificate management moved to separate resource
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_authenticated_origin_pulls", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_authenticated_origin_pulls_settings"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_authenticated_origin_pulls"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for authenticated_origin_pulls migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_authenticated_origin_pulls", "cloudflare_authenticated_origin_pulls_settings"
}

// TransformConfig handles configuration file transformations.
// Transformations:
// - Rename resource type: cloudflare_authenticated_origin_pulls → cloudflare_authenticated_origin_pulls_settings
// - Remove hostname attribute (if present)
// - Remove authenticated_origin_pulls_certificate attribute (if present)
// - Generate moved block for Terraform 1.8+ automatic state migration
// - Keep: zone_id, enabled (both exist in v5 with same semantics)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Get the resource name before renaming
	resourceName := block.Labels()[1]

	// Rename the resource type
	tfhcl.RenameResourceType(block, "cloudflare_authenticated_origin_pulls", "cloudflare_authenticated_origin_pulls_settings")

	// Remove fields that don't exist in v5
	tfhcl.RemoveAttributes(body, "hostname", "authenticated_origin_pulls_certificate")

	// Generate moved block for Terraform 1.8+ automatic state migration
	// This allows users on Terraform 1.8+ to automatically migrate state without manual `terraform state mv`
	fromRef := "cloudflare_authenticated_origin_pulls." + resourceName
	toRef := "cloudflare_authenticated_origin_pulls_settings." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(fromRef, toRef)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// TransformState is disabled - state transformation is handled by provider StateUpgraders.
// This method is a no-op and returns the state unchanged.
// Users should use `terraform state mv` or Terraform 1.8+ `moved` blocks for resource renaming,
// which will trigger the provider's StateUpgrader to handle the schema transformation.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// Return state unchanged - provider StateUpgraders will handle transformation
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
