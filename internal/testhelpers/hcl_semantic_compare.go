package testhelpers

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// semanticHCLEqual compares two HCL files semantically, ignoring attribute order
func semanticHCLEqual(t *testing.T, expected, actual *hclwrite.File) bool {
	t.Helper()
	return compareBlocks(t, "root", expected.Body().Blocks(), actual.Body().Blocks())
}

// compareBlocks compares two slices of HCL blocks semantically
func compareBlocks(t *testing.T, path string, expected, actual []*hclwrite.Block) bool {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("%s: block count mismatch: expected %d, got %d", path, len(expected), len(actual))
		return false
	}

	equal := true
	for i := range expected {
		blockPath := path + "[" + expected[i].Type() + "]"
		if len(expected[i].Labels()) > 0 {
			blockPath += "." + strings.Join(expected[i].Labels(), ".")
		}

		// Compare block type and labels
		if expected[i].Type() != actual[i].Type() {
			t.Errorf("%s: block type mismatch: expected %q, got %q", blockPath, expected[i].Type(), actual[i].Type())
			equal = false
			continue
		}

		if len(expected[i].Labels()) != len(actual[i].Labels()) {
			t.Errorf("%s: label count mismatch: expected %d, got %d", blockPath, len(expected[i].Labels()), len(actual[i].Labels()))
			equal = false
			continue
		}

		for j := range expected[i].Labels() {
			if expected[i].Labels()[j] != actual[i].Labels()[j] {
				t.Errorf("%s: label[%d] mismatch: expected %q, got %q", blockPath, j, expected[i].Labels()[j], actual[i].Labels()[j])
				equal = false
			}
		}

		// Compare block body (attributes and nested blocks)
		if !compareBody(t, blockPath, expected[i].Body(), actual[i].Body()) {
			equal = false
		}
	}

	return equal
}

// compareBody compares two HCL block bodies semantically
func compareBody(t *testing.T, path string, expected, actual *hclwrite.Body) bool {
	t.Helper()

	equal := true

	// Compare attributes (order-independent)
	expectedAttrs := expected.Attributes()
	actualAttrs := actual.Attributes()

	if len(expectedAttrs) != len(actualAttrs) {
		t.Errorf("%s: attribute count mismatch: expected %d, got %d", path, len(expectedAttrs), len(actualAttrs))
		equal = false
	}

	// Check all expected attributes exist in actual with same values
	for name, expectedAttr := range expectedAttrs {
		actualAttr, exists := actualAttrs[name]
		if !exists {
			t.Errorf("%s: missing attribute %q", path, name)
			equal = false
			continue
		}

		// Compare attribute values
		// For object expressions, we need to parse and compare recursively
		if !compareAttributeValues(t, path+"."+name, expectedAttr, actualAttr) {
			equal = false
		}
	}

	// Check for unexpected attributes in actual
	for name := range actualAttrs {
		if _, exists := expectedAttrs[name]; !exists {
			t.Errorf("%s: unexpected attribute %q", path, name)
			equal = false
		}
	}

	// Compare nested blocks
	expectedBlocks := expected.Blocks()
	actualBlocks := actual.Blocks()

	if !compareBlocks(t, path, expectedBlocks, actualBlocks) {
		equal = false
	}

	return equal
}

// compareAttributeValues compares two attribute values, handling both simple and complex values
func compareAttributeValues(t *testing.T, path string, expected, actual *hclwrite.Attribute) bool {
	t.Helper()

	expectedTokens := expected.Expr().BuildTokens(nil)
	actualTokens := actual.Expr().BuildTokens(nil)

	// Check if this is an object expression by looking for opening brace
	expectedStr := tokensToString(expectedTokens)
	actualStr := tokensToString(actualTokens)

	expectedStr = strings.TrimSpace(expectedStr)
	actualStr = strings.TrimSpace(actualStr)

	// If both start with '{', they're object expressions - parse and compare recursively
	if strings.HasPrefix(expectedStr, "{") && strings.HasPrefix(actualStr, "{") {
		return compareObjectExpression(t, path, expectedStr, actualStr)
	}

	// For non-object values, compare tokens with normalization
	if !compareTokens(expectedTokens, actualTokens) {
		t.Errorf("%s: attribute value mismatch: expected %q, got %q",
			path,
			expectedStr,
			actualStr)
		return false
	}

	return true
}

// compareObjectExpression parses and compares two object expressions recursively
func compareObjectExpression(t *testing.T, path string, expected, actual string) bool {
	t.Helper()

	// Wrap in a dummy attribute so we can parse as valid HCL
	expectedHCL := "dummy = " + expected
	actualHCL := "dummy = " + actual

	// Parse both expressions
	expectedFile, diags := hclwrite.ParseConfig([]byte(expectedHCL), "expected.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Errorf("%s: failed to parse expected object expression: %v", path, diags)
		return false
	}

	actualFile, diags := hclwrite.ParseConfig([]byte(actualHCL), "actual.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Errorf("%s: failed to parse actual object expression: %v", path, diags)
		return false
	}

	// Get the dummy attribute from both
	expectedAttr := expectedFile.Body().GetAttribute("dummy")
	actualAttr := actualFile.Body().GetAttribute("dummy")

	if expectedAttr == nil || actualAttr == nil {
		t.Errorf("%s: failed to extract dummy attribute", path)
		return false
	}

	// Parse the object content recursively
	return compareObjectTokens(t, path, expectedAttr.Expr().BuildTokens(nil), actualAttr.Expr().BuildTokens(nil))
}

// compareObjectTokens recursively compares object expression tokens
func compareObjectTokens(t *testing.T, path string, expected, actual hclwrite.Tokens) bool {
	t.Helper()

	// For object expressions, we need to extract key-value pairs and compare them
	// This is a simplified parser that handles the common case of { key = value, ... }

	expectedPairs := extractObjectPairs(expected)
	actualPairs := extractObjectPairs(actual)

	if len(expectedPairs) != len(actualPairs) {
		t.Errorf("%s: object field count mismatch: expected %d, got %d", path, len(expectedPairs), len(actualPairs))
		return false
	}

	equal := true
	for key, expectedValue := range expectedPairs {
		actualValue, exists := actualPairs[key]
		if !exists {
			t.Errorf("%s: missing field %q", path, key)
			equal = false
			continue
		}

		// Check if the value is a nested object or array - if so, compare recursively
		expectedStr := strings.TrimSpace(tokensToString(expectedValue))
		actualStr := strings.TrimSpace(tokensToString(actualValue))

		if strings.HasPrefix(expectedStr, "{") && strings.HasPrefix(actualStr, "{") {
			// Nested object - compare recursively
			if !compareObjectTokens(t, path+"."+key, expectedValue, actualValue) {
				equal = false
			}
		} else if strings.HasPrefix(expectedStr, "[") && strings.HasPrefix(actualStr, "[") {
			// Array - compare recursively
			if !compareArrayTokens(t, path+"."+key, expectedValue, actualValue) {
				equal = false
			}
		} else {
			// Simple value - compare with normalization
			if !compareTokens(expectedValue, actualValue) {
				t.Errorf("%s.%s: value mismatch: expected %q, got %q",
					path, key,
					tokensToString(expectedValue),
					tokensToString(actualValue))
				equal = false
			}
		}
	}

	// Check for unexpected fields
	for key := range actualPairs {
		if _, exists := expectedPairs[key]; !exists {
			t.Errorf("%s: unexpected field %q", path, key)
			equal = false
		}
	}

	return equal
}

// compareArrayTokens recursively compares array expression tokens
func compareArrayTokens(t *testing.T, path string, expected, actual hclwrite.Tokens) bool {
	t.Helper()

	expectedElements := extractArrayElements(expected)
	actualElements := extractArrayElements(actual)

	if len(expectedElements) != len(actualElements) {
		t.Errorf("%s: array length mismatch: expected %d, got %d", path, len(expectedElements), len(actualElements))
		return false
	}

	equal := true
	for i := range expectedElements {
		elementPath := path + "[" + string(rune(i)) + "]"

		// Check if the element is a nested object or array
		expectedStr := strings.TrimSpace(tokensToString(expectedElements[i]))
		actualStr := strings.TrimSpace(tokensToString(actualElements[i]))

		if strings.HasPrefix(expectedStr, "{") && strings.HasPrefix(actualStr, "{") {
			// Nested object - compare recursively
			if !compareObjectTokens(t, elementPath, expectedElements[i], actualElements[i]) {
				equal = false
			}
		} else if strings.HasPrefix(expectedStr, "[") && strings.HasPrefix(actualStr, "[") {
			// Nested array - compare recursively
			if !compareArrayTokens(t, elementPath, expectedElements[i], actualElements[i]) {
				equal = false
			}
		} else {
			// Simple value - compare with normalization
			if !compareTokens(expectedElements[i], actualElements[i]) {
				t.Errorf("%s: value mismatch: expected %q, got %q",
					elementPath,
					tokensToString(expectedElements[i]),
					tokensToString(actualElements[i]))
				equal = false
			}
		}
	}

	return equal
}

// extractArrayElements extracts elements from array expression tokens
func extractArrayElements(tokens hclwrite.Tokens) []hclwrite.Tokens {
	var elements []hclwrite.Tokens

	i := 0
	// Skip opening bracket
	for i < len(tokens) && string(tokens[i].Bytes) != "[" {
		i++
	}
	i++ // Skip the '['

	var currentElement hclwrite.Tokens
	depth := 0 // Track nested braces and brackets

	for i < len(tokens) {
		tok := tokens[i]
		bytes := string(tok.Bytes)

		// Track nesting depth
		if bytes == "{" || bytes == "[" {
			depth++
			currentElement = append(currentElement, tok)
		} else if bytes == "}" || bytes == "]" {
			if depth == 0 && bytes == "]" {
				// End of array
				if len(currentElement) > 0 {
					// Trim trailing whitespace from the element
					currentElement = trimTrailingWhitespace(currentElement)
					elements = append(elements, currentElement)
				}
				break
			}
			depth--
			currentElement = append(currentElement, tok)
		} else if depth == 0 && bytes == "," {
			// End of current element
			if len(currentElement) > 0 {
				// Trim trailing whitespace from the element
				currentElement = trimTrailingWhitespace(currentElement)
				elements = append(elements, currentElement)
			}
			currentElement = nil
		} else {
			// Skip leading whitespace for each element
			if len(currentElement) == 0 && isWhitespace(tok) {
				i++
				continue
			}
			currentElement = append(currentElement, tok)
		}

		i++
	}

	// Don't forget to trim the last element
	if len(currentElement) > 0 {
		currentElement = trimTrailingWhitespace(currentElement)
	}

	return elements
}

// trimTrailingWhitespace removes whitespace tokens from the end of a token slice
func trimTrailingWhitespace(tokens hclwrite.Tokens) hclwrite.Tokens {
	// Find the last non-whitespace token
	lastNonWhitespace := len(tokens) - 1
	for lastNonWhitespace >= 0 && isWhitespace(tokens[lastNonWhitespace]) {
		lastNonWhitespace--
	}
	return tokens[:lastNonWhitespace+1]
}

// extractObjectPairs extracts key-value pairs from object expression tokens
// Returns a map of field name to value tokens
func extractObjectPairs(tokens hclwrite.Tokens) map[string]hclwrite.Tokens {
	pairs := make(map[string]hclwrite.Tokens)

	i := 0
	// Skip opening brace
	for i < len(tokens) && string(tokens[i].Bytes) != "{" {
		i++
	}
	i++ // Skip the '{'

	for i < len(tokens) {
		// Skip whitespace
		for i < len(tokens) && isWhitespace(tokens[i]) {
			i++
		}

		if i >= len(tokens) || string(tokens[i].Bytes) == "}" {
			break
		}

		// Read field name
		if i >= len(tokens) {
			break
		}
		fieldName := string(tokens[i].Bytes)
		i++

		// Skip whitespace and '='
		for i < len(tokens) && (isWhitespace(tokens[i]) || string(tokens[i].Bytes) == "=") {
			i++
		}

		// Read value tokens until newline or comma or closing brace
		var valueTokens hclwrite.Tokens
		depth := 0 // Track nested braces and brackets
		for i < len(tokens) {
			tok := tokens[i]
			bytes := string(tok.Bytes)

			// Track nesting depth
			if bytes == "{" || bytes == "[" {
				depth++
			} else if bytes == "}" || bytes == "]" {
				if depth == 0 {
					// End of object
					break
				}
				depth--
			}

			// If at depth 0, check for end of value
			if depth == 0 && (bytes == "\n" || bytes == "\r\n" || bytes == ",") {
				i++ // Skip the delimiter
				break
			}

			valueTokens = append(valueTokens, tok)
			i++
		}

		pairs[fieldName] = valueTokens
	}

	return pairs
}

// isWhitespace checks if a token is whitespace
func isWhitespace(tok *hclwrite.Token) bool {
	bytes := string(tok.Bytes)
	return bytes == "\n" || bytes == "\r\n" || bytes == " " || bytes == "\t"
}

// compareTokens compares two token slices, normalizing whitespace
func compareTokens(expected, actual hclwrite.Tokens) bool {
	// Normalize by removing newline and space tokens, keeping only semantic tokens
	expectedNorm := normalizeTokens(expected)
	actualNorm := normalizeTokens(actual)

	if len(expectedNorm) != len(actualNorm) {
		return false
	}

	for i := range expectedNorm {
		// Compare bytes with trimmed whitespace
		expectedBytes := strings.TrimSpace(string(expectedNorm[i].Bytes))
		actualBytes := strings.TrimSpace(string(actualNorm[i].Bytes))

		// If content matches after trimming, that's what matters for semantic comparison
		if expectedBytes != actualBytes {
			return false
		}
	}

	return true
}

// normalizeTokens filters out whitespace tokens to enable semantic comparison
func normalizeTokens(tokens hclwrite.Tokens) hclwrite.Tokens {
	var result hclwrite.Tokens
	for _, tok := range tokens {
		// Skip newline and space tokens (hclsyntax.TokenNewline, TokenSpace)
		// These are used for formatting only and don't affect semantics
		bytes := string(tok.Bytes)
		trimmed := strings.TrimSpace(bytes)
		if trimmed == "" {
			// Skip whitespace-only tokens
			continue
		}
		result = append(result, tok)
	}
	return result
}

// tokensToString converts tokens to a string representation for error messages
func tokensToString(tokens hclwrite.Tokens) string {
	var result strings.Builder
	for _, tok := range tokens {
		result.Write(tok.Bytes)
	}
	return result.String()
}
