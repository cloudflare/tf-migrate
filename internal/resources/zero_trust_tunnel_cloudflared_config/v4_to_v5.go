package zero_trust_tunnel_cloudflared_config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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
		{"keep_alive_timeout", int64(90), true}, // 90 seconds (1m30s)
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

// removeIncompleteAccessBlocks removes access blocks that don't have both aud_tag and team_name
// In v4, access could have just "required", but v5 requires aud_tag and team_name when access block exists
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

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)

	// Rename resource type if using deprecated name
	if resourceType == "cloudflare_tunnel_config" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_config", "cloudflare_zero_trust_tunnel_cloudflared_config")
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

	// Always use relative path within the instance JSON (not the full state path)
	path := "attributes"

	// 1. Transform config array [{}] → object {}
	result = state.TransformFieldArrayToObject(result, path, attrs, "config", state.ArrayToObjectOptions{})

	// Get the transformed config to work with (re-parse from root)
	resultParsed := gjson.Parse(result)
	attrs = resultParsed.Get(path)
	configObj := attrs.Get("config")

	if configObj.Exists() && configObj.IsObject() {
		configPath := path + ".config"

		// 2. Rename ingress_rule → ingress inside config
		result = state.RenameField(result, configPath, configObj, "ingress_rule", "ingress")

		// 3. Remove deprecated fields from config
		result = state.RemoveFields(result, configPath, configObj, "warp_routing")

		// Refresh config after changes
		configObj = gjson.Parse(result).Get(configPath)

		// 4. Transform config-level origin_request array → object (preserving nil fields)
		originReqPath := configPath + ".origin_request"
		result = state.TransformFieldArrayToObject(result, configPath, configObj, "origin_request", state.ArrayToObjectOptions{
			// Keep empty objects with nil fields as-is, don't convert to null
		})
		// Post-process: remove deprecated fields, convert durations, transform nested access
		configObj = gjson.Parse(result).Get(configPath)
		originReqField := configObj.Get("origin_request")
		if originReqField.Exists() && !originReqField.IsArray() {
			result = transformOriginRequestPostProcess(result, originReqPath, originReqField)
		}

		// 5. Transform ingress array elements' origin_request
		ingressArray := gjson.Parse(result).Get(configPath + ".ingress")
		if ingressArray.Exists() && ingressArray.IsArray() {
			for i, ingressItem := range ingressArray.Array() {
				originReq := ingressItem.Get("origin_request")
				if originReq.Exists() {
					ingressItemPath := fmt.Sprintf("%s.ingress.%d", configPath, i)
					ingressOriginReqPath := ingressItemPath + ".origin_request"

					// Transform array to object (preserving nil fields)
					result = state.TransformFieldArrayToObject(result, ingressItemPath, ingressItem, "origin_request", state.ArrayToObjectOptions{
						// Keep empty objects with nil fields as-is, don't convert to null
					})

					// Post-process
					ingressItem = gjson.Parse(result).Get(ingressItemPath)
					originReqField := ingressItem.Get("origin_request")
					if originReqField.Exists() && !originReqField.IsArray() {
						result = transformOriginRequestPostProcess(result, ingressOriginReqPath, originReqField)
					}
				}
			}
		}

	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Update the type field if it exists (for unit tests that pass instance-level type)
	if instance.Get("type").Exists() {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_tunnel_cloudflared_config")
	}

	return result, nil
}

// transformOriginRequestPostProcess performs post-processing on origin_request after array-to-object conversion
// This handles: removing deprecated fields, converting duration strings to nanoseconds, transforming nested access
func transformOriginRequestPostProcess(stateJSON string, path string, originReq gjson.Result) string {
	if !originReq.Exists() || !originReq.IsObject() {
		return stateJSON
	}

	// Remove deprecated fields
	stateJSON = state.RemoveFields(stateJSON, path, originReq, "bastion_mode", "proxy_address", "proxy_port", "ip_rules")

	// Refresh after removals
	originReq = gjson.Parse(stateJSON).Get(path)

	// Convert duration fields from strings to int64 seconds
	// v4 stored these as strings (e.g., "30s", "1m30s"), v5 expects integers (seconds)
	durationFields := []string{"connect_timeout", "tls_timeout", "tcp_keep_alive", "keep_alive_timeout"}
	for _, field := range durationFields {
		fieldValue := originReq.Get(field)
		if fieldValue.Exists() {
			// Use the general converter which handles both strings and numbers
			if converted := state.ConvertDurationToSeconds(fieldValue); converted != nil {
				stateJSON, _ = sjson.Set(stateJSON, path+"."+field, converted)
			}
		}
	}

	// Convert keep_alive_connections from int to int64
	keepAliveConns := originReq.Get("keep_alive_connections")
	if keepAliveConns.Exists() {
		stateJSON, _ = sjson.Set(stateJSON, path+".keep_alive_connections", state.ConvertToInt64(keepAliveConns))
	}

	// Refresh origin_request after duration conversions
	originReq = gjson.Parse(stateJSON).Get(path)

	// Transform nested access array → object
	accessField := originReq.Get("access")
	if accessField.Exists() {
		stateJSON = state.TransformFieldArrayToObject(stateJSON, path, originReq, "access", state.ArrayToObjectOptions{
			// Keep empty objects with nil fields as-is
		})

		// After transforming, check if access has required fields. If not, remove it.
		// Refresh after transformation
		originReq = gjson.Parse(stateJSON).Get(path)
		accessObj := originReq.Get("access")
		if accessObj.Exists() && accessObj.IsObject() {
			hasAudTag := accessObj.Get("aud_tag").Exists()
			hasTeamName := accessObj.Get("team_name").Exists()

			// If either required field is missing, remove the entire access block
			if !hasAudTag || !hasTeamName {
				stateJSON = state.RemoveFields(stateJSON, path, originReq, "access")
			}
		}
	}

	// Note: We do NOT delete empty origin_request objects with all nil fields
	// v5 expects to keep them as-is to match v4 state behavior

	return stateJSON
}
