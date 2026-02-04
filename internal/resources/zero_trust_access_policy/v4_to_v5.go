package zero_trust_access_policy

import (
	"regexp"
	"sort"
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

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// This function receives a single instance and needs to return the transformed instance JSON
	result := stateJSON.String()

	// Get attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Even for invalid instances, set schema_version
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Field renames: approval_group → approval_groups
	if attrs.Get("approval_group").Exists() {
		approvalGroupValue := attrs.Get("approval_group").Value()
		result, _ = sjson.Set(result, "attributes.approval_groups", approvalGroupValue)
		result, _ = sjson.Delete(result, "attributes.approval_group")
	}

	// 2. Remove deprecated/invalid fields
	result, _ = sjson.Delete(result, "attributes.application_id")
	result, _ = sjson.Delete(result, "attributes.precedence")
	result, _ = sjson.Delete(result, "attributes.zone_id")
	result, _ = sjson.Delete(result, "attributes.session_duration")

	// 3. Type conversions in approval_groups
	// Convert approvals_needed from int to float64
	if approvalGroups := attrs.Get("approval_groups"); approvalGroups.Exists() && approvalGroups.IsArray() {
		var transformedGroups []map[string]interface{}
		approvalGroups.ForEach(func(k, v gjson.Result) bool {
			group := make(map[string]interface{})

			// Convert approvals_needed to float64
			if approvalsNeeded := v.Get("approvals_needed"); approvalsNeeded.Exists() {
				group["approvals_needed"] = state.ConvertToFloat64(approvalsNeeded)
			}

			// Copy other fields
			if emailAddresses := v.Get("email_addresses"); emailAddresses.Exists() {
				group["email_addresses"] = emailAddresses.Value()
			}
			if emailListUUID := v.Get("email_list_uuid"); emailListUUID.Exists() {
				group["email_list_uuid"] = emailListUUID.Value()
			}

			transformedGroups = append(transformedGroups, group)
			return true
		})

		if len(transformedGroups) > 0 {
			result, _ = sjson.Set(result, "attributes.approval_groups", transformedGroups)
		}
	}

	// 4. Transform connection_rules (MaxItems:1 array → object)
	if connRules := attrs.Get("connection_rules"); connRules.Exists() && connRules.IsArray() {
		arr := connRules.Array()
		if len(arr) > 0 {
			connRulesObj := arr[0]

			// Transform nested ssh from array to object
			transformedConnRules := make(map[string]interface{})

			if ssh := connRulesObj.Get("ssh"); ssh.Exists() && ssh.IsArray() {
				sshArr := ssh.Array()
				if len(sshArr) > 0 {
					sshObj := sshArr[0]

					// Build ssh object
					transformedSSH := make(map[string]interface{})
					if usernames := sshObj.Get("usernames"); usernames.Exists() {
						transformedSSH["usernames"] = usernames.Value()
					}
					if allowEmailAlias := sshObj.Get("allow_email_alias"); allowEmailAlias.Exists() {
						transformedSSH["allow_email_alias"] = allowEmailAlias.Value()
					}

					transformedConnRules["ssh"] = transformedSSH
				}
			}

			// Set the transformed connection_rules as an object
			result, _ = sjson.Set(result, "attributes.connection_rules", transformedConnRules)
		} else {
			// Empty connection_rules array - remove it
			result, _ = sjson.Delete(result, "attributes.connection_rules")
		}
	}

	// 5. Transform include/exclude/require conditions in state
	conditionFields := []string{"include", "exclude", "require"}
	for _, field := range conditionFields {
		if conditionValue := attrs.Get(field); conditionValue.Exists() && conditionValue.IsArray() {
			transformedConditions := m.transformStateConditions(conditionValue)

			// Clean each condition element to remove null nested object fields
			// This prevents drift from empty objects like "email_domain = {}"
			cleanedConditions := m.cleanConditionElements(transformedConditions)

			// Always set the transformed conditions, even if empty (e.g., when all items were "false" booleans)
			// Special handling for empty arrays to ensure they serialize as [] not null
			if len(cleanedConditions) == 0 {
				result, _ = sjson.SetRaw(result, "attributes."+field, "[]")
			} else {
				result, _ = sjson.Set(result, "attributes."+field, cleanedConditions)
			}
		}
	}

	// 6. Clean up zero-value fields to prevent drift
	// Remove empty arrays that the v5 provider normalizes to null
	// Keep include even if empty, but remove empty exclude and require
	emptyArrayFields := []string{
		"attributes.approval_groups",
		"attributes.exclude",
		"attributes.require",
	}
	for _, fieldPath := range emptyArrayFields {
		if value := gjson.Get(result, fieldPath); value.Exists() && value.IsArray() && len(value.Array()) == 0 {
			result, _ = sjson.Delete(result, fieldPath)
		}
	}

	// Remove false boolean fields that the v5 provider normalizes to null
	falseBoolFields := []string{
		"attributes.approval_required",
		"attributes.isolation_required",
		"attributes.purpose_justification_required",
	}
	for _, fieldPath := range falseBoolFields {
		if value := gjson.Get(result, fieldPath); value.Exists() && value.IsBool() && !value.Bool() {
			result, _ = sjson.Delete(result, fieldPath)
		}
	}

	// 7. Final cleanup: Remove null nested object fields from condition elements in the JSON
	// This must be done AFTER sjson.Set has written the arrays, because Terraform
	// re-adds null fields when reading the state
	validEmptyObjects := map[string]bool{
		"everyone":                true,
		"certificate":             true,
		"any_valid_service_token": true,
	}
	conditionFields = []string{"include", "exclude", "require"}
	for _, field := range conditionFields {
		fieldPath := "attributes." + field
		if condArray := gjson.Get(result, fieldPath); condArray.Exists() && condArray.IsArray() {
			var cleanedArray []map[string]interface{}
			condArray.ForEach(func(_, elem gjson.Result) bool {
				cleanedElem := make(map[string]interface{})
				elem.ForEach(func(key, value gjson.Result) bool {
					keyStr := key.String()
					// Only include non-null values
					if value.Exists() && value.Type != gjson.Null {
						// Keep empty objects for boolean conditions
						if value.IsObject() && len(value.Map()) == 0 {
							if validEmptyObjects[keyStr] {
								cleanedElem[keyStr] = value.Value()
							}
							// Skip other empty objects
							return true
						}
						cleanedElem[keyStr] = value.Value()
					}
					return true
				})
				if len(cleanedElem) > 0 {
					cleanedArray = append(cleanedArray, cleanedElem)
				}
				return true
			})
			// Always set the array, even if empty (to preserve empty arrays like [])
			if len(cleanedArray) == 0 {
				// Use SetRaw to ensure empty array is serialized as [] not null
				result, _ = sjson.SetRaw(result, fieldPath, "[]")
			} else {
				result, _ = sjson.Set(result, fieldPath, cleanedArray)
			}
		}
	}

	// Always set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformStateConditions transforms condition arrays in state
// Handles boolean → object conversion and array expansion
func (m *V4ToV5Migrator) transformStateConditions(conditionsArray gjson.Result) []map[string]interface{} {
	var result []map[string]interface{}

	conditionsArray.ForEach(func(_, conditionItem gjson.Result) bool {
		// Each condition item is an object with keys like "everyone", "email", "ip", etc.
		conditionMap := conditionItem.Map()

		// Sort keys alphabetically for deterministic output
		// This matches the v5 provider's normalization behavior
		keys := make([]string, 0, len(conditionMap))
		for key := range conditionMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := conditionMap[key]

			// Skip null values and empty objects entirely
			if !value.Exists() || value.Type == gjson.Null {
				continue
			}
			// Also skip empty objects {}
			if value.IsObject() {
				valueMap := value.Map()
				if len(valueMap) == 0 {
					continue
				}
			}
			// Skip empty arrays []
			if value.IsArray() && len(value.Array()) == 0 {
				continue
			}

			// Handle boolean attributes (everyone, certificate, any_valid_service_token)
			if key == "everyone" || key == "certificate" || key == "any_valid_service_token" {
				if value.IsBool() {
					if value.Bool() {
						// true → {}
						result = append(result, map[string]interface{}{
							key: map[string]interface{}{},
						})
					}
					// false → skip (remove)
					continue
				} else if value.IsObject() {
					// Already an object, keep as-is
					result = append(result, map[string]interface{}{
						key: value.Value(),
					})
					continue
				}
			}

			// Handle array fields that need expansion
			arrayFields := map[string]string{
				"email":        "email",
				"ip":           "ip",
				"email_domain": "domain",
				"geo":          "country_code",
				"group":        "id",
				"login_method": "id",
			}

			if nestedKey, isArrayField := arrayFields[key]; isArrayField {
				if value.IsArray() {
					// Expand array: email = ["a", "b"] → [{email: {email: "a"}}, {email: {email: "b"}}]
					value.ForEach(func(_, arrayItem gjson.Result) bool {
						itemValue := arrayItem.Value()
						// Normalize IP addresses by adding /32 suffix if not present
						if key == "ip" {
							if ipStr, ok := itemValue.(string); ok {
								itemValue = normalizeIP(ipStr)
							}
						}
						result = append(result, map[string]interface{}{
							key: map[string]interface{}{
								nestedKey: itemValue,
							},
						})
						return true
					})
					continue
				} else if value.IsObject() {
					// Check if it's an empty object - skip it
					valueMap := value.Map()
					if len(valueMap) == 0 {
						continue
					}
					// Check if already in v5 format (has the nested key)
					if _, hasNestedKey := valueMap[nestedKey]; hasNestedKey {
						// Already transformed, keep as-is
						// But ONLY include this field, not the whole parent object
						// This prevents multi-field objects from causing drift
						result = append(result, map[string]interface{}{
							key: value.Value(),
						})
						continue
					}
				}
			}

			// Handle MaxItems:1 fields that were arrays in v4 but are single objects in v5
			// These fields should convert array → first element object
			maxItemsOneFields := map[string]bool{
				"auth_context":        true,
				"auth_method":         true,
				"azure_ad":            true,
				"common_name":         true,
				"device_posture":      true,
				"email_list":          true,
				"external_evaluation": true,
				"gsuite":              true,
				"ip_list":             true,
				"linked_app_token":    true,
				"oidc":                true,
				"okta":                true,
				"saml":                true,
				"service_token":       true,
			}

			if maxItemsOneFields[key] {
				if value.IsArray() {
					arr := value.Array()
					if len(arr) > 0 && arr[0].Exists() {
						// Array → take first element as single object
						result = append(result, map[string]interface{}{
							key: arr[0].Value(),
						})
					}
					// Empty array → skip
					continue
				} else if value.IsObject() {
					// Already an object, keep as-is
					result = append(result, map[string]interface{}{
						key: value.Value(),
					})
					continue
				}
			}

			// Handle github_organization special case
			if key == "github_organization" {
				if value.IsObject() {
					org := value.Map()
					name := org["name"]
					identityProviderID := org["identity_provider_id"]
					teams := org["teams"]

					if teams.Exists() && teams.IsArray() {
						// Expand teams
						teams.ForEach(func(_, team gjson.Result) bool {
							newOrg := map[string]interface{}{
								"github_organization": map[string]interface{}{
									"name": name.Value(),
									"team": team.Value(),
								},
							}
							if identityProviderID.Exists() {
								newOrg["github_organization"].(map[string]interface{})["identity_provider_id"] = identityProviderID.Value()
							}
							result = append(result, newOrg)
							return true
						})
					} else {
						// No teams, keep as-is
						newOrg := map[string]interface{}{
							"github_organization": map[string]interface{}{
								"name": name.Value(),
							},
						}
						if identityProviderID.Exists() {
							newOrg["github_organization"].(map[string]interface{})["identity_provider_id"] = identityProviderID.Value()
						}
						result = append(result, newOrg)
					}
					continue
				}
			}

			// For all other fields, keep as-is (wrapped in the key)
			if value.IsObject() {
				// Skip empty objects
				valueMap := value.Map()
				if len(valueMap) > 0 {
					result = append(result, map[string]interface{}{
						key: value.Value(),
					})
				}
			} else if value.IsArray() {
				result = append(result, map[string]interface{}{
					key: value.Value(),
				})
			}
		}

		return true
	})

	return result
}

// cleanConditionElements removes null nested object fields from condition elements
// This prevents drift from empty objects like "email_domain = {}" in plan outputs
// Each condition element should only have ONE top-level key with a populated value
func (m *V4ToV5Migrator) cleanConditionElements(conditions []map[string]interface{}) []map[string]interface{} {
	cleaned := make([]map[string]interface{}, 0, len(conditions))

	// Boolean conditions that are valid as empty objects in v5
	validEmptyObjects := map[string]bool{
		"everyone":                true,
		"certificate":             true,
		"any_valid_service_token": true,
	}

	for _, elem := range conditions {
		cleanedElem := make(map[string]interface{})

		// Iterate through each key in the condition element
		for key, value := range elem {
			// Skip null values
			if value == nil {
				continue
			}

			// Check if this is an empty map
			if valueMap, ok := value.(map[string]interface{}); ok && len(valueMap) == 0 {
				// Keep empty objects for boolean conditions (everyone, certificate, etc.)
				if validEmptyObjects[key] {
					cleanedElem[key] = value
				}
				// Skip other empty objects
				continue
			}

			// Keep non-null, non-empty values
			cleanedElem[key] = value
		}

		// Only add the element if it has at least one populated field
		if len(cleanedElem) > 0 {
			cleaned = append(cleaned, cleanedElem)
		}
	}

	return cleaned
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
