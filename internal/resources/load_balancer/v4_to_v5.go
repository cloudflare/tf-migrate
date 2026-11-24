package load_balancer

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_load_balancer"}, "cloudflare_load_balancer"
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

	// Transform rules blocks to list attribute.
	// v4: repeated rules { ... overrides { ... } fixed_response { ... } } blocks
	// v5: rules = [{ ... overrides = { ... } fixed_response = { ... } }, ...]
	//
	// Each rule's overrides sub-block contains the same block→attribute patterns
	// as the top-level LB resource, so we apply the same transformations recursively.
	//
	// dynamic "rules" blocks become for expressions. Rules are evaluated in
	// order, so when there are several rules blocks, static or dynamic, the
	// list is a concat() of them in source order.
	if tfhcl.HasDynamicBlockFor(body, "rules") {
		if reason := m.convertRulesWithDynamic(body); reason != "" {
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary: fmt.Sprintf(
					"dynamic \"rules\" not migrated on cloudflare_load_balancer.%s", tfhcl.GetResourceName(block)),
				Detail: fmt.Sprintf("The rules blocks could not be converted automatically: %s.\n\n"+
					"They have been left in place, but v5 expects a list attribute, so the "+
					"configuration will not validate until they are rewritten by hand, for example:\n\n"+
					"  rules = [for k, v in <for_each> : {\n"+
					"    name      = k\n"+
					"    condition = \"...\"\n"+
					"    overrides = { steering_policy = \"off\" }\n"+
					"  }]", reason),
			})
		}
	} else {
		tfhcl.ConvertBlocksToAttributeList(body, "rules", m.transformRuleBlock)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// convertRulesWithDynamic converts rules blocks when at least one of them is
// a dynamic block. It returns a reason, and leaves the blocks in place, when
// a block can't be converted.
func (m *V4ToV5Migrator) convertRulesWithDynamic(body *hclwrite.Body) string {
	type segment struct {
		static  []*hclwrite.Block
		dynamic *hclwrite.Block
	}
	var segments []*segment
	var all []*hclwrite.Block
	for _, b := range body.Blocks() {
		switch {
		case b.Type() == "rules":
			if len(segments) == 0 || segments[len(segments)-1].dynamic != nil {
				segments = append(segments, &segment{})
			}
			segments[len(segments)-1].static = append(segments[len(segments)-1].static, b)
		case b.Type() == "dynamic" && len(b.Labels()) > 0 && b.Labels()[0] == "rules":
			segments = append(segments, &segment{dynamic: b})
		default:
			continue
		}
		all = append(all, b)
	}

	// Check every dynamic block before changing anything.
	for _, seg := range segments {
		if seg.dynamic == nil {
			continue
		}
		dynBody := seg.dynamic.Body()
		if dynBody.GetAttribute("for_each") == nil {
			return "a dynamic block has no for_each"
		}
		content := tfhcl.FindBlockByType(dynBody, "content")
		if content == nil {
			return "a dynamic block has no content block"
		}
		for _, nested := range content.Body().Blocks() {
			if nested.Type() == "dynamic" {
				return "a rule's content has a nested dynamic block"
			}
		}
	}

	var lists []hclwrite.Tokens
	for _, seg := range segments {
		if seg.dynamic == nil {
			var elems []hclwrite.Tokens
			for _, rb := range seg.static {
				m.transformRuleBlock(rb)
				elems = append(elems, tfhcl.BuildObjectFromBlock(rb))
			}
			lists = append(lists, hclwrite.TokensForTuple(elems))
			continue
		}
		m.transformRuleBlock(tfhcl.FindBlockByType(seg.dynamic.Body(), "content"))
		tokens, ok, reason := tfhcl.BuildDynamicForListTokens(seg.dynamic, "rules")
		if !ok {
			return reason
		}
		lists = append(lists, tokens)
	}

	value := lists[0]
	if len(lists) > 1 {
		value = tfhcl.BuildConcatExpression(lists)
	}
	for _, b := range all {
		body.RemoveBlock(b)
	}
	body.SetAttributeRaw("rules", value)
	return ""
}

// transformRuleBlock applies all necessary transformations to a single rules block
// before it is converted to an object in the rules list attribute.
func (m *V4ToV5Migrator) transformRuleBlock(ruleBlock *hclwrite.Block) {
	ruleBody := ruleBlock.Body()

	// Transform fixed_response block (MaxItems:1) → single object attribute
	// v4: fixed_response { message_body = "..." status_code = 200 ... }
	// v5: fixed_response = { message_body = "..." status_code = 200 ... }
	tfhcl.ConvertSingleBlockToAttribute(ruleBody, "fixed_response", "fixed_response")

	// Transform overrides block (TypeList, MaxItems:1 in practice) → single object attribute.
	// The overrides body itself contains the same block→attribute patterns as the top-level LB.
	// We must transform those nested blocks before converting overrides to an attribute.
	overridesBlocks := tfhcl.FindBlocksByType(ruleBody, "overrides")
	for _, overridesBlock := range overridesBlocks {
		m.transformOverridesBlock(overridesBlock)
	}

	// Now convert the overrides block(s) to a single object attribute
	// v4: overrides { ... }
	// v5: overrides = { ... }
	tfhcl.ConvertSingleBlockToAttribute(ruleBody, "overrides", "overrides")
}

// transformOverridesBlock applies all block→attribute transformations inside
// a rules[].overrides block, mirroring the top-level LB transformations.
func (m *V4ToV5Migrator) transformOverridesBlock(overridesBlock *hclwrite.Block) {
	overridesBody := overridesBlock.Body()

	// session_affinity_attributes block → single object attribute
	tfhcl.ConvertSingleBlockToAttribute(overridesBody, "session_affinity_attributes", "session_affinity_attributes")

	// adaptive_routing block → single object attribute
	tfhcl.ConvertSingleBlockToAttribute(overridesBody, "adaptive_routing", "adaptive_routing")

	// location_strategy block → single object attribute
	tfhcl.ConvertSingleBlockToAttribute(overridesBody, "location_strategy", "location_strategy")

	// random_steering block → single object attribute
	tfhcl.ConvertSingleBlockToAttribute(overridesBody, "random_steering", "random_steering")

	// region_pools / pop_pools / country_pools blocks → map attributes
	m.transformPoolsBlocks(overridesBody, "region_pools", "region")
	m.transformPoolsBlocks(overridesBody, "pop_pools", "pop")
	m.transformPoolsBlocks(overridesBody, "country_pools", "country")
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
