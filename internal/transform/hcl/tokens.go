// Package hcl provides utilities for building and manipulating HCL tokens
package hcl

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// BuildTemplateStringTokens creates tokens for a template string like "${expr}/literal"
// This is useful for creating import block IDs and other template expressions
func BuildTemplateStringTokens(exprTokens hclwrite.Tokens, suffix string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOQuote, Bytes: []byte{'"'}},
		{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")},
	}

	tokens = append(tokens, exprTokens...)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte{'}'}},
	)

	if suffix != "" {
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateControl, Bytes: []byte(suffix)})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte{'"'}})

	return tokens
}

// BuildResourceReference creates tokens for a resource reference like "type.name"
// Used for creating references to resources in moved blocks and import blocks
func BuildResourceReference(resourceType, resourceName string) hclwrite.Tokens {
	return hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(resourceType)},
		{Type: hclsyntax.TokenDot, Bytes: []byte{'.'}},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(resourceName)},
	}
}

// TokensForSimpleValue creates tokens for a simple value (string, number, bool)
// This is a low-level utility for creating HCL tokens from Go primitive types
func TokensForSimpleValue(val interface{}) hclwrite.Tokens {
	switch v := val.(type) {
	case string:
		return hclwrite.TokensForValue(cty.StringVal(v))
	case int:
		return hclwrite.TokensForValue(cty.NumberIntVal(int64(v)))
	case int64:
		return hclwrite.TokensForValue(cty.NumberIntVal(v))
	case float64:
		return hclwrite.TokensForValue(cty.NumberFloatVal(v))
	case bool:
		return hclwrite.TokensForValue(cty.BoolVal(v))
	default:
		return nil
	}
}

// TokensForEmptyArray creates tokens for an empty array []
func TokensForEmptyArray() hclwrite.Tokens {
	return hclwrite.TokensForTuple([]hclwrite.Tokens{})
}

// AppendWarningComment adds a migration warning comment to a body.
// Used to flag configurations that need manual review after automated migration.
//
// The comment will be formatted as: # MIGRATION WARNING: <message>
//
// Example:
//
//	body := hclwrite.NewEmptyFile().Body()
//	AppendWarningComment(body, "Cannot determine list kind for merging list_item resources")
//
// Result in HCL:
//
//	# MIGRATION WARNING: Cannot determine list kind for merging list_item resources
func AppendWarningComment(body *hclwrite.Body, message string) {
	comment := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenComment,
			Bytes: []byte("# MIGRATION WARNING: " + message + "\n"),
		},
	}
	body.AppendUnstructuredTokens(comment)
}

// JoinQuotedLiteral concatenates the literal runs of a *plain* quoted string
// and reports whether the token sequence was in fact a plain quoted string.
//
// This exists because HCL's scanner does not emit one TokenQuotedLit per
// string: it breaks the literal into separate runs at every '%' and '$',
// since those characters may begin a template sequence ("%{" / "${"). A
// value such as
//
//	"https://example.com/a?resid=ABC%21108"
//
// scans as three TokenQuotedLit tokens — "https://example.com/a?resid=ABC",
// "%", "21108". Code that returns the *first* TokenQuotedLit therefore
// silently truncates the value at the first '%' or '$' (BUGS-2160). Backslash
// escapes (\n, \", \uXXXX) are *not* split and need no special handling.
//
// ok is false when the sequence is anything other than a plain quoted string
// — an interpolation ("${...}"), a template control directive ("%{ if }"), a
// traversal, a function call, a number, a bool, and so on. Callers must then
// preserve the raw tokens instead of stringifying, because no string can
// faithfully represent those expressions.
//
// The returned string is the raw *source* bytes between the quotes, with
// escape sequences left exactly as written. That makes it round-trip safe:
// re-emitting it as a TokenQuotedLit reproduces the original value verbatim.
func JoinQuotedLiteral(tokens hclwrite.Tokens) (string, bool) {
	var (
		sb       []byte
		sawOpen  bool
		sawClose bool
	)

	for _, token := range tokens {
		switch token.Type {
		case hclsyntax.TokenOQuote:
			// A second opening quote means this is not a single plain string.
			if sawOpen {
				return "", false
			}
			sawOpen = true
		case hclsyntax.TokenCQuote:
			if !sawOpen || sawClose {
				return "", false
			}
			sawClose = true
		case hclsyntax.TokenQuotedLit:
			// Literal runs may only appear between the quotes.
			if !sawOpen || sawClose {
				return "", false
			}
			sb = append(sb, token.Bytes...)
		default:
			// Interpolation, template control, traversal, call, literal of
			// another type, … — not a plain string.
			return "", false
		}
	}

	if !sawOpen || !sawClose {
		return "", false
	}
	return string(sb), true
}
