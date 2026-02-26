package zero_trust_tunnel_cloudflared_virtual_network

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared virtual network resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (deprecated and preferred)
	internal.RegisterMigrator("cloudflare_tunnel_virtual_network", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel_virtual_network", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_tunnel_cloudflared_virtual_network"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel_virtual_network" ||
		resourceType == "cloudflare_zero_trust_tunnel_virtual_network"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - HCL parser can handle all transformations
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tunnel_virtual_network", "cloudflare_zero_trust_tunnel_cloudflared_virtual_network"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Generate moved block using the original (pre-rename) resource type.
	// This allows Terraform to use the provider's MoveState hook with SourceSchema to decode
	// the old state entry, eliminating the need for tf-migrate to rename the type in state.
	from := resourceType + "." + resourceName
	to := "cloudflare_zero_trust_tunnel_cloudflared_virtual_network." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	// Rename resource type based on which v4 name is used
	if resourceType == "cloudflare_tunnel_virtual_network" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_virtual_network", "cloudflare_zero_trust_tunnel_cloudflared_virtual_network")
	} else if resourceType == "cloudflare_zero_trust_tunnel_virtual_network" {
		tfhcl.RenameResourceType(block, "cloudflare_zero_trust_tunnel_virtual_network", "cloudflare_zero_trust_tunnel_cloudflared_virtual_network")
	}

	// No field renames needed for virtual_network
	// All fields remain the same: account_id, name, is_default_network, comment

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// UsesProviderStateUpgrader indicates that the provider's StateUpgrader handles all
// state migration for this resource. TransformState is a no-op.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// State migration is handled by the provider's StateUpgrader (MoveState + UpgradeState).
	// tf-migrate passes state through unchanged.
	return instance.String(), nil
}
