package zero_trust_tunnel_cloudflared

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names:
	//   - cloudflare_tunnel: the original v4 name (requires type rename + moved block)
	//   - cloudflare_zero_trust_tunnel_cloudflared: the preferred v4 name (in-place attr updates only)
	internal.RegisterMigrator("cloudflare_tunnel", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel_cloudflared", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_tunnel_cloudflared"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel" || resourceType == "cloudflare_zero_trust_tunnel_cloudflared"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - HCL parser can handle all transformations
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tunnel", "cloudflare_zero_trust_tunnel_cloudflared"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	body := block.Body()

	if resourceType == "cloudflare_zero_trust_tunnel_cloudflared" {
		// Already the correct resource type in v4 — just update attributes in-place.
		// No moved block needed since the type name doesn't change.
		tfhcl.RenameAttribute(body, "secret", "tunnel_secret")
		if body.GetAttribute("config_src") == nil {
			tfhcl.SetAttributeValue(body, "config_src", "local")
		}
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// resourceType == "cloudflare_tunnel": rename type and generate moved block.
	tfhcl.RenameResourceType(block, "cloudflare_tunnel", "cloudflare_zero_trust_tunnel_cloudflared")

	// Rename attribute: secret → tunnel_secret
	tfhcl.RenameAttribute(body, "secret", "tunnel_secret")

	// Add config_src = "local" if not present
	// In v4, this field didn't exist. In v5, it's Optional with no default, but the API default is "local"
	// We add it to config to prevent drift after migration
	if body.GetAttribute("config_src") == nil {
		tfhcl.SetAttributeValue(body, "config_src", "local")
	}

	// Generate moved block using the original (pre-rename) resource type.
	// This allows Terraform to use the provider's MoveState hook with SourceSchema to decode
	// the old state entry, eliminating the need for tf-migrate to rename the type in state.
	from := "cloudflare_tunnel." + resourceName
	to := "cloudflare_zero_trust_tunnel_cloudflared." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op: state migration is handled entirely by the provider's
// MoveState and UpgradeState hooks (triggered by the moved block in TransformConfig).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
