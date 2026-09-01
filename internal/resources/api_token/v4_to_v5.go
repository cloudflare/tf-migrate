package api_token

import (
	"fmt"
	"sort"

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

// transformPermissionGroups converts permission_groups from v4 format to v5 format.
//
// v4 accepts:
//   - A list of bare string IDs:   ["id1", "id2"]
//   - A list of objects:           [{ id = "id1" }, { id = "id2" }]
//   - A list of data-source refs:  [data.cloudflare_api_token_permission_groups.all.permissions["Name"]]
//
// v5 requires a list of objects:  [{ id = "id1" }, { id = "id2" }]
//
// For string / object elements the IDs are sorted alphabetically to match
// the v5 provider's canonical ordering.
//
// Expression elements that reference cloudflare_api_token_permission_groups are
// transformed to for-expressions that look up the ID by name from the v5
// replacement data source (cloudflare_api_token_permission_groups_list).
func (m *V4ToV5Migrator) transformPermissionGroups(ctx *transform.Context, body *hclwrite.Body) {
	permGroupsAttr := body.GetAttribute("permission_groups")
	if permGroupsAttr == nil {
		return
	}

	exprTokens := permGroupsAttr.Expr().BuildTokens(nil)

	// Scan for string IDs using bracket-depth tracking.
	// Only TokenQuotedLit tokens at bracketDepth == 1 are direct string values;
	// tokens at deeper depths (e.g. ["key"] index access on a data source) are
	// NOT collected — this is the fix for the bug where "DNS Read" (the map key
	// in data.source.permissions["DNS Read"]) was incorrectly extracted as an ID.
	var permIDs []string
	bracketDepth := 0
	for _, tok := range exprTokens {
		switch tok.Type {
		case hclsyntax.TokenOBrack:
			bracketDepth++
		case hclsyntax.TokenCBrack:
			bracketDepth--
		case hclsyntax.TokenQuotedLit:
			if bracketDepth == 1 {
				permIDs = append(permIDs, string(tok.Bytes))
			}
		}
	}

	if len(permIDs) > 0 {
		// All elements are string literals or already-objects with string IDs.
		// Sort alphabetically to match the v5 provider's canonical ordering.
		sort.Strings(permIDs)

		var permObjects []hclwrite.Tokens
		for _, id := range permIDs {
			objAttrs := []hclwrite.ObjectAttrTokens{
				{
					Name:  hclwrite.TokensForIdentifier("id"),
					Value: hclwrite.TokensForValue(cty.StringVal(id)),
				},
			}
			permObjects = append(permObjects, hclwrite.TokensForObject(objAttrs))
		}

		body.RemoveAttribute("permission_groups")
		body.SetAttributeRaw("permission_groups", hclwrite.TokensForTuple(permObjects))
		return
	}

	// No string IDs found at depth 1.  The list likely contains expression
	// references such as data-source lookups.  Split the list into individual
	// elements and try to transform data source references into for-expressions.
	elements := splitPermGroupExprElements(exprTokens)
	if len(elements) == 0 {
		return
	}

	// Try to parse each element as a cloudflare_api_token_permission_groups
	// data source reference and convert it to a for-expression against the
	// v5 replacement data source.
	var forExprParts []string
	allParsed := true
	for _, elem := range elements {
		ref := parsePermGroupDataSourceRef(elem)
		if ref == nil {
			allParsed = false
			break
		}
		// Build: { id = [for pg in data.cloudflare_api_token_permission_groups_list.<label>.result : pg.id if pg.name == "<name>"][0] }
		forExpr := fmt.Sprintf(
			`{ id = [for pg in data.cloudflare_api_token_permission_groups_list.%s.result : pg.id if pg.name == %q][0] }`,
			ref.label, ref.permName,
		)
		forExprParts = append(forExprParts, forExpr)
	}

	if allParsed && len(forExprParts) > 0 {
		// Build the full permission_groups value as an expression string and
		// parse it through hclwrite so the tokens are well-formed.
		exprStr := "["
		for i, part := range forExprParts {
			if i > 0 {
				exprStr += ", "
			}
			exprStr += part
		}
		exprStr += "]"

		if err := tfhcl.SetAttributeFromExpressionString(body, "permission_groups", exprStr); err == nil {
			return
		}
		// Fall through to the generic wrapping if parsing fails
	}

	// Fallback: wrap each element as { id = <expr> } and emit a warning.
	ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Manual action required: permission_groups contains unrecognized expressions",
		Detail: "Some permission_groups values could not be automatically migrated. " +
			"Replace the expression references with actual permission group ID values. " +
			"Use the cloudflare_api_token_permission_groups_list data source to look up IDs by name.",
	})

	var permObjects []hclwrite.Tokens
	for _, elem := range elements {
		permObjects = append(permObjects, buildPermGroupIDObject(elem))
	}

	body.RemoveAttribute("permission_groups")
	body.SetAttributeRaw("permission_groups", hclwrite.TokensForTuple(permObjects))
}

// permGroupRef holds the parsed components of a cloudflare_api_token_permission_groups
// data source reference expression.
type permGroupRef struct {
	label    string // e.g. "all"
	permName string // e.g. "DNS Read"
}

// parsePermGroupDataSourceRef attempts to parse an expression token stream as a
// reference to data.cloudflare_api_token_permission_groups.<label>.<category>["<name>"].
// Returns nil if the tokens don't match this pattern.
func parsePermGroupDataSourceRef(tokens hclwrite.Tokens) *permGroupRef {
	// Collect the identifiers and the quoted string from the expression.
	// Expected token pattern (ignoring whitespace):
	//   data . cloudflare_api_token_permission_groups . <label> . <category> [ " <name> " ]
	//
	// We need: ident("data"), dot, ident("cloudflare_api_token_permission_groups"),
	//          dot, ident(<label>), dot, ident(<category>), [, ", <name>, ", ]
	var idents []string
	var quotedLit string
	foundBracket := false

	for _, tok := range tokens {
		switch tok.Type {
		case hclsyntax.TokenIdent:
			if !foundBracket {
				idents = append(idents, string(tok.Bytes))
			}
		case hclsyntax.TokenDot:
			// skip dots
		case hclsyntax.TokenOBrack:
			foundBracket = true
		case hclsyntax.TokenQuotedLit:
			if foundBracket {
				quotedLit = string(tok.Bytes)
			}
		case hclsyntax.TokenNewline:
			// skip
		case hclsyntax.TokenOQuote, hclsyntax.TokenCQuote, hclsyntax.TokenCBrack:
			// skip quote/bracket delimiters
		}
	}

	// Validate the pattern: data, cloudflare_api_token_permission_groups, <label>, <category>
	if len(idents) != 4 || idents[0] != "data" || idents[1] != "cloudflare_api_token_permission_groups" {
		return nil
	}
	if quotedLit == "" {
		return nil
	}

	return &permGroupRef{
		label:    idents[2],
		permName: quotedLit,
	}
}

// splitPermGroupExprElements splits an HCL list expression token stream into
// individual element token groups.  Depth is tracked across '[', '{', and '('
// so that nested structures (index access, objects, function calls) are not
// mistakenly treated as element boundaries.  Returns nil when no outer '[' is found.
func splitPermGroupExprElements(exprTokens hclwrite.Tokens) []hclwrite.Tokens {
	outerStart := -1
	for i, tok := range exprTokens {
		if tok.Type == hclsyntax.TokenOBrack {
			outerStart = i
			break
		}
	}
	if outerStart == -1 {
		return nil
	}

	var elements []hclwrite.Tokens
	depth := 0
	elemStart := outerStart + 1

	for i := outerStart; i < len(exprTokens); i++ {
		tok := exprTokens[i]
		switch tok.Type {
		case hclsyntax.TokenOBrack, hclsyntax.TokenOBrace, hclsyntax.TokenOParen:
			depth++
		case hclsyntax.TokenCBrack:
			depth--
			if depth == 0 {
				// End of the outer list.
				elem := trimPermGroupElement(exprTokens[elemStart:i])
				if len(elem) > 0 {
					elements = append(elements, elem)
				}
				return elements
			}
		case hclsyntax.TokenCBrace, hclsyntax.TokenCParen:
			depth--
		case hclsyntax.TokenComma:
			if depth == 1 {
				elem := trimPermGroupElement(exprTokens[elemStart:i])
				if len(elem) > 0 {
					elements = append(elements, elem)
				}
				elemStart = i + 1
			}
		}
	}
	return elements
}

// trimPermGroupElement strips leading and trailing newline tokens from a token slice.
func trimPermGroupElement(tokens hclwrite.Tokens) hclwrite.Tokens {
	start := 0
	for start < len(tokens) && tokens[start].Type == hclsyntax.TokenNewline {
		start++
	}
	end := len(tokens)
	for end > start && tokens[end-1].Type == hclsyntax.TokenNewline {
		end--
	}
	return tokens[start:end]
}

// buildPermGroupIDObject wraps expression tokens as { id = <tokens> }.
func buildPermGroupIDObject(valueTokens hclwrite.Tokens) hclwrite.Tokens {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("  id")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
	}
	tokens = append(tokens, valueTokens...)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
	)
	return tokens
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
