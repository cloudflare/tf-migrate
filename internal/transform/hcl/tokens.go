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
