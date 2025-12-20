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
