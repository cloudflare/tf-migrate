package address_map

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

const resourceType = "cloudflare_address_map"

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	m := &V4ToV5Migrator{}
	internal.RegisterMigrator(resourceType, "v4", "v5", m)
	return m
}

func (*V4ToV5Migrator) GetResourceType() string    { return resourceType }
func (*V4ToV5Migrator) CanHandle(t string) bool    { return t == resourceType }
func (*V4ToV5Migrator) Preprocess(s string) string { return s }

func (*V4ToV5Migrator) GetInvalidAttributeReferences() []transform.InvalidAttributeReference {
	return []transform.InvalidAttributeReference{
		{ResourceType: resourceType, Attribute: "ips", ReviewOnly: true,
			Suggestion: "In v5, ips contains strings, not objects. Remove .ip projections from expressions that read the entries."},
		{ResourceType: resourceType, Attribute: "memberships", ReviewOnly: true,
			Suggestion: "In v5, membership entries contain only identifier and kind. Review consumers that relied on the removed per-membership can_delete field."},
	}
}

func (*V4ToV5Migrator) TransformConfig(_ *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	// Build both replacements before mutating the resource so malformed input
	// cannot leave a partially converted address map.
	ips, ipBlocks, err := collection(body, "ips", "ip")
	if err != nil {
		return nil, err
	}
	members, memberBlocks, err := collection(body, "memberships", "identifier", "kind")
	if err != nil {
		return nil, err
	}
	replacements := hclwrite.NewEmptyFile().Body()
	for _, attr := range []struct{ name, expression string }{{"ips", ips}, {"memberships", members}} {
		if attr.expression != "" {
			if err := tfhcl.SetAttributeFromExpressionString(replacements, attr.name, attr.expression); err != nil {
				return nil, fmt.Errorf("address map %s: %w", attr.name, err)
			}
		}
	}
	for _, old := range append(ipBlocks, memberBlocks...) {
		body.RemoveBlock(old)
	}
	for _, attr := range tfhcl.AttributesOrdered(replacements) {
		body.SetAttributeRaw(attr.Name, attr.Attribute.Expr().BuildTokens(nil))
	}
	return &transform.TransformResult{Blocks: []*hclwrite.Block{block}}, nil
}

func collection(body *hclwrite.Body, name string, fields ...string) (string, []*hclwrite.Block, error) {
	var blocks []*hclwrite.Block
	var parts, static []string
	flushStatic := func() {
		if len(static) > 0 {
			parts = append(parts, "["+strings.Join(static, ", ")+"]")
			static = nil
		}
	}
	dynamic := false
	staticCount := 0
	for _, block := range body.Blocks() {
		labels := block.Labels()
		isDynamic := block.Type() == "dynamic" && len(labels) == 1 && labels[0] == name
		if block.Type() != name && !isDynamic {
			continue
		}
		if body.GetAttribute(name) != nil {
			return "", nil, fmt.Errorf("address map has both a %s attribute and blocks", name)
		}
		blocks = append(blocks, block)
		if isDynamic {
			flushStatic()
			dynamic = true
			expr, err := dynamicList(block, name, fields)
			if err != nil {
				return "", nil, fmt.Errorf("address map dynamic %s: %w", name, err)
			}
			parts = append(parts, expr)
			continue
		}
		item, err := listItem(block.Body(), fields)
		if err != nil {
			return "", nil, fmt.Errorf("address map %s: %w", name, err)
		}
		static = append(static, item)
		staticCount++
	}
	flushStatic()
	if body.GetAttribute(name) != nil {
		return "", nil, nil // Already in v5 form.
	}
	if len(parts) == 0 {
		return "[]", nil, nil // v4 stored omitted sets as empty collections.
	}
	expr := parts[0]
	if len(parts) > 1 {
		expr = "concat(" + strings.Join(parts, ", ") + ")"
	}
	// The v4 provider treated both collections as sets, including when
	// duplicates only become visible after evaluating variables.
	if dynamic || staticCount > 1 {
		expr = "distinct(" + expr + ")"
	}
	return expr, blocks, nil
}

func listItem(body *hclwrite.Body, fields []string) (string, error) {
	if len(body.Blocks()) != 0 {
		return "", fmt.Errorf("unexpected nested blocks")
	}
	var values []string
	for _, field := range fields {
		attr := body.GetAttribute(field)
		if attr == nil {
			return "", fmt.Errorf("missing %s", field)
		}
		value := string(attr.Expr().BuildTokens(nil).Bytes())
		if len(fields) == 1 {
			return value, nil
		}
		values = append(values, field+" = "+value)
	}
	return "{ " + strings.Join(values, ", ") + " }", nil
}

func dynamicList(block *hclwrite.Block, iterator string, fields []string) (string, error) {
	body := block.Body()
	forEach := body.GetAttribute("for_each")
	contents := tfhcl.FindBlocksByType(body, "content")
	if forEach == nil || len(contents) != 1 {
		return "", fmt.Errorf("expected for_each and one content block")
	}
	if attr := body.GetAttribute("iterator"); attr != nil {
		iterator = strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
		if !hclsyntax.ValidIdentifier(iterator) {
			return "", fmt.Errorf("invalid iterator %q", iterator)
		}
	}
	item, err := listItem(contents[0].Body(), fields)
	if err != nil {
		return "", err
	}
	// Choose names absent from the item expression, including nested loop
	// bindings, so rebinding the dynamic iterator cannot capture a variable.
	key, value := "address_map_key", "address_map_value"
	for strings.Contains(item, key) || strings.Contains(item, value) {
		key += "_"
		value += "_"
	}
	src := []byte(item)
	expression, diags := hclsyntax.ParseExpression(src, "", hcl.InitialPos)
	if diags.HasErrors() {
		return "", diags
	}
	// Variables returns free traversals, excluding nested expressions that
	// rebind the iterator. Replace ranges backwards to keep offsets valid.
	vars := expression.Variables()
	sort.Slice(vars, func(i, j int) bool { return vars[i].SourceRange().Start.Byte < vars[j].SourceRange().Start.Byte })
	for i := len(vars) - 1; i >= 0; i-- {
		ref := vars[i]
		if ref.RootName() != iterator {
			continue
		}
		start, end := ref[0].SourceRange().Start.Byte, ref[0].SourceRange().End.Byte
		replacement := fmt.Sprintf("({key = %s, value = %s})", key, value)
		if len(ref) > 1 {
			if attr, ok := ref[1].(hcl.TraverseAttr); ok && (attr.Name == "key" || attr.Name == "value") {
				end = attr.SrcRange.End.Byte
				replacement = value
				if attr.Name == "key" {
					replacement = key
				}
			}
		}
		item = item[:start] + replacement + item[end:]
	}
	return fmt.Sprintf("[for %s, %s in %s : %s]", key, value, forEach.Expr().BuildTokens(nil).Bytes(), item), nil
}
