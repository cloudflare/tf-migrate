package zero_trust_access_application

import (
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// v4 had both cloudflare_access_application and cloudflare_zero_trust_access_application
	internal.RegisterMigrator("cloudflare_access_application", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_application", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_access_application" ||
		resourceType == "cloudflare_zero_trust_access_application"
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_application", "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Get the resource name before renaming (for moved block generation)
	resourceName := tfhcl.GetResourceName(block)
	resourceType := tfhcl.GetResourceType(block)

	// Track if we need to generate a moved block
	var movedBlock *hclwrite.Block

	// Rename resource type if it's the old name
	if resourceType == "cloudflare_access_application" {
		tfhcl.RenameResourceType(block, "cloudflare_access_application", "cloudflare_zero_trust_access_application")

		// Generate moved block for state migration
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock = tfhcl.CreateMovedBlock(from, to)
	}

	body := block.Body()

	// V4 has type default = "self_hosted", default to this value if type is not specified in V4 config
	tfhcl.EnsureAttribute(body, "type", "self_hosted")

	// V5 changed the default for http_only_cookie_attribute from false to true
	// Explicitly set to false to maintain v4 behavior when not specified
	// Only applicable for types: self_hosted, ssh, vnc, rdp, mcp_portal
	appType := tfhcl.ExtractStringFromAttribute(body.GetAttribute("type"))
	if appType == "self_hosted" || appType == "ssh" || appType == "vnc" || appType == "rdp" || appType == "mcp_portal" {
		tfhcl.EnsureAttribute(body, "http_only_cookie_attribute", "false")
	}

	tfhcl.RemoveAttributes(body, "domain_type")

	// Remove attributes with default/empty values that v4 provider removes from state
	// This prevents drift when migrating to v5
	removeDefaultValueAttributes(body)

	tfhcl.ConvertBlocksToAttribute(body, "cors_headers", "cors_headers", nil)
	tfhcl.ConvertBlocksToAttributeList(body, "destinations", nil)
	tfhcl.ConvertBlocksToAttributeList(body, "footer_links", nil)
	tfhcl.ConvertBlocksToAttribute(body, "landing_page_design", "landing_page_design", nil)

	tfhcl.ConvertArrayAttributeToObjectArray(body, "policies", func(element hclwrite.Tokens, index int) map[string]hclwrite.Tokens {
		return map[string]hclwrite.Tokens{
			"id": element,
			"precedence": {
				&hclwrite.Token{
					Type:  hclsyntax.TokenNumberLit,
					Bytes: []byte(strconv.Itoa(index + 1)),
				},
			},
		}
	})

	tfhcl.RemoveFunctionWrapper(body, "allowed_idps", "toset")
	tfhcl.RemoveFunctionWrapper(body, "custom_pages", "toset")
	tfhcl.RemoveFunctionWrapper(body, "self_hosted_domains", "toset")

	// Sort self_hosted_domains to match provider ordering and avoid drift
	tfhcl.SortStringArrayAttribute(body, "self_hosted_domains")

	m.transformSaasAppBlock(body)
	m.transformScimConfigBlock(body)
	m.transformTargetCriteriaBlocks(body)

	// Build result blocks
	blocks := []*hclwrite.Block{block}
	if movedBlock != nil {
		blocks = append(blocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: movedBlock != nil, // Remove original if we generated a moved block
	}, nil
}

// removeDefaultValueAttributes removes attributes that have default/empty values.
// v4 provider removes these from state, so we should remove them from config to avoid drift.
func removeDefaultValueAttributes(body *hclwrite.Body) {
	// Boolean attributes that should be removed if false
	boolAttrs := []string{
		"auto_redirect_to_identity",
		"enable_binding_cookie",
		"options_preflight_bypass",
		"service_auth_401_redirect",
		"skip_interstitial",
	}

	for _, attrName := range boolAttrs {
		if attr := body.GetAttribute(attrName); attr != nil {
			if val, ok := tfhcl.ExtractBoolFromAttribute(attr); ok && !val {
				// Remove if value is explicitly false
				tfhcl.RemoveAttributes(body, attrName)
			}
		}
	}

	// Array attributes that should be removed if empty
	arrayAttrs := []string{"allowed_idps", "tags"}
	for _, attrName := range arrayAttrs {
		if attr := body.GetAttribute(attrName); attr != nil {
			tokens := attr.Expr().BuildTokens(nil)
			// Check if it's an empty array []
			tokenStr := string(tokens.Bytes())
			if strings.TrimSpace(tokenStr) == "[]" {
				tfhcl.RemoveAttributes(body, attrName)
			}
		}
	}
}

// sortStringArrayAttribute sorts a string array attribute alphabetically.
// This is needed when the provider returns arrays in a consistent (sorted) order
// different from the user-specified order, causing drift.
// sortOIDCScopes sorts OIDC scopes according to the OIDC spec ordering.
// The provider orders scopes: openid, profile, email, address, phone,
// offline_access, then others alphabetically.
func sortOIDCScopes(strings []string) {
	// Define canonical OIDC scope order
	scopeOrder := map[string]int{
		"openid":         1,
		"profile":        2,
		"email":          3,
		"address":        4,
		"phone":          5,
		"offline_access": 6,
	}

	sort.SliceStable(strings, func(i, j int) bool {
		orderI, hasI := scopeOrder[strings[i]]
		orderJ, hasJ := scopeOrder[strings[j]]

		// Both have defined order - use it
		if hasI && hasJ {
			return orderI < orderJ
		}
		// Only i has order - it comes first
		if hasI {
			return true
		}
		// Only j has order - it comes first
		if hasJ {
			return false
		}
		// Neither has order - sort alphabetically
		return strings[i] < strings[j]
	})
}

func (m *V4ToV5Migrator) transformSaasAppBlock(body *hclwrite.Body) {
	saasAppBlocks := tfhcl.FindBlocksByType(body, "saas_app")
	if len(saasAppBlocks) == 0 {
		return
	}

	for _, saasAppBlock := range saasAppBlocks {
		saasAppBody := saasAppBlock.Body()

		// Process custom_attribute blocks before converting to list
		customAttrBlocks := tfhcl.FindBlocksByType(saasAppBody, "custom_attribute")
		for _, customAttrBlock := range customAttrBlocks {
			customAttrBody := customAttrBlock.Body()
			// Convert source block
			if sourceBlock := tfhcl.FindBlockByType(customAttrBody, "source"); sourceBlock != nil {
				sourceBody := sourceBlock.Body()
				// Convert source.name_by_idp from map to object array (SAML)
				tfhcl.ConvertMapAttributeToObjectArray(sourceBody, "name_by_idp", func(key hclwrite.Tokens, value hclwrite.Tokens) map[string]hclwrite.Tokens {
					return map[string]hclwrite.Tokens{
						"idp_id":      key,
						"source_name": value,
					}
				})
			}

			tfhcl.ConvertSingleBlockToAttribute(customAttrBody, "source", "source")
		}

		// Process custom_claim blocks before converting to list
		customClaimBlocks := tfhcl.FindBlocksByType(saasAppBody, "custom_claim")
		for _, customClaimBlock := range customClaimBlocks {
			customClaimBody := customClaimBlock.Body()
			// Convert source block to attribute
			// NOTE: For custom_claims (OIDC), name_by_idp stays as a map, so no transformation needed
			tfhcl.ConvertSingleBlockToAttribute(customClaimBody, "source", "source")
		}

		tfhcl.ConvertBlocksToAttributeList(saasAppBody, "custom_attribute", nil)
		tfhcl.RenameAttribute(saasAppBody, "custom_attribute", "custom_attributes")

		tfhcl.ConvertBlocksToAttributeList(saasAppBody, "custom_claim", nil)
		tfhcl.RenameAttribute(saasAppBody, "custom_claim", "custom_claims")

		tfhcl.ConvertSingleBlockToAttribute(saasAppBody, "hybrid_and_implicit_options", "hybrid_and_implicit_options")
		tfhcl.ConvertSingleBlockToAttribute(saasAppBody, "refresh_token_options", "refresh_token_options")

		// Sort scopes array to match provider ordering and avoid drift
		tfhcl.SortStringArrayAttribute(saasAppBody, "scopes", sortOIDCScopes)
	}

	tfhcl.ConvertSingleBlockToAttribute(body, "saas_app", "saas_app")
}

func (m *V4ToV5Migrator) transformScimConfigBlock(body *hclwrite.Body) {
	scimConfigBlocks := tfhcl.FindBlocksByType(body, "scim_config")
	if len(scimConfigBlocks) == 0 {
		return
	}

	for _, scimConfigBlock := range scimConfigBlocks {
		scimConfigBody := scimConfigBlock.Body()

		// Process authentication block
		if authBlock := tfhcl.FindBlockByType(scimConfigBody, "authentication"); authBlock != nil {
			authBody := authBlock.Body()
			// Convert toset() for scopes attribute
			tfhcl.RemoveFunctionWrapper(authBody, "scopes", "toset")
		}

		// Convert authentication block to attribute
		tfhcl.ConvertSingleBlockToAttribute(scimConfigBody, "authentication", "authentication")

		// Process mappings blocks
		mappingsBlocks := tfhcl.FindBlocksByType(scimConfigBody, "mappings")
		for _, mappingBlock := range mappingsBlocks {
			mappingBody := mappingBlock.Body()
			// Convert operations block to attribute
			tfhcl.ConvertSingleBlockToAttribute(mappingBody, "operations", "operations")
		}

		// Convert mappings blocks to list attribute
		tfhcl.ConvertBlocksToAttributeList(scimConfigBody, "mappings", nil)
	}

	// Convert scim_config block to attribute
	tfhcl.ConvertSingleBlockToAttribute(body, "scim_config", "scim_config")
}

func (m *V4ToV5Migrator) transformTargetCriteriaBlocks(body *hclwrite.Body) {
	// Get all target_criteria blocks
	targetCriteriaBlocks := tfhcl.FindBlocksByType(body, "target_criteria")

	// Convert nested target_attributes blocks within each target_criteria block to a map
	for _, tcBlock := range targetCriteriaBlocks {
		tcBody := tcBlock.Body()
		// Convert target_attributes blocks to map attribute
		m.convertTargetAttributesToMap(tcBody)
	}

	// Then convert the outer target_criteria blocks to list attribute
	tfhcl.ConvertBlocksToAttributeList(body, "target_criteria", nil)
}

// convertTargetAttributesToMap converts target_attributes blocks to a map attribute
// where keys are the "name" values and values are the "values" arrays
func (m *V4ToV5Migrator) convertTargetAttributesToMap(body *hclwrite.Body) {
	targetAttrBlocks := tfhcl.FindBlocksByType(body, "target_attributes")
	if len(targetAttrBlocks) == 0 {
		return
	}

	// Build map tokens
	var mapTokens hclwrite.Tokens

	// Opening brace
	mapTokens = append(mapTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})
	mapTokens = append(mapTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	// Process each target_attributes block
	for _, block := range targetAttrBlocks {
		blockBody := block.Body()

		// Get the name attribute (the map key)
		nameAttr := blockBody.GetAttribute("name")
		if nameAttr == nil {
			continue
		}

		// Get the values attribute (the map value)
		valuesAttr := blockBody.GetAttribute("values")
		if valuesAttr == nil {
			continue
		}

		// Add indentation
		mapTokens = append(mapTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("  "),
		})

		// Add the key (name value as a quoted string)
		nameTokens := nameAttr.Expr().BuildTokens(nil)
		mapTokens = append(mapTokens, nameTokens...)

		// Add equals sign
		mapTokens = append(mapTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte(" = "),
		})

		// Add the value (values array)
		valuesTokens := valuesAttr.Expr().BuildTokens(nil)
		mapTokens = append(mapTokens, valuesTokens...)

		// Add newline
		mapTokens = append(mapTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
	}

	// Closing brace
	mapTokens = append(mapTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte("}"),
	})

	// Set the map attribute
	body.SetAttributeRaw("target_attributes", mapTokens)

	// Remove the original blocks
	for _, block := range targetAttrBlocks {
		body.RemoveBlock(block)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is now handled by the provider's StateUpgraders.
	// This function is a no-op - just return the original state unchanged.
	return stateJSON.String(), nil
}
