package zero_trust_tunnel_cloudflared_config

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared config resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (deprecated and preferred)
	internal.RegisterMigrator("cloudflare_tunnel_config", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel_cloudflared_config", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name (same as preferred v4 name)
	return "cloudflare_zero_trust_tunnel_cloudflared_config"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel_config" || resourceType == "cloudflare_zero_trust_tunnel_cloudflared_config"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - HCL parser can handle all transformations
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tunnel_config", "cloudflare_zero_trust_tunnel_cloudflared_config"
}

// addOriginRequestDefaults adds v4 default values to origin_request block for fields not already specified
// This preserves v4 behavior where the API applied defaults that v5 does not apply
// Also converts any existing string duration values to integer seconds
func addOriginRequestDefaults(originReqBody *hclwrite.Body) {
	// Use a slice to ensure deterministic ordering of defaults
	type defaultPair struct {
		field      string
		value      interface{}
		isDuration bool
	}

	// v4 API defaults that need to be explicitly specified in v5
	// Duration fields are in seconds (Int64)
	// Ordered alphabetically for consistency
	v4Defaults := []defaultPair{
		{"ca_pool", "", false},
		{"connect_timeout", int64(30), true}, // 30 seconds
		{"disable_chunked_encoding", false, false},
		{"http2_origin", false, false},
		{"keep_alive_connections", int64(100), false},
		{"keep_alive_timeout", int64(90), true}, // 90 seconds (1m30s)
		{"no_happy_eyeballs", false, false},
		{"no_tls_verify", false, false},
		{"origin_server_name", "", false},
		{"proxy_type", "", false},
		{"tcp_keep_alive", int64(30), true}, // 30 seconds
		{"tls_timeout", int64(10), true},    // 10 seconds
	}

	for _, pair := range v4Defaults {
		existingAttr := originReqBody.GetAttribute(pair.field)
		if existingAttr == nil {
			// Field not specified, add default
			tfhcl.SetAttributeValue(originReqBody, pair.field, pair.value)
		} else if pair.isDuration {
			// Duration field exists - try to convert string duration to integer seconds
			if seconds, ok := tryConvertDurationAttribute(existingAttr); ok {
				tfhcl.SetAttributeValue(originReqBody, pair.field, seconds)
			}
			// If conversion fails, leave as-is (may already be an integer)
		}
	}
}

// removeIncompleteAccessBlocks removes access blocks that don't have both aud_tag and team_name.
// In v4, access could have just "required", but v5 requires aud_tag and team_name when access block exists.
//
// Dropping the block is safe because aud_tag and team_name identify which Access app to validate against.
// Without them, the API receives empty values and does not meaningfully enforce access regardless of the
// required field value:
//   - required = false (or omitted): soft enforcement — JWT is checked but traffic is not denied on failure.
//     Dropping the block has no practical security impact.
//   - required = true: hard enforcement is intended, but without aud_tag/team_name the Access app is
//     unidentified and enforcement is ineffective in v4 as well. Dropping preserves that behavior.
func removeIncompleteAccessBlocks(originReqBody *hclwrite.Body) {
	accessBlocks := tfhcl.FindBlocksByType(originReqBody, "access")
	for _, accessBlock := range accessBlocks {
		accessBody := accessBlock.Body()
		hasAudTag := accessBody.GetAttribute("aud_tag") != nil
		hasTeamName := accessBody.GetAttribute("team_name") != nil

		// If either aud_tag or team_name is missing, remove the entire access block
		if !hasAudTag || !hasTeamName {
			tfhcl.RemoveBlocksByType(originReqBody, "access")
			break // Only one access block allowed (MaxItems:1)
		}
	}
}

// tryConvertDurationAttribute attempts to parse a duration string attribute and convert it to integer seconds
// Returns (seconds, true) if successful, (0, false) if the attribute is not a string duration or parsing fails
func tryConvertDurationAttribute(attr *hclwrite.Attribute) (int64, bool) {
	// Get all tokens for the expression
	tokens := attr.Expr().BuildTokens(nil)
	if len(tokens) == 0 {
		return 0, false
	}

	// Concatenate all token bytes to get the full expression string
	var fullExpr strings.Builder
	for _, tok := range tokens {
		fullExpr.Write(tok.Bytes)
	}
	exprStr := fullExpr.String()

	// Check if it's a quoted string literal
	exprStr = strings.TrimSpace(exprStr)
	if len(exprStr) < 3 || exprStr[0] != '"' || exprStr[len(exprStr)-1] != '"' {
		return 0, false // Not a string literal
	}

	// Extract the string content (remove quotes)
	durationStr := exprStr[1 : len(exprStr)-1]

	// Try to parse as a duration string
	seconds, err := state.ParseDurationStringToSeconds(durationStr)
	if err != nil {
		return 0, false
	}

	return seconds, true
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// When the deprecated name is used, rename the resource type AND generate a moved block.
	// The moved block tells Terraform to invoke the provider's MoveState hook, which reads the
	// old cloudflare_tunnel_config state using SourceV4TunnelConfigSchema and transforms it.
	var movedBlock *hclwrite.Block
	if resourceType == "cloudflare_tunnel_config" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_config", "cloudflare_zero_trust_tunnel_cloudflared_config")
		from := "cloudflare_tunnel_config." + resourceName
		to := "cloudflare_zero_trust_tunnel_cloudflared_config." + resourceName
		movedBlock = tfhcl.CreateMovedBlock(from, to)
	}

	body := block.Body()

	// First, process config block to remove deprecated fields before converting to attribute
	configBlocks := tfhcl.FindBlocksByType(body, "config")
	for _, configBlock := range configBlocks {
		configBody := configBlock.Body()

		// Remove deprecated blocks that were removed in v5
		tfhcl.RemoveBlocksByType(configBody, "warp_routing")

		// Remove ip_rules from all origin_request blocks (at config level)
		originReqBlocks := tfhcl.FindBlocksByType(configBody, "origin_request")
		for _, originReqBlock := range originReqBlocks {
			originReqBody := originReqBlock.Body()
			tfhcl.RemoveBlocksByType(originReqBody, "ip_rules")

			// Remove deprecated attributes
			tfhcl.RemoveAttributes(originReqBody, "bastion_mode", "proxy_address", "proxy_port")

			// Remove access blocks that don't have required fields (aud_tag and team_name)
			// In v4, access could have just "required=false", but v5 requires all fields
			removeIncompleteAccessBlocks(originReqBody)

			// Add v4 defaults for fields not specified to preserve v4 behavior
			// v4 API applied these defaults, v5 does not, so we must specify them explicitly
			addOriginRequestDefaults(originReqBody)
		}

		// Remove ip_rules from nested origin_request blocks within ingress_rule
		ingressBlocks := tfhcl.FindBlocksByType(configBody, "ingress_rule")
		for _, ingressBlock := range ingressBlocks {
			nestedOriginReqBlocks := tfhcl.FindBlocksByType(ingressBlock.Body(), "origin_request")
			for _, nestedOriginReqBlock := range nestedOriginReqBlocks {
				nestedOriginReqBody := nestedOriginReqBlock.Body()
				tfhcl.RemoveBlocksByType(nestedOriginReqBody, "ip_rules")

				// Remove deprecated attributes
				tfhcl.RemoveAttributes(nestedOriginReqBody, "bastion_mode", "proxy_address", "proxy_port")

				// Remove incomplete access blocks here too
				removeIncompleteAccessBlocks(nestedOriginReqBody)

				// Add v4 defaults for nested origin_request too
				addOriginRequestDefaults(nestedOriginReqBody)
			}
		}
	}

	// Now convert config block syntax to attribute syntax
	// v4: config { } → v5: config = { }
	// This needs to handle nested structures recursively
	if len(configBlocks) > 0 {
		// Define which blocks should always be arrays (ingress, even with 1 element)
		alwaysArrayFields := map[string]bool{
			"ingress":      true, // ingress is always an array in v5
			"ingress_rule": true, // ingress_rule gets renamed to ingress (array)
		}

		// First, rename ingress_rule blocks to ingress before conversion
		for _, configBlock := range configBlocks {
			configBody := configBlock.Body()
			ingressRuleBlocks := tfhcl.FindBlocksByType(configBody, "ingress_rule")
			for _, ingressBlock := range ingressRuleBlocks {
				// Change the block type by creating a new block with the correct type
				newIngressBlock := hclwrite.NewBlock("ingress", nil)
				// Copy all attributes in alphabetical order for deterministic output
				attrNames := make([]string, 0, len(ingressBlock.Body().Attributes()))
				attrs := ingressBlock.Body().Attributes()
				for name := range attrs {
					attrNames = append(attrNames, name)
				}
				// Sort to ensure deterministic ordering
				sort.Strings(attrNames)
				for _, name := range attrNames {
					newIngressBlock.Body().SetAttributeRaw(name, attrs[name].Expr().BuildTokens(nil))
				}
				// Copy nested blocks
				for _, nestedBlock := range ingressBlock.Body().Blocks() {
					newIngressBlock.Body().AppendBlock(nestedBlock)
				}
				configBody.AppendBlock(newIngressBlock)
			}
			// Remove the old ingress_rule blocks
			tfhcl.RemoveBlocksByType(configBody, "ingress_rule")
		}

		// Now convert config block to attribute with all nested blocks
		tfhcl.ConvertBlockToAttributeWithNestedAndArrays(body, "config", alwaysArrayFields)
	}

	resultBlocks := []*hclwrite.Block{block}
	if movedBlock != nil {
		resultBlocks = append(resultBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         resultBlocks,
		RemoveOriginal: movedBlock != nil,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// State migration is now handled by the provider's UpgradeState mechanism.
	// The v5 provider transforms v4 state directly when loading it, so tf-migrate
	// no longer needs to perform state transformation.
	return instance.String(), nil
}
