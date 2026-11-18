package hcl

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// InstanceAttributeRef is a reference to an attribute of an instance of a
// resource or data source type, found by FindInstanceAttributeReferences.
type InstanceAttributeRef struct {
	Type      string    // resource type as passed in, e.g. "data.cloudflare_dcv_delegation"
	Attribute string    // attribute name, e.g. "id"
	NameRange hcl.Range // the attribute name only, without the leading "."
	Range     hcl.Range // the whole reference, for messages
}

// FindInstanceAttributeReferences returns every reference in body to an
// attribute of an instance of one of types. types are written as in
// configuration: "cloudflare_x" or "data.cloudflare_x".
//
// Besides direct references (type.name.attr), it follows the forms that reach
// an instance through an index, a splat, or an iteration variable:
//
//	type.name[0].attr  type.name["k"].attr  type.name[each.key].attr
//	type.name[*].attr  type.name.*.attr
//	[for v in type.name : v.attr]
//	for_each = type.name  ...  each.value.attr
//	dynamic "x" { for_each = type.name  content { ... x.value.attr } }
//
// Iteration variables are only followed when the collection is exactly
// type.name; anything derived from it (a function call, a for expression,
// a local) is not followed.
//
// iterable reports whether type.name is declared with count or for_each. An
// index or iteration over a resource without either reaches its attributes,
// not its instances, so those forms are only followed when iterable returns
// true. A nil iterable treats every resource as iterable.
func FindInstanceAttributeReferences(body *hclsyntax.Body, src []byte, types []string, iterable func(resourceType, name string) bool) []InstanceAttributeRef {
	if iterable == nil {
		iterable = func(string, string) bool { return true }
	}
	f := &instanceRefFinder{src: src, seen: make(map[[2]int]bool), iterable: iterable}
	for _, t := range types {
		f.types = append(f.types, strings.Split(t, "."))
	}
	f.body(body, "", "")
	sort.Slice(f.refs, func(i, j int) bool { return f.refs[i].NameRange.Start.Byte < f.refs[j].NameRange.Start.Byte })
	return f.refs
}

type instanceRefFinder struct {
	src      []byte
	types    [][]string
	refs     []InstanceAttributeRef
	seen     map[[2]int]bool
	iterable func(resourceType, name string) bool
}

// body visits every expression in b. eachType is the resource type that
// each.value refers to inside b, or "" when it does not refer to one.
// forEachType is the same for b's own for_each attribute, which is evaluated
// outside the iteration it defines.
func (f *instanceRefFinder) body(b *hclsyntax.Body, eachType, forEachType string) {
	if b == nil {
		return
	}
	for name, attr := range b.Attributes {
		if name == "for_each" {
			f.expr(attr.Expr, forEachType)
			continue
		}
		f.expr(attr.Expr, eachType)
	}
	for _, block := range b.Blocks {
		inner := eachType
		// Only top-level blocks (resource, data, module) bind each. A dynamic
		// block's for_each binds its own iterator, and each keeps referring to
		// the enclosing resource.
		if fe, ok := block.Body.Attributes["for_each"]; ok && block.Type != "dynamic" {
			inner, _ = f.wholeCollection(fe.Expr)
		}
		f.body(block.Body, inner, eachType)

		// dynamic "x" { for_each = type.name ... } binds x.value (or the
		// iterator attribute's name) to each instance.
		if block.Type == "dynamic" && len(block.Labels) == 1 {
			fe, ok := block.Body.Attributes["for_each"]
			if !ok {
				continue
			}
			t, ok := f.wholeCollection(fe.Expr)
			if !ok {
				continue
			}
			iter := block.Labels[0]
			if it, ok := block.Body.Attributes["iterator"]; ok {
				if path, ok := TraversalPath(it.Expr, f.src); ok && len(path) == 1 {
					iter = path[0].Name
				}
			}
			for _, content := range block.Body.Blocks {
				if content.Type == "content" {
					f.loopVarBody(content.Body, t, []string{iter, "value"})
				}
			}
		}
	}
}

// loopVarBody applies loopVar to every expression in b, including nested blocks.
func (f *instanceRefFinder) loopVarBody(b *hclsyntax.Body, t string, prefix []string) {
	for _, attr := range b.Attributes {
		f.loopVar(attr.Expr, t, prefix)
	}
	for _, block := range b.Blocks {
		f.loopVarBody(block.Body, t, prefix)
	}
}

func (f *instanceRefFinder) expr(e hclsyntax.Expression, eachType string) {
	_ = hclsyntax.VisitAll(e, func(node hclsyntax.Node) hcl.Diagnostics {
		switch n := node.(type) {
		case *hclsyntax.SplatExpr:
			// A splat over a single object wraps it in a one-element list,
			// so it reaches an instance whether or not the resource is iterable.
			if t, ok := f.wholeResource(n.Source); ok {
				f.splat(t, n)
			}
		case *hclsyntax.ForExpr:
			if t, ok := f.wholeCollection(n.CollExpr); ok {
				for _, sub := range []hclsyntax.Expression{n.KeyExpr, n.ValExpr, n.CondExpr} {
					if sub != nil {
						f.loopVar(sub, t, []string{n.ValVar})
					}
				}
			}
		case hclsyntax.Expression:
			f.direct(n)
		}
		return nil
	})
	if eachType != "" {
		f.loopVar(e, eachType, []string{"each", "value"})
	}
}

// matchType reports which type path begins with, and the index of the step
// after the type (the resource name).
func (f *instanceRefFinder) matchType(path []PathStep) (string, int, bool) {
	for _, parts := range f.types {
		if len(path) <= len(parts) {
			continue
		}
		ok := true
		for i, p := range parts {
			ok = ok && path[i].Name == p && path[i].Index == ""
		}
		if ok && path[len(parts)].Name != "" {
			return strings.Join(parts, "."), len(parts), true
		}
	}
	return "", 0, false
}

// wholeCollection reports whether e is exactly type.name, with no index or
// attribute after it, and type.name is iterable.
func (f *instanceRefFinder) wholeCollection(e hclsyntax.Expression) (string, bool) {
	t, name, ok := f.wholeRef(e)
	if !ok || !f.iterable(t, name) {
		return "", false
	}
	return t, true
}

// wholeResource reports whether e is exactly type.name.
func (f *instanceRefFinder) wholeResource(e hclsyntax.Expression) (string, bool) {
	t, _, ok := f.wholeRef(e)
	return t, ok
}

func (f *instanceRefFinder) wholeRef(e hclsyntax.Expression) (string, string, bool) {
	path, ok := TraversalPath(e, f.src)
	if !ok {
		return "", "", false
	}
	t, nameIdx, ok := f.matchType(path)
	if !ok || len(path) != nameIdx+1 {
		return "", "", false
	}
	return t, path[nameIdx].Name, true
}

// direct handles type.name.attr and type.name[key].attr.
func (f *instanceRefFinder) direct(e hclsyntax.Expression) {
	path, ok := TraversalPath(e, f.src)
	if !ok {
		return
	}
	t, nameIdx, ok := f.matchType(path)
	if !ok {
		return
	}
	attr := nameIdx + 1
	if attr < len(path) && path[attr].Index != "" {
		if !f.iterable(t, path[nameIdx].Name) {
			return
		}
		attr++
	}
	if attr >= len(path) || path[attr].Name == "" {
		return
	}
	f.add(t, path[attr], e.Range())
}

// splat handles type.name[*].attr and type.name.*.attr.
func (f *instanceRefFinder) splat(t string, s *hclsyntax.SplatExpr) {
	rel, ok := s.Each.(*hclsyntax.RelativeTraversalExpr)
	if !ok || len(rel.Traversal) == 0 {
		return
	}
	if _, ok := rel.Source.(*hclsyntax.AnonSymbolExpr); !ok {
		return
	}
	step, ok := rel.Traversal[0].(hcl.TraverseAttr)
	if !ok {
		return
	}
	f.add(t, PathStep{Name: step.Name, Range: step.SrcRange}, s.Range())
}

// loopVar handles <prefix>.attr, where prefix (a for variable, or each.value)
// is an instance of t. Inner for expressions that rebind the variable are not
// treated specially; rebinding a loop variable to a different value is rare.
func (f *instanceRefFinder) loopVar(e hclsyntax.Expression, t string, prefix []string) {
	_ = hclsyntax.VisitAll(e, func(node hclsyntax.Node) hcl.Diagnostics {
		ex, ok := node.(hclsyntax.Expression)
		if !ok {
			return nil
		}
		path, ok := TraversalPath(ex, f.src)
		if !ok || len(path) <= len(prefix) {
			return nil
		}
		for i, p := range prefix {
			if path[i].Name != p || path[i].Index != "" {
				return nil
			}
		}
		if step := path[len(prefix)]; step.Name != "" {
			f.add(t, step, ex.Range())
		}
		return nil
	})
}

func (f *instanceRefFinder) add(t string, step PathStep, whole hcl.Range) {
	name := step.Range
	// TraverseAttr ranges include the leading ".".
	if name.Start.Byte < len(f.src) && f.src[name.Start.Byte] == '.' {
		name.Start.Byte++
		name.Start.Column++
	}
	key := [2]int{name.Start.Byte, name.End.Byte}
	if f.seen[key] {
		return
	}
	f.seen[key] = true
	whole.End = step.Range.End
	f.refs = append(f.refs, InstanceAttributeRef{Type: t, Attribute: step.Name, NameRange: name, Range: whole})
}
