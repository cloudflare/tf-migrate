package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// PathStep is a root/attribute name or an index expression (without brackets).
// Range identifies the step in the original expression source.
type PathStep struct {
	Name  string
	Index string
	Range hcl.Range
}

// TraversalPath flattens static and dynamic traversal/index AST nodes. It does
// not treat function calls, operators, or splats as direct references.
func TraversalPath(expr hclsyntax.Expression, src []byte) ([]PathStep, bool) {
	var path []PathStep
	var traversal hcl.Traversal
	switch expr := expr.(type) {
	case *hclsyntax.ParenthesesExpr:
		return TraversalPath(expr.Expression, src)
	case *hclsyntax.ScopeTraversalExpr:
		traversal = expr.Traversal
	case *hclsyntax.RelativeTraversalExpr:
		var ok bool
		path, ok = TraversalPath(expr.Source, src)
		if !ok {
			return nil, false
		}
		traversal = expr.Traversal
	case *hclsyntax.IndexExpr:
		path, ok := TraversalPath(expr.Collection, src)
		if !ok {
			return nil, false
		}
		return append(path, PathStep{Index: string(expr.Key.Range().SliceBytes(src)), Range: expr.Range()}), true
	default:
		return nil, false
	}
	for _, step := range traversal {
		switch step := step.(type) {
		case hcl.TraverseRoot:
			path = append(path, PathStep{Name: step.Name, Range: step.SrcRange})
		case hcl.TraverseAttr:
			path = append(path, PathStep{Name: step.Name, Range: step.SrcRange})
		case hcl.TraverseIndex:
			path = append(path, PathStep{Index: string(hclwrite.TokensForValue(step.Key).Bytes()), Range: step.SrcRange})
		default:
			return nil, false
		}
	}
	return path, true
}
