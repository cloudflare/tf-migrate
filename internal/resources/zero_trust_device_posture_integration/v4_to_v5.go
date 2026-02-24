package zero_trust_device_posture_integration

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of device posture integration resources from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register BOTH v4 resource names (deprecated and current)
	internal.RegisterMigrator("cloudflare_device_posture_integration", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_device_posture_integration", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_posture_integration"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_posture_integration" ||
		resourceType == "cloudflare_zero_trust_device_posture_integration"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done at HCL level
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from cloudflare_device_posture_integration to cloudflare_zero_trust_device_posture_integration
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_posture_integration", "cloudflare_zero_trust_device_posture_integration"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceName := tfhcl.GetResourceName(block)

	// Rename both possible v4 resource names to v5 name
	tfhcl.RenameResourceType(block, "cloudflare_device_posture_integration", "cloudflare_zero_trust_device_posture_integration")

	body := block.Body()

	// Ensure interval is present (required in v5, optional in v4)
	// Default to "24h" if missing
	if !tfhcl.HasAttribute(body, "interval") {
		tfhcl.EnsureAttribute(body, "interval", "24h")
	}

	// Remove deprecated identifier field
	tfhcl.RemoveAttributes(body, "identifier")

	// Convert config block to attribute syntax
	// v4: config { ... }  (block syntax)
	// v5: config = { ... } (attribute syntax)
	if configBlock := tfhcl.FindBlockByType(body, "config"); configBlock != nil {
		tfhcl.ConvertBlocksToAttribute(body, "config", "config", func(cb *hclwrite.Block) {
			// No nested transformations needed - config fields remain the same
		})
	}

	// If config doesn't exist at all, add empty config (required in v5)
	if !tfhcl.HasAttribute(body, "config") && tfhcl.FindBlockByType(body, "config") == nil {
		// Create empty config object
		tfhcl.EnsureAttribute(body, "config", map[string]any{})
	}

	// Generate moved block for resource rename
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState)
	// The moved block generated in TransformConfig triggers the provider's migration logic
	// This function is a no-op for zero_trust_device_posture_integration migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
