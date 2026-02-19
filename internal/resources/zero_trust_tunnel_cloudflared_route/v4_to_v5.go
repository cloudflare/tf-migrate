package zero_trust_tunnel_cloudflared_route

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared route resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (deprecated and preferred)
	internal.RegisterMigrator("cloudflare_tunnel_route", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel_route", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_tunnel_cloudflared_route"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel_route" || resourceType == "cloudflare_zero_trust_tunnel_route"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - HCL parser can handle all transformations
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tunnel_route", "cloudflare_zero_trust_tunnel_cloudflared_route"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Capture original type before renaming, then rename
	fromType := resourceType
	if resourceType == "cloudflare_tunnel_route" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_route", "cloudflare_zero_trust_tunnel_cloudflared_route")
	} else if resourceType == "cloudflare_zero_trust_tunnel_route" {
		tfhcl.RenameResourceType(block, "cloudflare_zero_trust_tunnel_route", "cloudflare_zero_trust_tunnel_cloudflared_route")
	}

	// All fields remain the same - no field renames or transformations needed
	// Fields: account_id, tunnel_id, network, comment, virtual_network_id

	// Generate moved block so Terraform knows to move state from the old resource type
	from := fromType + "." + resourceName
	to := "cloudflare_zero_trust_tunnel_cloudflared_route." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
	// The moved block generated in TransformConfig triggers the provider's migration logic.
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
