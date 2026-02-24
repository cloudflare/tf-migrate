package zero_trust_device_managed_networks

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of device managed networks resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	internal.RegisterMigrator("cloudflare_device_managed_networks", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_device_managed_networks", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_managed_networks"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_managed_networks" || resourceType == "cloudflare_zero_trust_device_managed_networks"
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from cloudflare_device_managed_networks to cloudflare_zero_trust_device_managed_networks
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_managed_networks", "cloudflare_zero_trust_device_managed_networks"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Get the resource name before renaming (for moved block generation)
	resourceName := tfhcl.GetResourceName(block)

	// Rename cloudflare_device_managed_networks to cloudflare_zero_trust_device_managed_networks
	tfhcl.RenameResourceType(block, "cloudflare_device_managed_networks", "cloudflare_zero_trust_device_managed_networks")

	body := block.Body()

	if configBlock := tfhcl.FindBlockByType(body, "config"); configBlock != nil {
		tfhcl.ConvertBlocksToAttribute(body, "config", "config", func(configBlock *hclwrite.Block) {})
	}

	// Generate moved block for state migration
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op for this resource - state transformation is handled by the provider's StateUpgraders.
// tf-migrate only transforms configs and generates moved blocks.
// The provider's MoveState and UpgradeState handlers will automatically transform the state when Terraform runs.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// No-op: Return state unchanged
	// Provider StateUpgraders handle:
	// 1. MoveState: cloudflare_device_managed_networks → cloudflare_zero_trust_device_managed_networks
	// 2. UpgradeState: config array → config object, network_id population
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
