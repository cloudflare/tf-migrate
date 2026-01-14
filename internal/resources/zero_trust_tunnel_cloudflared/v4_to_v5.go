package zero_trust_tunnel_cloudflared

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (deprecated and preferred)
	internal.RegisterMigrator("cloudflare_tunnel", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_tunnel_cloudflared"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel" || resourceType == "cloudflare_zero_trust_tunnel"
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

	// Rename resource type based on which v4 name is used
	if resourceType == "cloudflare_tunnel" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel", "cloudflare_zero_trust_tunnel_cloudflared")
	} else if resourceType == "cloudflare_zero_trust_tunnel" {
		tfhcl.RenameResourceType(block, "cloudflare_zero_trust_tunnel", "cloudflare_zero_trust_tunnel_cloudflared")
	}

	// Rename attribute: secret → tunnel_secret
	body := block.Body()
	tfhcl.RenameAttribute(body, "secret", "tunnel_secret")

	// Add config_src = "local" if not present
	// In v4, this field didn't exist. In v5, it's Optional with no default, but the API default is "local"
	// We add it to config to prevent drift after migration
	if body.GetAttribute("config_src") == nil {
		tfhcl.SetAttributeValue(body, "config_src", "local")
	}

	// All other fields remain the same
	// Fields: account_id, name, tunnel_secret (renamed from secret), config_src

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

	// Rename field: secret → tunnel_secret
	result = state.RenameField(result, "attributes", attrs, "secret", "tunnel_secret")

	// Remove computed fields that don't exist in v5 or are deprecated
	// cname and tunnel_token were computed in v4 but don't exist in v5
	result = state.RemoveFields(result, "attributes", attrs, "cname", "tunnel_token")

	// Add config_src if not present (v5 Computed+Optional field defaults to "local")
	// This prevents plan drift when migrating from v4 where this field didn't exist
	refreshedAttrs := gjson.Parse(result).Get("attributes")
	if !refreshedAttrs.Get("config_src").Exists() {
		result, _ = sjson.Set(result, "attributes.config_src", "local")
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Update the type field if it exists (for unit tests that pass instance-level type)
	if instance.Get("type").Exists() {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_tunnel_cloudflared")
	}

	return result, nil
}
