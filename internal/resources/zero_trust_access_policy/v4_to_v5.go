package zero_trust_access_policy

import (
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust Access Policy resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the OLD (v4) resource name: cloudflare_access_policy
	internal.RegisterMigrator("cloudflare_access_policy", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_access_policy"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_access_policy"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for now - all transformations can be done with HCL manipulation
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_policy", "cloudflare_zero_trust_access_policy"
}

// TransformState is a no-op for this migrator - state transformation is handled by moved blocks
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Get the resource name before renaming (for moved block generation)
	resourceName := tfhcl.GetResourceName(block)
	resourceType := tfhcl.GetResourceType(block)

	// Track if we need to generate a moved block
	var movedBlock *hclwrite.Block

	// 1. Rename resource type: cloudflare_access_policy → cloudflare_zero_trust_access_policy
	if resourceType == "cloudflare_access_policy" {
		tfhcl.RenameResourceType(block, "cloudflare_access_policy", "cloudflare_zero_trust_access_policy")

		// Generate moved block for state migration
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock = tfhcl.CreateMovedBlock(from, to)
	}

	body := block.Body()

	// 2. Simple field operations
	// Remove deprecated fields
	// session_duration is not valid for policies (only for applications)
	tfhcl.RemoveAttributes(body, "application_id", "precedence", "zone_id", "session_duration")

	// Convert approval_group block to approval_groups attribute array
	// In v4: approval_group { approvals_needed = 1 }
	// In v5: approval_groups = [{ approvals_needed = 1 }]
	tfhcl.ConvertBlocksToArrayAttribute(body, "approval_group", false)
	tfhcl.RenameAttribute(body, "approval_group", "approval_groups")

	// 3. Process connection_rules (MaxItems:1 transformation)
	// First process nested ssh block, then convert parent block
	if connRulesBlock := tfhcl.FindBlockByType(body, "connection_rules"); connRulesBlock != nil {
		connRulesBody := connRulesBlock.Body()
		// Convert ssh block to attribute syntax
		tfhcl.ConvertSingleBlockToAttribute(connRulesBody, "ssh", "ssh")
	}
	// Convert connection_rules block to attribute syntax
	tfhcl.ConvertSingleBlockToAttribute(body, "connection_rules", "connection_rules")

	// 4. First convert include/exclude/require blocks to attributes if needed
	m.convertConditionBlocksToAttributes(body)

	// 5. Then process include/exclude/require condition transformations
	// These require complex AST manipulation
	m.transformConditionAttributes(body)

	// 6. Remove empty exclude and require arrays (v5 provider normalizes them to null)
	// Only remove exclude and require, keep include even if empty
	m.removeEmptyConditionArrays(body)

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

// convertConditionBlocksToAttributes converts include/exclude/require blocks to attribute arrays
// This is what Grit patterns do in the old migration, but we need to handle it here
func (m *V4ToV5Migrator) convertConditionBlocksToAttributes(body *hclwrite.Body) {
	conditionNames := []string{"include", "exclude", "require"}

	for _, condName := range conditionNames {
		// Use the helper to convert blocks to attributes
		// This will convert: include { email = [...] } -> include = [{ email = [...] }]
		tfhcl.ConvertBlocksToArrayAttribute(body, condName, false)
	}
}

// removeEmptyConditionArrays removes empty exclude and require arrays
// The v5 provider normalizes empty arrays to null, so we should omit them
func (m *V4ToV5Migrator) removeEmptyConditionArrays(body *hclwrite.Body) {
	// Only remove exclude and require, keep include even if empty
	conditionsToCheck := []string{"exclude", "require"}

	for _, attrName := range conditionsToCheck {
		attr := body.GetAttribute(attrName)
		if attr == nil {
			continue
		}

		// Parse the attribute expression to check if it's an empty array
		expr := attr.Expr()
		src := hclwrite.Format(expr.BuildTokens(nil).Bytes())

		// Parse as syntax expression
		syntaxExpr, diags := hclsyntax.ParseExpression(src, attrName, hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}

		// Check if it's an empty tuple (array)
		if tup, ok := syntaxExpr.(*hclsyntax.TupleConsExpr); ok {
			if len(tup.Exprs) == 0 {
				// It's an empty array, remove the attribute
				body.RemoveAttribute(attrName)
			}
		}
	}
}

// transformConditionAttributes transforms include/exclude/require attributes
// Handles:
// 1. Boolean attributes (everyone, certificate, any_valid_service_token) -> empty objects
// 2. Array attributes (email, group, ip, email_domain, geo) -> split into multiple objects
// 3. GitHub blocks -> rename to github_organization and expand teams
func (m *V4ToV5Migrator) transformConditionAttributes(body *hclwrite.Body) {
	conditionAttrs := []string{"include", "exclude", "require"}

	for _, attrName := range conditionAttrs {
		attr := body.GetAttribute(attrName)
		if attr == nil {
			continue
		}

		// Parse the attribute expression
		expr := attr.Expr()
		src := hclwrite.Format(expr.BuildTokens(nil).Bytes())

		// Normalize IP addresses in the source before parsing
		// This ensures single IPs like "192.168.1.1" become "192.168.1.1/32"
		src = []byte(m.normalizeIPsInSource(string(src)))

		// Parse as syntax expression to manipulate
		syntaxExpr, diags := hclsyntax.ParseExpression(src, attrName, hcl.InitialPos)
		if diags.HasErrors() {
			// Can't parse - leave as is
			continue
		}

		// Transform the expression
		transformedExpr := m.transformConditionExpression(syntaxExpr)
		if transformedExpr == nil {
			continue
		}

		// Convert back to tokens and set the attribute
		tokens := m.exprToTokens(transformedExpr)
		body.SetAttributeRaw(attrName, tokens)
	}
}

// transformConditionExpression transforms a condition list expression
// Input: [{everyone = true, email = ["a", "b"]}]
// Output: [{everyone = {}}, {email = {email = "a"}}, {email = {email = "b"}}]
func (m *V4ToV5Migrator) transformConditionExpression(expr hclsyntax.Expression) hclsyntax.Expression {
	tup, ok := expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		// Not a tuple/array, can't transform
		return nil
	}

	var newExprs []hclsyntax.Expression

	for _, itemExpr := range tup.Exprs {
		obj, ok := itemExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			// Keep non-object expressions as-is
			newExprs = append(newExprs, itemExpr)
			continue
		}

		// First, transform booleans in place
		obj = m.transformBooleans(obj)

		// Then check if we need to expand arrays or github
		expanded := m.expandObject(obj)
		if len(expanded) > 0 {
			// Object was expanded into multiple objects
			newExprs = append(newExprs, expanded...)
		} else {
			// No expansion needed, keep original object
			newExprs = append(newExprs, obj)
		}
	}

	// Return the modified tuple
	return &hclsyntax.TupleConsExpr{
		Exprs:     newExprs,
		SrcRange:  tup.SrcRange,
		OpenRange: tup.OpenRange,
	}
}

// transformBooleans transforms boolean attributes to empty objects
// everyone = true -> everyone = {}
// certificate = false -> (removed)
func (m *V4ToV5Migrator) transformBooleans(obj *hclsyntax.ObjectConsExpr) *hclsyntax.ObjectConsExpr {
	boolAttrs := map[string]bool{
		"everyone":                true,
		"certificate":             true,
		"any_valid_service_token": true,
	}

	var newItems []hclsyntax.ObjectConsItem

	for _, item := range obj.Items {
		key := m.getKeyString(item.KeyExpr)

		if boolAttrs[key] {
			// Check if it's a boolean literal
			lit, ok := item.ValueExpr.(*hclsyntax.LiteralValueExpr)
			if ok && lit.Val.Type() == cty.Bool {
				if lit.Val.False() {
					// Remove false values
					continue
				}
				// Replace true with empty object
				newItems = append(newItems, hclsyntax.ObjectConsItem{
					KeyExpr:   item.KeyExpr,
					ValueExpr: &hclsyntax.ObjectConsExpr{},
				})
				continue
			}
		}

		// Keep other items as-is
		newItems = append(newItems, item)
	}

	return &hclsyntax.ObjectConsExpr{
		Items:    newItems,
		SrcRange: obj.SrcRange,
	}
}

// expandObject checks if an object needs expansion and returns expanded objects
// Returns nil if no expansion needed
func (m *V4ToV5Migrator) expandObject(obj *hclsyntax.ObjectConsExpr) []hclsyntax.Expression {
	var allExpanded []hclsyntax.Expression
	var remainingItems []hclsyntax.ObjectConsItem

	for _, item := range obj.Items {
		key := m.getKeyString(item.KeyExpr)

		// Handle github specially
		if key == "github" {
			expanded := m.expandGithub(item)
			if len(expanded) > 0 {
				allExpanded = append(allExpanded, expanded...)
				continue
			}
		}

		// Handle simple array attributes
		expanded := m.expandArrayAttribute(key, item)
		if len(expanded) > 0 {
			allExpanded = append(allExpanded, expanded...)
			continue
		}

		// Keep other attributes as-is
		remainingItems = append(remainingItems, item)
	}

	// If we expanded some attributes, each remaining item becomes its own object
	if len(allExpanded) > 0 {
		for _, item := range remainingItems {
			singleItemObj := &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{item},
			}
			allExpanded = append(allExpanded, singleItemObj)
		}
		return allExpanded
	}

	// No expansion happened
	return nil
}

// expandArrayAttribute expands array attributes like email, group, ip
// email = ["a", "b"] -> [{email = {email = "a"}}, {email = {email = "b"}}]
// Also handles single string values: common_name = "device" -> {common_name = {common_name = "device"}}
func (m *V4ToV5Migrator) expandArrayAttribute(key string, item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	// Map of attribute names to their inner field names
	arrayAttrs := map[string]string{
		"email":        "email",
		"group":        "id",
		"ip":           "ip",
		"email_domain": "domain",
		"geo":          "country_code",
		"common_name":  "common_name",
		"auth_method":  "auth_method",
		"login_method": "id",
	}

	innerFieldName, isArrayAttr := arrayAttrs[key]
	if !isArrayAttr {
		return nil
	}

	// Check if already transformed: if value is an object with a single item matching innerFieldName, skip
	if obj, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr); ok {
		if len(obj.Items) == 1 {
			itemKey := m.getKeyString(obj.Items[0].KeyExpr)
			if itemKey == innerFieldName {
				// Already transformed, return nil to skip
				return nil
			}
		}
	}

	var result []hclsyntax.Expression

	// Check if the value is a tuple/array
	if tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr); ok {
		// Create a new object for each item in the array
		for _, elem := range tup.Exprs {
			// Normalize IP addresses: add /32 suffix if missing
			valueExpr := elem
			if key == "ip" {
				valueExpr = m.normalizeIPExpression(elem)
			}

			newObj := &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{
					{
						KeyExpr: m.newKeyExpr(key),
						ValueExpr: &hclsyntax.ObjectConsExpr{
							Items: []hclsyntax.ObjectConsItem{
								{
									KeyExpr:   m.newKeyExpr(innerFieldName),
									ValueExpr: valueExpr,
								},
							},
						},
					},
				},
			}
			result = append(result, newObj)
		}
		return result
	}

	// Handle single string value (not an array)
	// common_name = "device" -> {common_name = {common_name = "device"}}
	newObj := &hclsyntax.ObjectConsExpr{
		Items: []hclsyntax.ObjectConsItem{
			{
				KeyExpr: m.newKeyExpr(key),
				ValueExpr: &hclsyntax.ObjectConsExpr{
					Items: []hclsyntax.ObjectConsItem{
						{
							KeyExpr:   m.newKeyExpr(innerFieldName),
							ValueExpr: item.ValueExpr,
						},
					},
				},
			},
		},
	}
	return []hclsyntax.Expression{newObj}
}

// expandGithub handles the special case of github blocks
// github = [{name = "org", teams = ["t1", "t2"], identity_provider_id = "id"}]
// -> Multiple github_organization objects, one per team
func (m *V4ToV5Migrator) expandGithub(item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	// Check if the value is a tuple/array
	tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil
	}

	var result []hclsyntax.Expression

	for _, githubExpr := range tup.Exprs {
		githubObj, ok := githubExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			continue
		}

		// Extract fields
		var nameExpr hclsyntax.Expression
		var teamsExpr *hclsyntax.TupleConsExpr
		var identityProviderExpr hclsyntax.Expression
		var otherItems []hclsyntax.ObjectConsItem

		for _, githubItem := range githubObj.Items {
			itemKey := m.getKeyString(githubItem.KeyExpr)
			switch itemKey {
			case "name":
				nameExpr = githubItem.ValueExpr
			case "teams":
				teamsExpr, _ = githubItem.ValueExpr.(*hclsyntax.TupleConsExpr)
			case "identity_provider_id":
				identityProviderExpr = githubItem.ValueExpr
			default:
				otherItems = append(otherItems, githubItem)
			}
		}

		// Expand teams array
		if teamsExpr != nil && len(teamsExpr.Exprs) > 0 {
			for _, teamExpr := range teamsExpr.Exprs {
				items := m.buildGithubOrgItems(nameExpr, teamExpr, identityProviderExpr, otherItems)
				newObj := &hclsyntax.ObjectConsExpr{
					Items: []hclsyntax.ObjectConsItem{
						{
							KeyExpr: m.newKeyExpr("github_organization"),
							ValueExpr: &hclsyntax.ObjectConsExpr{
								Items: items,
							},
						},
					},
				}
				result = append(result, newObj)
			}
		} else {
			// No teams array, create single github_organization
			items := m.buildGithubOrgItems(nameExpr, nil, identityProviderExpr, otherItems)
			newObj := &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{
					{
						KeyExpr: m.newKeyExpr("github_organization"),
						ValueExpr: &hclsyntax.ObjectConsExpr{
							Items: items,
						},
					},
				},
			}
			result = append(result, newObj)
		}
	}

	return result
}

// buildGithubOrgItems builds the items for a github_organization object
func (m *V4ToV5Migrator) buildGithubOrgItems(nameExpr, teamExpr, identityProviderExpr hclsyntax.Expression, otherItems []hclsyntax.ObjectConsItem) []hclsyntax.ObjectConsItem {
	var items []hclsyntax.ObjectConsItem

	if nameExpr != nil {
		items = append(items, hclsyntax.ObjectConsItem{
			KeyExpr:   m.newKeyExpr("name"),
			ValueExpr: nameExpr,
		})
	}

	if teamExpr != nil {
		items = append(items, hclsyntax.ObjectConsItem{
			KeyExpr:   m.newKeyExpr("team"),
			ValueExpr: teamExpr,
		})
	}

	if identityProviderExpr != nil {
		items = append(items, hclsyntax.ObjectConsItem{
			KeyExpr:   m.newKeyExpr("identity_provider_id"),
			ValueExpr: identityProviderExpr,
		})
	}

	items = append(items, otherItems...)
	return items
}

// Helper functions

// getKeyString extracts the string value from a key expression
func (m *V4ToV5Migrator) getKeyString(keyExpr hclsyntax.Expression) string {
	switch k := keyExpr.(type) {
	case *hclsyntax.ObjectConsKeyExpr:
		if k.ForceNonLiteral {
			return ""
		}
		// Extract the key from the wrapped expression
		if scope, ok := k.Wrapped.(*hclsyntax.ScopeTraversalExpr); ok {
			if len(scope.Traversal) > 0 {
				if root, ok := scope.Traversal[0].(hcl.TraverseRoot); ok {
					return root.Name
				}
			}
		}
	case *hclsyntax.ScopeTraversalExpr:
		if len(k.Traversal) > 0 {
			if root, ok := k.Traversal[0].(hcl.TraverseRoot); ok {
				return root.Name
			}
		}
	}
	// Fallback - try to extract from the range
	return ""
}

// newKeyExpr creates a new key expression for an object item
func (m *V4ToV5Migrator) newKeyExpr(key string) hclsyntax.Expression {
	return &hclsyntax.ObjectConsKeyExpr{
		Wrapped: &hclsyntax.ScopeTraversalExpr{
			Traversal: hcl.Traversal{
				hcl.TraverseRoot{Name: key},
			},
		},
	}
}

// exprToTokens converts a syntax expression to write tokens
func (m *V4ToV5Migrator) exprToTokens(expr hclsyntax.Expression) hclwrite.Tokens {
	if expr == nil {
		return nil
	}

	// Directly build tokens from the syntax expression
	return m.buildExprTokens(expr)
}

// syntaxToBytes converts a syntax expression to formatted bytes
func (m *V4ToV5Migrator) syntaxToBytes(expr hclsyntax.Expression) []byte {
	// Build a temporary file to serialize the expression
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	// Recursively build the expression
	tokens := m.buildExprTokens(expr)
	body.SetAttributeRaw("_temp", tokens)

	tempAttr := body.GetAttribute("_temp")
	if tempAttr != nil {
		return tempAttr.Expr().BuildTokens(nil).Bytes()
	}

	return []byte{}
}

// syntaxToBytesAsTokens converts syntax expression to hclwrite tokens
func (m *V4ToV5Migrator) syntaxToBytesAsTokens(expr hclsyntax.Expression) hclwrite.Tokens {
	return m.buildExprTokens(expr)
}

// buildExprTokens recursively builds hclwrite tokens from hclsyntax expression
func (m *V4ToV5Migrator) buildExprTokens(expr hclsyntax.Expression) hclwrite.Tokens {
	var tokens hclwrite.Tokens

	// Handle nil
	if expr == nil {
		return tokens
	}

	switch e := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
		for i, item := range e.Exprs {
			if i > 0 {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}
			tokens = append(tokens, m.buildExprTokens(item)...)
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

	case *hclsyntax.ObjectConsExpr:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
		for i, item := range e.Items {
			if i > 0 {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}
			tokens = append(tokens, m.buildExprTokens(item.KeyExpr)...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
			tokens = append(tokens, m.buildExprTokens(item.ValueExpr)...)
		}
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})

	case *hclsyntax.ObjectConsKeyExpr:
		// For object keys, just use the wrapped expression
		if e.ForceNonLiteral {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOParen, Bytes: []byte("(")})
			tokens = append(tokens, m.buildExprTokens(e.Wrapped)...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
		} else {
			tokens = append(tokens, m.buildExprTokens(e.Wrapped)...)
		}

	case *hclsyntax.ScopeTraversalExpr:
		if len(e.Traversal) > 0 {
			if root, ok := e.Traversal[0].(hcl.TraverseRoot); ok {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(root.Name)})
			}
		}

	case *hclsyntax.LiteralValueExpr:
		// Format the literal value with proper token types
		if e.Val.Type() == cty.String {
			// String literals need to be quoted
			strVal := e.Val.AsString()
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenOQuote,
				Bytes: []byte("\""),
			})
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenQuotedLit,
				Bytes: []byte(strVal),
			})
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenCQuote,
				Bytes: []byte("\""),
			})
		} else if e.Val.Type() == cty.Number {
			bf := e.Val.AsBigFloat()
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenNumberLit,
				Bytes: []byte(bf.Text('f', -1)),
			})
		} else if e.Val.Type() == cty.Bool {
			if e.Val.True() {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("true"),
				})
			} else {
				tokens = append(tokens, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("false"),
				})
			}
		} else {
			// Fallback for other types
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(e.Val.GoString()),
			})
		}

	case *hclsyntax.TemplateExpr:
		// Handle template expressions (interpolations)
		if len(e.Parts) == 1 {
			// Simple template with just a literal
			tokens = append(tokens, m.buildExprTokens(e.Parts[0])...)
		} else {
			// Complex template with interpolations - serialize properly
			// Start the quoted string
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")})

			for _, part := range e.Parts {
				switch p := part.(type) {
				case *hclsyntax.LiteralValueExpr:
					// Literal string part - add as quoted literal
					if p.Val.Type() == cty.String {
						tokens = append(tokens, &hclwrite.Token{
							Type:  hclsyntax.TokenQuotedLit,
							Bytes: []byte(p.Val.AsString()),
						})
					}
				case *hclsyntax.ScopeTraversalExpr:
					// Interpolation - add ${ ... }
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")})
					// Build the traversal
					for i, part := range p.Traversal {
						switch t := part.(type) {
						case hcl.TraverseRoot:
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(t.Name)})
						case hcl.TraverseAttr:
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenDot, Bytes: []byte(".")})
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(t.Name)})
						case hcl.TraverseIndex:
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
							// Handle index key
							if t.Key.Type() == cty.String {
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")})
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(t.Key.AsString())})
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")})
							} else if t.Key.Type() == cty.Number {
								bf := t.Key.AsBigFloat()
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte(bf.String())})
							}
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
						}
						_ = i // Suppress unused warning
					}
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte("}")})
				default:
					// Other expression types in templates - try to serialize recursively
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")})
					tokens = append(tokens, m.buildExprTokens(p)...)
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte("}")})
				}
			}

			// End the quoted string
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")})
		}

	default:
		// For unknown types, try to get the value from the range
		// This is a fallback that shouldn't normally be hit
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte("/* UNKNOWN TYPE */")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("null")})
	}

	return tokens
}




// normalizeIP adds /32 suffix to single IP addresses without CIDR notation
// The Cloudflare API normalizes single IPs to /32 format
func normalizeIP(ip string) string {
	// Check if IP already has CIDR notation (contains '/')
	if !strings.Contains(ip, "/") {
		// Single IP address - add /32 suffix
		return ip + "/32"
	}
	return ip
}

// normalizeIPExpression normalizes IP address literals in HCL expressions
func (m *V4ToV5Migrator) normalizeIPExpression(expr hclsyntax.Expression) hclsyntax.Expression {
	// Check if it's a string literal
	if lit, ok := expr.(*hclsyntax.LiteralValueExpr); ok {
		if lit.Val.Type() == cty.String {
			ipStr := lit.Val.AsString()
			normalizedIP := normalizeIP(ipStr)
			if normalizedIP != ipStr {
				// Return new literal with normalized IP
				return &hclsyntax.LiteralValueExpr{
					Val:      cty.StringVal(normalizedIP),
					SrcRange: lit.SrcRange,
				}
			}
		}
	}
	// Return original expression if not a string literal or already normalized
	return expr
}

// normalizeIPsInSource normalizes IP addresses in HCL source code
// Finds IP addresses in quoted strings and adds /32 suffix if missing
func (m *V4ToV5Migrator) normalizeIPsInSource(src string) string {
	// Regex to match: "X.X.X.X" (IPv4 address in quotes)
	// We'll check in the replacement function if it already has CIDR
	ipPattern := regexp.MustCompile(`"(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(/\d+)?)"`)

	// Replace function that adds /32 to the IP if it doesn't have CIDR
	result := ipPattern.ReplaceAllStringFunc(src, func(match string) string {
		// match is like "192.168.1.1" or "192.168.1.1/24"
		// Extract the IP with CIDR if present (remove quotes)
		ipWithCIDR := match[1 : len(match)-1]
		// Check if it already has CIDR notation
		if strings.Contains(ipWithCIDR, "/") {
			// Already has CIDR, return as-is
			return match
		}
		// Add /32 suffix
		return `"` + ipWithCIDR + `/32"`
	})

	return result
}
