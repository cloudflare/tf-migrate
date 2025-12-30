package zero_trust_access_application

import (
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
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
	// Rename resource type if it's the old name
	resourceType := tfhcl.GetResourceType(block)
	if resourceType == "cloudflare_access_application" {
		tfhcl.RenameResourceType(block, "cloudflare_access_application", "cloudflare_zero_trust_access_application")
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
	sortStringArrayAttribute(body, "self_hosted_domains")

	m.transformSaasAppBlock(body)
	m.transformScimConfigBlock(body)
	m.transformTargetCriteriaBlocks(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
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
func sortStringArrayAttribute(body *hclwrite.Body, attrName string) {
	attr := body.GetAttribute(attrName)
	if attr == nil {
		return
	}

	// Parse the expression to extract string values
	expr := attr.Expr()

	// Try to parse as tuple (array)
	tokens := expr.BuildTokens(nil)
	tokenBytes := tokens.Bytes()

	// Parse the HCL expression
	parsed, diags := hclsyntax.ParseExpression(tokenBytes, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return
	}

	// Check if it's a tuple (array)
	tuple, ok := parsed.(*hclsyntax.TupleConsExpr)
	if !ok {
		return
	}

	// Extract string values
	var strings []string
	for _, elem := range tuple.Exprs {
		if template, ok := elem.(*hclsyntax.TemplateExpr); ok {
			// Handle string literals
			if len(template.Parts) == 1 {
				if lit, ok := template.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
					if lit.Val.Type() == cty.String {
						strings = append(strings, lit.Val.AsString())
					}
				}
			}
		}
	}

	// If we couldn't extract all strings, don't modify
	if len(strings) != len(tuple.Exprs) {
		return
	}

	// Sort the strings
	// Special handling for OIDC scopes: use canonical OIDC scope ordering
	// The provider orders scopes according to the OIDC spec: openid, profile, email, then others alphabetically
	if attrName == "scopes" && len(strings) > 0 {
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
	} else {
		// Not scopes, sort normally
		sort.Strings(strings)
	}

	// Build new array tokens with sorted values
	var sortedTokens []hclwrite.Tokens
	for _, s := range strings {
		sortedTokens = append(sortedTokens, hclwrite.TokensForValue(cty.StringVal(s)))
	}

	// Set the attribute with sorted values
	body.SetAttributeRaw(attrName, hclwrite.TokensForTuple(sortedTokens))
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
		sortStringArrayAttribute(saasAppBody, "scopes")
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
	result := stateJSON.String()

	// Handle both full state and single instance transformation
	if stateJSON.Get("resources").Exists() {
		return m.transformFullState(result, stateJSON, ctx)
	}

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	// Rename resource type if it's the old name (for single instance tests)
	resourceType := stateJSON.Get("type").String()
	if resourceType == "cloudflare_access_application" {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_access_application")
	}

	result = m.transformSingleInstance(result, stateJSON, ctx, resourcePath, resourceName)

	return result, nil
}

func (m *V4ToV5Migrator) transformFullState(result string, stateJSON gjson.Result, ctx *transform.Context) (string, error) {
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		if !m.CanHandle(resourceType) {
			return true // continue
		}

		// Rename cloudflare_access_application to cloudflare_zero_trust_access_application
		if resourceType == "cloudflare_access_application" {
			resourcePath := "resources." + key.String() + ".type"
			result, _ = sjson.Set(result, resourcePath, "cloudflare_zero_trust_access_application")
		}

		resourceName := resource.Get("name").String()
		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()
			resourcePath := "resources." + key.String()

			attrs := instance.Get("attributes")
			if attrs.Exists() {
				instJSON := instance.String()
				transformedInst := m.transformSingleInstance(instJSON, instance, ctx, resourcePath, resourceName)
				transformedInstParsed := gjson.Parse(transformedInst)
				result, _ = sjson.SetRaw(result, instPath, transformedInstParsed.Raw)
			}
			return true
		})

		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result, ctx *transform.Context, resourcePath, resourceName string) string {
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result
	}

	// If ctx is nil, create an empty context so transformations can still run
	if ctx == nil {
		ctx = &transform.Context{
			Diagnostics: make(hcl.Diagnostics, 0),
		}
	}

	attrPath := "attributes"

	// Set default type to "self_hosted" if not present (V4 schema default)
	result = state.EnsureField(result, attrPath, attrs, "type", "self_hosted")

	// Remove deprecated domain_type attribute
	result = state.RemoveFields(result, attrPath, attrs, "domain_type")

	// Apply transformations in logical order
	result = m.transformSetToListFields(result, attrs, attrPath)
	result = m.transformCoorsHeaders(result, attrs, attrPath, ctx, resourceName)
	result = m.transformLandingPageDesign(result, attrs, attrPath, ctx, resourceName)
	result = m.transformSaasApp(result, attrs, attrPath, ctx, resourceName)
	result = m.transformScimConfig(result, attrs, attrPath, ctx, resourceName)
	result = m.transformPolicies(result, attrs, attrPath)
	result = m.transformTargetCriteria(result, attrs, attrPath)
	result = m.transformDestinations(result, attrPath)

	// Transform empty values to null for top-level attributes not explicitly set in config
	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
		Ctx:              ctx,
		Result:           result,
		FieldPath:        attrPath,
		FieldResult:      gjson.Parse(result).Get(attrPath),
		ResourceName:     resourceName,
		HCLAttributePath: "",
		CanHandle:        m.CanHandle,
	})

	// Always set schema_version
	result = state.SetSchemaVersion(result, 0)

	return result
}

// transformSetToListFields transforms set-typed fields to list-typed fields
func (m *V4ToV5Migrator) transformSetToListFields(result string, attrs gjson.Result, attrPath string) string {
	// Transform allowed_idps from set to list (same values, different type metadata)
	allowedIdPs := attrs.Get("allowed_idps")
	if allowedIdPs.Exists() {
		result, _ = sjson.Set(result, attrPath+".allowed_idps", allowedIdPs.Value())
	}

	// Transform custom_pages from set to list (same values, different type metadata)
	customPages := attrs.Get("custom_pages")
	if customPages.Exists() {
		result, _ = sjson.Set(result, attrPath+".custom_pages", customPages.Value())
	}

	// Transform self_hosted_domains from set to list (same values, different type metadata)
	selfHostedDomains := attrs.Get("self_hosted_domains")
	if selfHostedDomains.Exists() {
		result, _ = sjson.Set(result, attrPath+".self_hosted_domains", selfHostedDomains.Value())
	}

	return result
}

func (m *V4ToV5Migrator) transformCoorsHeaders(result string, attrs gjson.Result, attrPath string, ctx *transform.Context, resourceName string) string {
	// First, check if cors_headers field should be null based on HCL config
	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
		Ctx:              ctx,
		Result:           result,
		FieldPath:        attrPath,
		FieldResult:      attrs,
		ResourceName:     resourceName,
		HCLAttributePath: "cors_headers",
		CanHandle:        m.CanHandle,
	})

	// Re-parse to check if field still exists after transformation
	attrs = gjson.Parse(result).Get(attrPath)

	// Transform cors_headers from array format to object format (if it still exists)
	result = state.TransformFieldArrayToObject(result, attrPath, attrs, "cors_headers", state.ArrayToObjectOptions{
		TransformEmptyToNull: true,
	})

	transformedCorsHeaders := gjson.Parse(result).Get(attrPath + ".cors_headers")
	if transformedCorsHeaders.Exists() {
		maxAge := transformedCorsHeaders.Get("max_age")
		if maxAge.Exists() {
			result, _ = sjson.Set(result, attrPath+".cors_headers.max_age", state.ConvertToFloat64(maxAge))
		}

		// Transform empty values within cors_headers nested fields
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".cors_headers",
			FieldResult:      gjson.Parse(result).Get(attrPath + ".cors_headers"),
			ResourceName:     resourceName,
			HCLAttributePath: "cors_headers",
			CanHandle:        m.CanHandle,
		})
	}

	return result
}

func (m *V4ToV5Migrator) transformLandingPageDesign(result string, attrs gjson.Result, attrPath string, ctx *transform.Context, resourceName string) string {
	// First, check if landing_page_design field should be null based on HCL config
	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
		Ctx:              ctx,
		Result:           result,
		FieldPath:        attrPath,
		FieldResult:      attrs,
		ResourceName:     resourceName,
		HCLAttributePath: "landing_page_design",
		CanHandle:        m.CanHandle,
	})

	// Re-parse to check if field still exists after transformation
	attrs = gjson.Parse(result).Get(attrPath)

	// Transform landing_page_design from array format to object format (if it still exists)
	result = state.TransformFieldArrayToObject(result, attrPath, attrs, "landing_page_design", state.ArrayToObjectOptions{
		TransformEmptyToNull: true,
	})

	// Add default: landing_page_design.title = "Welcome!" if not present (only when landing_page_design exists)
	transformedLandingPage := gjson.Parse(result).Get(attrPath + ".landing_page_design")
	if transformedLandingPage.Exists() && transformedLandingPage.IsObject() {
		result = state.EnsureField(result, attrPath+".landing_page_design", transformedLandingPage, "title", "Welcome!")

		// Transform empty values within landing_page_design nested fields
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".landing_page_design",
			FieldResult:      gjson.Parse(result).Get(attrPath + ".landing_page_design"),
			ResourceName:     resourceName,
			HCLAttributePath: "landing_page_design",
			CanHandle:        m.CanHandle,
		})
	}

	return result
}

// transformSaasApp transforms the saas_app block and its nested structures
func (m *V4ToV5Migrator) transformSaasApp(result string, attrs gjson.Result, attrPath string, ctx *transform.Context, resourceName string) string {
	// First, check if saas_app field should be null based on HCL config
	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
		Ctx:              ctx,
		Result:           result,
		FieldPath:        attrPath,
		FieldResult:      attrs,
		ResourceName:     resourceName,
		HCLAttributePath: "saas_app",
		CanHandle:        m.CanHandle,
	})

	// Re-parse to check if field still exists after transformation
	attrs = gjson.Parse(result).Get(attrPath)

	// Transform saas_app from array format to object format (if it still exists)
	result = state.TransformFieldArrayToObject(result, attrPath, attrs, "saas_app", state.ArrayToObjectOptions{
		TransformEmptyToNull: true,
	})

	// Transform nested MaxItems:1 fields within saas_app
	transformedSaasApp := gjson.Parse(result).Get(attrPath + ".saas_app")
	if transformedSaasApp.Exists() {
		// First check if hybrid_and_implicit_options should be null
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".saas_app",
			FieldResult:      transformedSaasApp,
			ResourceName:     resourceName,
			HCLAttributePath: "saas_app.hybrid_and_implicit_options",
			CanHandle:        m.CanHandle,
		})

		// Re-parse after transformation
		transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")

		// Transform hybrid_and_implicit_options from array to object (if it still exists)
		result = state.TransformFieldArrayToObject(result, attrPath+".saas_app", transformedSaasApp, "hybrid_and_implicit_options", state.ArrayToObjectOptions{
			TransformEmptyToNull: true,
		})

		// Transform empty values within hybrid_and_implicit_options nested fields
		transformedHybrid := gjson.Parse(result).Get(attrPath + ".saas_app.hybrid_and_implicit_options")
		if transformedHybrid.Exists() && transformedHybrid.IsObject() {
			result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
				Ctx:              ctx,
				Result:           result,
				FieldPath:        attrPath + ".saas_app.hybrid_and_implicit_options",
				FieldResult:      transformedHybrid,
				ResourceName:     resourceName,
				HCLAttributePath: "saas_app.hybrid_and_implicit_options",
				CanHandle:        m.CanHandle,
			})
		}

		// First check if refresh_token_options should be null
		transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".saas_app",
			FieldResult:      transformedSaasApp,
			ResourceName:     resourceName,
			HCLAttributePath: "saas_app.refresh_token_options",
			CanHandle:        m.CanHandle,
		})

		// Re-parse after transformation
		transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")

		// Transform refresh_token_options from array to object (if it still exists)
		result = state.TransformFieldArrayToObject(result, attrPath+".saas_app", transformedSaasApp, "refresh_token_options", state.ArrayToObjectOptions{
			TransformEmptyToNull: true,
		})

		// Transform empty values within refresh_token_options nested fields
		transformedRefresh := gjson.Parse(result).Get(attrPath + ".saas_app.refresh_token_options")
		if transformedRefresh.Exists() && transformedRefresh.IsObject() {
			result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
				Ctx:              ctx,
				Result:           result,
				FieldPath:        attrPath + ".saas_app.refresh_token_options",
				FieldResult:      transformedRefresh,
				ResourceName:     resourceName,
				HCLAttributePath: "saas_app.refresh_token_options",
				CanHandle:        m.CanHandle,
			})
		}

		// Transform custom_attribute[].source from array to object for each item
		customAttrs := transformedSaasApp.Get("custom_attribute")
		if customAttrs.Exists() && customAttrs.IsArray() {
			customAttrs.ForEach(func(idx, item gjson.Result) bool {
				itemPath := attrPath + ".saas_app.custom_attribute." + idx.String()
				result = state.TransformFieldArrayToObject(result, itemPath, item, "source", state.ArrayToObjectOptions{})
				return true
		})

			// Transform custom_attributes[].source.name_by_idp from map to list of objects (SAML)
			transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
			customAttrs = transformedSaasApp.Get("custom_attribute")
			if customAttrs.Exists() && customAttrs.IsArray() {
				customAttrs.ForEach(func(idx, item gjson.Result) bool {
					nameByIdp := item.Get("source.name_by_idp")
					if nameByIdp.Exists() && nameByIdp.IsObject() {
						// Convert map to array of objects with idp_id and source_name
						var nameByIdpArray []map[string]interface{}
						nameByIdp.ForEach(func(key, value gjson.Result) bool {
							nameByIdpArray = append(nameByIdpArray, map[string]interface{}{
								"idp_id":      key.String(),
								"source_name": value.String(),
							})
							return true
						})
						result, _ = sjson.Set(result, attrPath+".saas_app.custom_attribute."+idx.String()+".source.name_by_idp", nameByIdpArray)
					}
					return true
				})
			}

			// Rename custom_attribute to custom_attributes (plural)
			transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
			result = state.RenameField(result, attrPath+".saas_app", transformedSaasApp, "custom_attribute", "custom_attributes")
		}

		// Transform custom_claim[].source from array to object for each item
		transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
		customClaims := transformedSaasApp.Get("custom_claim")
		if customClaims.Exists() && customClaims.IsArray() {
			customClaims.ForEach(func(idx, item gjson.Result) bool {
				itemPath := attrPath + ".saas_app.custom_claim." + idx.String()
				result = state.TransformFieldArrayToObject(result, itemPath, item, "source", state.ArrayToObjectOptions{})

				// Remove empty name_by_idp maps from custom_claims (OIDC)
				// Empty maps should be null/absent, not {}
				transformedItem := gjson.Parse(result).Get(itemPath)
				nameByIdp := transformedItem.Get("source.name_by_idp")
				if nameByIdp.Exists() && nameByIdp.IsObject() {
					// Check if it's an empty object
					isEmpty := true
					nameByIdp.ForEach(func(key, value gjson.Result) bool {
						isEmpty = false
						return false // Stop iteration
					})
					if isEmpty {
						// Remove the empty name_by_idp field
						result, _ = sjson.Delete(result, itemPath+".source.name_by_idp")
					}
				}

				return true
		})
			// Rename custom_claim to custom_claims (plural)
			transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
			result = state.RenameField(result, attrPath+".saas_app", transformedSaasApp, "custom_claim", "custom_claims")
		}

		// Add default: saas_app.auth_type = "saml" if not present
		transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
		if transformedSaasApp.Exists() && transformedSaasApp.IsObject() {
			result = state.EnsureField(result, attrPath+".saas_app", transformedSaasApp, "auth_type", "saml")
		}

		// Transform empty values within saas_app (top level of saas_app)
		transformedSaasApp = gjson.Parse(result).Get(attrPath + ".saas_app")
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".saas_app",
			FieldResult:      transformedSaasApp,
			ResourceName:     resourceName,
			HCLAttributePath: "saas_app",
			CanHandle:        m.CanHandle,
		})
	}

	return result
}

// transformScimConfig transforms the scim_config block and its nested structures
func (m *V4ToV5Migrator) transformScimConfig(result string, attrs gjson.Result, attrPath string, ctx *transform.Context, resourceName string) string {
	// First, check if scim_config field should be null based on HCL config
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath,
			FieldResult:      attrs,
			ResourceName:     resourceName,
			HCLAttributePath: "scim_config",
			CanHandle:        m.CanHandle,
		})

	// Re-parse to check if field still exists after transformation
	attrs = gjson.Parse(result).Get(attrPath)

	// Transform scim_config from array format to object format (if it still exists)
	result = state.TransformFieldArrayToObject(result, attrPath, attrs, "scim_config", state.ArrayToObjectOptions{
		TransformEmptyToNull: true,
	})

	// Transform nested MaxItems:1 fields within scim_config
	transformedScimConfig := gjson.Parse(result).Get(attrPath + ".scim_config")
	if transformedScimConfig.Exists() && transformedScimConfig.IsObject() {
		// First check if authentication should be null
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".scim_config",
			FieldResult:      transformedScimConfig,
			ResourceName:     resourceName,
			HCLAttributePath: "scim_config.authentication",
			CanHandle:        m.CanHandle,
		})

		// Re-parse after transformation
		transformedScimConfig = gjson.Parse(result).Get(attrPath + ".scim_config")

		// Transform authentication from array to object (if it still exists)
		result = state.TransformFieldArrayToObject(result, attrPath+".scim_config", transformedScimConfig, "authentication", state.ArrayToObjectOptions{})

		// Transform empty values within authentication nested fields
		transformedAuth := gjson.Parse(result).Get(attrPath + ".scim_config.authentication")
		if transformedAuth.Exists() && transformedAuth.IsObject() {
			result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
				Ctx:              ctx,
				Result:           result,
				FieldPath:        attrPath + ".scim_config.authentication",
				FieldResult:      transformedAuth,
				ResourceName:     resourceName,
				HCLAttributePath: "scim_config.authentication",
				CanHandle:        m.CanHandle,
			})
		}

		// Transform mappings[].operations from array to object for each mapping
		transformedScimConfig = gjson.Parse(result).Get(attrPath + ".scim_config")
		mappings := transformedScimConfig.Get("mappings")
		if mappings.Exists() && mappings.IsArray() {
			mappings.ForEach(func(idx, mapping gjson.Result) bool {
				itemPath := attrPath + ".scim_config.mappings." + idx.String()

				// First check if operations should be null
					result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
						Ctx:              ctx,
						Result:           result,
						FieldPath:        itemPath,
						FieldResult:      gjson.Parse(result).Get(itemPath),
						ResourceName:     resourceName,
						HCLAttributePath: "scim_config.mappings.operations",
						CanHandle:        m.CanHandle,
					})

				// Re-parse after transformation
				mapping = gjson.Parse(result).Get(itemPath)

				// Transform operations from array to object (if it still exists)
				result = state.TransformFieldArrayToObject(result, itemPath, mapping, "operations", state.ArrayToObjectOptions{})

				// Transform empty values within operations nested fields
				transformedOps := gjson.Parse(result).Get(itemPath + ".operations")
				if transformedOps.Exists() && transformedOps.IsObject() {
					result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
						Ctx:              ctx,
						Result:           result,
						FieldPath:        itemPath + ".operations",
						FieldResult:      transformedOps,
						ResourceName:     resourceName,
						HCLAttributePath: "scim_config.mappings.operations",
						CanHandle:        m.CanHandle,
					})
				}
				return true
		})
		}

		// Transform empty values within scim_config (top level of scim_config)
		transformedScimConfig = gjson.Parse(result).Get(attrPath + ".scim_config")
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        attrPath + ".scim_config",
			FieldResult:      transformedScimConfig,
			ResourceName:     resourceName,
			HCLAttributePath: "scim_config",
			CanHandle:        m.CanHandle,
		})
	}

	return result
}

// transformPolicies transforms the policies field from string list to object list
func (m *V4ToV5Migrator) transformPolicies(result string, attrs gjson.Result, attrPath string) string {
	// Transform policies from simple string list to complex object list
	policies := attrs.Get("policies")
	if policies.IsArray() {
		var transformedPolicies []interface{}
		policies.ForEach(func(idx, policy gjson.Result) bool {
			if policy.Type == gjson.String {
				// Convert string policy ID to object with id and precedence fields
				transformedPolicies = append(transformedPolicies, map[string]interface{}{
					"id":         policy.String(),
					"precedence": idx.Int() + 1,
				})
			} else {
				// Keep as-is if already an object
				transformedPolicies = append(transformedPolicies, policy.Value())
			}
			return true
		})
		if len(transformedPolicies) > 0 {
			result, _ = sjson.Set(result, attrPath+".policies", transformedPolicies)
		}
	}

	return result
}

// transformTargetCriteria transforms target_criteria nested structures
func (m *V4ToV5Migrator) transformTargetCriteria(result string, attrs gjson.Result, attrPath string) string {
	// Transform target_criteria[].target_attributes from array of {name, values} to map
	targetCriteria := attrs.Get("target_criteria")
	if targetCriteria.Exists() && targetCriteria.IsArray() {
		targetCriteria.ForEach(func(criteriaIdx, criteria gjson.Result) bool {
			targetAttrs := criteria.Get("target_attributes")
			if targetAttrs.Exists() && targetAttrs.IsArray() {
				// Build map from array of {name, values} objects
				attrMap := make(map[string]interface{})
				targetAttrs.ForEach(func(_, attr gjson.Result) bool {
					name := attr.Get("name")
					values := attr.Get("values")
					if name.Exists() && values.Exists() {
						attrMap[name.String()] = values.Value()
					}
					return true
				})
				// Replace the array with the map
				if len(attrMap) > 0 {
					result, _ = sjson.Set(result, attrPath+".target_criteria."+criteriaIdx.String()+".target_attributes", attrMap)
				}
			}
			return true
		})
	}

	return result
}

// transformDestinations adds default values to destinations
func (m *V4ToV5Migrator) transformDestinations(result string, attrPath string) string {
	// Add default: destinations[].type = "public" if not present
	destinations := gjson.Parse(result).Get(attrPath + ".destinations")
	if destinations.Exists() && destinations.IsArray() {
		destinations.ForEach(func(idx, dest gjson.Result) bool {
			if !dest.Get("type").Exists() {
				result, _ = sjson.Set(result, attrPath+".destinations."+idx.String()+".type", "public")
			}
			return true
		})
	}

	return result
}
