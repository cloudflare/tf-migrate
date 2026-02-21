package load_balancer

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of load balancer resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_load_balancer", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (unchanged from v4)
	return "cloudflare_load_balancer"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_load_balancer doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_load_balancer", "cloudflare_load_balancer"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Rename attributes (v4 → v5)
	// v4: default_pool_ids → v5: default_pools
	// v4: fallback_pool_id → v5: fallback_pool
	tfhcl.RenameAttribute(body, "default_pool_ids", "default_pools")
	tfhcl.RenameAttribute(body, "fallback_pool_id", "fallback_pool")

	// Transform session_affinity_attributes block to map attribute
	// v4: session_affinity_attributes { samesite = "Lax" secure = "Always" }
	// v5: session_affinity_attributes = { samesite = "Lax" secure = "Always" }
	tfhcl.ConvertSingleBlockToAttribute(body, "session_affinity_attributes", "session_affinity_attributes")

	// Transform adaptive_routing block to single object attribute
	// v4: adaptive_routing { failover_across_pools = false }
	// v5: adaptive_routing = { failover_across_pools = false }
	tfhcl.ConvertSingleBlockToAttribute(body, "adaptive_routing", "adaptive_routing")

	// Transform location_strategy block to single object attribute
	// v4: location_strategy { prefer_ecs = "proximity" mode = "pop" }
	// v5: location_strategy = { prefer_ecs = "proximity" mode = "pop" }
	tfhcl.ConvertSingleBlockToAttribute(body, "location_strategy", "location_strategy")

	// Transform random_steering block to single object attribute
	// v4: random_steering { default_weight = 0.5 }
	// v5: random_steering = { default_weight = 0.5 }
	tfhcl.ConvertSingleBlockToAttribute(body, "random_steering", "random_steering")

	// Transform region_pools blocks to map attribute
	// v4: region_pools { region = "WNAM" pool_ids = [...] }
	// v5: region_pools = { "WNAM" = [...] }
	m.transformPoolsBlocks(body, "region_pools", "region")

	// Transform pop_pools blocks to map attribute
	// v4: pop_pools { pop = "LAX" pool_ids = [...] }
	// v5: pop_pools = { "LAX" = [...] }
	m.transformPoolsBlocks(body, "pop_pools", "pop")

	// Transform country_pools blocks to map attribute
	// v4: country_pools { country = "US" pool_ids = [...] }
	// v5: country_pools = { "US" = [...] }
	m.transformPoolsBlocks(body, "country_pools", "country")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformPoolsBlocks converts region/pop/country_pools blocks to map attributes
// v4: region_pools { region = "WNAM" pool_ids = [...] }
// v5: region_pools = { "WNAM" = [...] }
func (m *V4ToV5Migrator) transformPoolsBlocks(body *hclwrite.Body, blockName, keyAttrName string) {
	blocks := tfhcl.FindBlocksByType(body, blockName)
	if len(blocks) == 0 {
		return
	}

	// Build a list of object attributes for the map
	var mapAttrs []hclwrite.ObjectAttrTokens

	for _, block := range blocks {
		blockBody := block.Body()

		// Get the key attribute (region, pop, or country)
		keyAttr := blockBody.GetAttribute(keyAttrName)
		if keyAttr == nil {
			continue
		}

		// Get pool_ids attribute
		poolIDsAttr := blockBody.GetAttribute("pool_ids")
		if poolIDsAttr == nil {
			continue
		}

		// Use the key as the map key and pool_ids as the map value
		keyTokens := keyAttr.Expr().BuildTokens(nil)
		valueTokens := poolIDsAttr.Expr().BuildTokens(nil)

		mapAttrs = append(mapAttrs, hclwrite.ObjectAttrTokens{
			Name:  keyTokens,
			Value: valueTokens,
		})
	}

	// Remove all blocks
	tfhcl.RemoveBlocksByType(body, blockName)

	// Create map attribute if we have any attributes
	if len(mapAttrs) > 0 {
		tokens := hclwrite.TokensForObject(mapAttrs)
		body.SetAttributeRaw(blockName, tokens)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// The provider will automatically upgrade state when it detects v4 schema_version=1 or v5 version=1
	// This function is a no-op for load_balancer migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
