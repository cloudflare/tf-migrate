package zero_trust_tunnel_cloudflared_virtual_network

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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

	// Rename resource type based on which v4 name is used
	if resourceType == "cloudflare_tunnel_virtual_network" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_virtual_network", "cloudflare_zero_trust_tunnel_cloudflared_virtual_network")
	} else if resourceType == "cloudflare_zero_trust_tunnel_virtual_network" {
		tfhcl.RenameResourceType(block, "cloudflare_zero_trust_tunnel_virtual_network", "cloudflare_zero_trust_tunnel_cloudflared_virtual_network")
	}

	// No field renames needed for virtual_network
	// All fields remain the same: account_id, name, is_default_network, comment

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Add defaults for optional fields with v5 defaults
	// This prevents v5 provider from triggering PATCH operations when fields are null

	// comment has default "" (empty string) in v5
	if !attrs.Get("comment").Exists() || attrs.Get("comment").Type == gjson.Null {
		result, _ = sjson.Set(result, "attributes.comment", "")
	}

	// is_default_network has default false in v5
	if !attrs.Get("is_default_network").Exists() || attrs.Get("is_default_network").Type == gjson.Null {
		result, _ = sjson.Set(result, "attributes.is_default_network", false)
	}

	// No computed fields from v4 to remove
	// v5 adds new computed fields (id, created_at, deleted_at) but provider will generate these

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Update the type field if it exists (for unit tests that pass instance-level type)
	if instance.Get("type").Exists() {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_tunnel_cloudflared_virtual_network")
	}

	return result, nil
}
