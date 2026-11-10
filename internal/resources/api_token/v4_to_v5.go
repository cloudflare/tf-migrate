package api_token

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_api_token", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_api_token"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_api_token"
}

// Preprocess handles string-level transformations before HCL parsing.
// The cloudflare_api_token_permission_groups data source is now handled
// by the dedicated datasource migrator in internal/datasources/api_token_permission_groups.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// This resource does not rename, so we return the same name for both old and new.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_api_token"}, "cloudflare_api_token"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	m.transformPolicyBlocks(ctx, body)
	m.transformConditionBlock(body)
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) transformPolicyBlocks(ctx *transform.Context, body *hclwrite.Body) {
	policyBlocks := tfhcl.FindBlocksByType(body, "policy")
	if len(policyBlocks) == 0 {
		return
	}

	var policyObjects []hclwrite.Tokens
	for _, policyBlock := range policyBlocks {
		policyBody := policyBlock.Body()

		// Ensure effect attribute exists (required in v5, optional in v4)
		if policyBody.GetAttribute("effect") == nil {
			// Default to "allow" if not specified
			policyBody.SetAttributeValue("effect", cty.StringVal("allow"))
		}

		// Transform permission_groups from list of strings to list of objects with id field
		m.transformPermissionGroups(ctx, policyBody)

		// Transform resources from map to jsonencode() wrapped map
		// v4: resources = { "com.cloudflare.api.account.*" = "*" }
		// v5: resources = jsonencode({ "com.cloudflare.api.account.*" = "*" })
		m.transformResources(policyBody)

		objTokens := tfhcl.BuildObjectFromBlock(policyBlock)
		policyObjects = append(policyObjects, objTokens)
	}

	listTokens := hclwrite.TokensForTuple(policyObjects)
	body.SetAttributeRaw("policies", listTokens)

	tfhcl.RemoveBlocksByType(body, "policy")
}

var permissionGroupScopes = map[string]string{
	"permissions": "",
	"zone":        "com.cloudflare.api.account.zone",
	"account":     "com.cloudflare.api.account",
	"user":        "com.cloudflare.api.user",
	"r2":          "com.cloudflare.edge.r2.bucket",
}

// Parse the whole expression before inspecting elements: quoted map keys are
// not IDs, and finding one literal must never discard the remaining elements.
func (m *V4ToV5Migrator) transformPermissionGroups(ctx *transform.Context, body *hclwrite.Body) {
	attr := body.GetAttribute("permission_groups")
	if attr == nil {
		return
	}
	src := attr.Expr().BuildTokens(nil).Bytes()
	expr, diags := hclsyntax.ParseExpression(src, ctx.Filename, hcl.InitialPos)
	if diags.HasErrors() {
		ctx.Diagnostics = append(ctx.Diagnostics, diags...)
		return
	}
	var result string
	needsReview := false
	switch expr := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		if len(expr.Exprs) == 0 {
			return
		}
		parts := make([]string, 0, len(expr.Exprs))
		allLiteral := true
		for _, element := range expr.Exprs {
			part, literal, review := permissionGroupObject(element, src)
			parts = append(parts, part)
			allLiteral = allLiteral && literal
			needsReview = needsReview || review
		}
		if allLiteral {
			sort.Strings(parts)
			// Keep the established multiline formatting and attribute position
			// for literal-only configurations.
			var objects []hclwrite.Tokens
			for _, part := range parts {
				parsed, _ := hclsyntax.ParseExpression([]byte(part), "", hcl.InitialPos)
				value, _ := parsed.Value(nil)
				objects = append(objects, hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{{
					Name: hclwrite.TokensForIdentifier("id"), Value: hclwrite.TokensForValue(value.GetAttr("id")),
				}}))
			}
			body.RemoveAttribute("permission_groups")
			body.SetAttributeRaw("permission_groups", hclwrite.TokensForTuple(objects))
			return
		}
		result = "[" + strings.Join(parts, ", ") + "]"
	case *hclsyntax.ForExpr:
		if expr.KeyExpr != nil {
			// An object comprehension is not a list of permission IDs. Preserve
			// it for manual resolution rather than inventing a collection shape.
			result = string(src)
			needsReview = true
			break
		}
		part, _, review := permissionGroupObject(expr.ValExpr, src)
		r := expr.ValExpr.Range()
		result = string(src[:r.Start.Byte]) + part + string(src[r.End.Byte:])
		needsReview = review
	default:
		// A variable or function may supply the entire v4 collection. Keep
		// the collection expression intact and wrap its IDs at evaluation time.
		result = fmt.Sprintf("[for permission_group_id in (%s) : { id = permission_group_id }]", src)
		needsReview = true
	}
	if err := tfhcl.SetAttributeFromExpressionString(body, "permission_groups", result); err != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to migrate permission_groups",
			Detail:   err.Error(),
		})
		return
	}
	if needsReview {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Manual action required: permission_groups contains unrecognized expressions",
			Detail:   fmt.Sprintf("In %s: permission_groups expressions were preserved. Verify that each id evaluates to a permission group ID. Replace remaining cloudflare_api_token_permission_groups map lookups with name and scope lookups against cloudflare_api_token_permission_groups_list.result.", ctx.Filename),
		})
	}
}

// Returns the object expression, whether it is a literal ID, and whether it
// needs manual review. Already-object values must not be wrapped a second time.
func permissionGroupObject(expr hclsyntax.Expression, src []byte) (string, bool, bool) {
	if parens, ok := expr.(*hclsyntax.ParenthesesExpr); ok {
		return permissionGroupObject(parens.Expression, src)
	}
	if object, ok := expr.(*hclsyntax.ObjectConsExpr); ok {
		value, diags := object.Value(nil)
		if !diags.HasErrors() && value.IsWhollyKnown() && value.Type().IsObjectType() && value.Type().HasAttribute("id") && len(value.Type().AttributeTypes()) == 1 {
			id := value.GetAttr("id")
			if !id.IsNull() && id.Type() == cty.String {
				return "{ id = " + string(hclwrite.TokensForValue(id).Bytes()) + " }", true, false
			}
		}
		return string(expr.Range().SliceBytes(src)), false, true
	}
	if value, diags := expr.Value(nil); !diags.HasErrors() && value.IsKnown() && !value.IsNull() && value.Type() == cty.String {
		return "{ id = " + string(hclwrite.TokensForValue(value).Bytes()) + " }", true, false
	}
	if lookup, ok := permissionGroupLookup(expr, src); ok {
		return "{ id = " + lookup + " }", false, false
	}
	return "{ id = (" + string(expr.Range().SliceBytes(src)) + ") }", false, true
}

// Match data.<old type>.<label>[optional instance].<category>[<key>].
// Preserve count/for_each instance selectors while changing the output shape.
func permissionGroupLookup(expr hclsyntax.Expression, src []byte) (string, bool) {
	path, ok := tfhcl.TraversalPath(expr, src)
	if !ok || (len(path) != 5 && len(path) != 6) || path[0].Name != "data" || path[1].Name != "cloudflare_api_token_permission_groups" || path[2].Name == "" {
		return "", false
	}
	instance := path[2].Name
	category := 3
	if len(path) == 6 {
		if path[3].Index == "" {
			return "", false
		}
		instance += "[" + path[3].Index + "]"
		category++
	}
	scope, ok := permissionGroupScopes[path[category].Name]
	key := path[category+1].Index
	if !ok || key == "" {
		return "", false
	}
	// The key may itself refer to a for variable named pg. Choose a lookup
	// iterator that cannot shadow any variable used by the key expression.
	iterator := "pg"
	for strings.Contains(key, iterator) {
		iterator += "_"
	}
	filter := fmt.Sprintf("%s.name == (%s)", iterator, key)
	if scope != "" {
		filter += fmt.Sprintf(" && contains(%s.scopes, %q)", iterator, scope)
	}
	return fmt.Sprintf("one([for %s in data.cloudflare_api_token_permission_groups_list.%s.result : %s.id if %s])", iterator, instance, iterator, filter), true
}

// transformResources wraps the resources map with jsonencode()
// v4: resources = { "com.cloudflare.api.account.*" = "*" }
// v5: resources = jsonencode({ "com.cloudflare.api.account.*" = "*" })
func (m *V4ToV5Migrator) transformResources(body *hclwrite.Body) {
	resourcesAttr := body.GetAttribute("resources")
	if resourcesAttr == nil {
		return
	}

	// Get the existing expression tokens (the map value)
	exprTokens := resourcesAttr.Expr().BuildTokens(nil)

	// Build jsonencode( ... ) wrapper
	// Start with "jsonencode("
	var newTokens hclwrite.Tokens
	newTokens = append(newTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("jsonencode"),
	})
	newTokens = append(newTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOParen,
		Bytes: []byte("("),
	})

	// Add the original map expression
	newTokens = append(newTokens, exprTokens...)

	// Close with ")"
	newTokens = append(newTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCParen,
		Bytes: []byte(")"),
	})

	// Replace the attribute with the wrapped version
	body.SetAttributeRaw("resources", newTokens)
}

func (m *V4ToV5Migrator) transformConditionBlock(body *hclwrite.Body) {
	conditionBlock := tfhcl.FindBlockByType(body, "condition")
	if conditionBlock == nil {
		return
	}

	var conditionAttrs []hclwrite.ObjectAttrTokens
	for _, nestedBlock := range conditionBlock.Body().Blocks() {
		if nestedBlock.Type() == "request_ip" {
			requestIPTokens := tfhcl.BuildObjectFromBlock(nestedBlock)
			conditionAttrs = append(conditionAttrs, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier("request_ip"),
				Value: requestIPTokens,
			})
		}
	}

	conditionTokens := hclwrite.TokensForObject(conditionAttrs)
	body.SetAttributeRaw("condition", conditionTokens)
	body.RemoveBlock(conditionBlock)
}
