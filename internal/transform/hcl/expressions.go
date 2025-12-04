package hcl

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// StripIteratorValueSuffix removes the .value suffix from iterator expressions.
// For example, "item.value" becomes "item" when iteratorName is "item".
func StripIteratorValueSuffix(expr string, iteratorName string) string {
	if iteratorName == "" {
		return expr
	}
	// Replace iterator.value with just the iterator name
	valueSuffix := iteratorName + ".value"
	if strings.Contains(expr, valueSuffix) {
		expr = strings.ReplaceAll(expr, valueSuffix, iteratorName)
	}
	return expr
}

// ConvertEnabledDisabledInExpr converts "enabled"/"disabled" string literals
// to true/false boolean values in an expression string.
func ConvertEnabledDisabledInExpr(expr string) string {
	expr = strings.ReplaceAll(expr, `"enabled"`, "true")
	expr = strings.ReplaceAll(expr, `"disabled"`, "false")
	return expr
}

// SetAttributeFromExpressionString parses an expression string and sets it as an attribute.
// Returns an error if the expression cannot be parsed.
func SetAttributeFromExpressionString(body *hclwrite.Body, attrName string, exprStr string) error {
	// Create a temporary HCL file to parse the expression
	tempHCL := attrName + " = " + exprStr
	file, diags := hclwrite.ParseConfig([]byte(tempHCL), "expr", hcl.InitialPos)
	if diags.HasErrors() {
		return diags
	}

	// Extract the attribute and set it on the body
	if attr := file.Body().GetAttribute(attrName); attr != nil {
		body.SetAttributeRaw(attrName, attr.Expr().BuildTokens(nil))
	}

	return nil
}

// IsExpressionAttribute checks if an attribute contains a non-literal expression
// (like variable references, function calls, etc.) rather than a simple literal value.
func IsExpressionAttribute(attr *hclwrite.Attribute) bool {
	if attr == nil {
		return false
	}

	tokens := attr.Expr().BuildTokens(nil)
	for _, token := range tokens {
		switch token.Type {
		case hclsyntax.TokenIdent:
			// Check for common expression patterns
			tokenStr := string(token.Bytes)
			if tokenStr == "each" || tokenStr == "var" || tokenStr == "local" ||
				tokenStr == "count" || tokenStr == "data" || tokenStr == "module" {
				return true
			}
		case hclsyntax.TokenTemplateInterp, hclsyntax.TokenTemplateSeqEnd:
			// Interpolation
			return true
		}
	}

	return false
}

// RemoveFunctionWrapper removes a function wrapper from an attribute expression.
// This is useful when migrating from v4 to v5 where certain function calls need to be unwrapped.
//
// Common use case: Converting toset() calls to plain lists when v5 changes a set type to a list type.
//
// Example with toset:
//
//	Before: allowed_idps = toset(["abc-123", "def-456"])
//	After:  allowed_idps = ["abc-123", "def-456"]
//
// The function extracts the argument from funcName(arg) and replaces the entire expression with just arg.
// If the attribute doesn't exist or doesn't contain the specified function, no changes are made.
//
// Parameters:
//   - body: The HCL body containing the attribute
//   - attrName: The name of the attribute to transform
//   - funcName: The name of the function to remove (e.g., "toset", "tonumber")
func RemoveFunctionWrapper(body *hclwrite.Body, attrName string, funcName string) {
	attr := body.GetAttribute(attrName)
	if attr == nil {
		return
	}

	tokens := attr.Expr().BuildTokens(nil)

	// Look for the pattern: funcName ( arg )
	// We need to find funcName followed by "(" and extract the content until matching ")"
	var result []*hclwrite.Token
	inFunction := false
	parenDepth := 0
	skipUntilArgStart := false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// Check if this is the start of funcName(
		if !inFunction && token.Type == hclsyntax.TokenIdent && string(token.Bytes) == funcName {
			inFunction = true
			skipUntilArgStart = true
			continue
		}

		if inFunction {
			if skipUntilArgStart {
				if token.Type == hclsyntax.TokenOParen {
					parenDepth++
					continue
				}
				// Found the start of the argument (could be a list, object, or other expression)
				// For toset specifically, this is typically a list literal [...]
				if token.Type == hclsyntax.TokenOBrack {
					skipUntilArgStart = false
					result = append(result, token)
					continue
				}
				// Skip whitespace and other tokens before the argument
				continue
			}

			// Track parentheses depth to know when funcName() ends
			if token.Type == hclsyntax.TokenOParen {
				parenDepth++
			} else if token.Type == hclsyntax.TokenCParen {
				parenDepth--
				if parenDepth == 0 {
					// End of funcName(), we're done
					break
				}
			}

			// Collect all other tokens (the function argument)
			result = append(result, token)
		} else {
			// Not in function call, keep token as-is
			result = append(result, token)
		}
	}

	// If we found and transformed the function, update the attribute
	if inFunction && len(result) > 0 {
		body.SetAttributeRaw(attrName, result)
	}
}
