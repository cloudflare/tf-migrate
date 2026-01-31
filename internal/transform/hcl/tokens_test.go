package hcl

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

func TestBuildTemplateStringTokens(t *testing.T) {
	tests := []struct {
		name       string
		exprTokens hclwrite.Tokens
		suffix     string
		expected   string
	}{
		{
			name: "Template with suffix",
			exprTokens: hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
				{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
				{Type: hclsyntax.TokenIdent, Bytes: []byte("zone_id")},
			},
			suffix:   "/settings",
			expected: `"${var.zone_id}/settings"`,
		},
		{
			name: "Template without suffix",
			exprTokens: hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("resource")},
				{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
				{Type: hclsyntax.TokenIdent, Bytes: []byte("example")},
				{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
				{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
			},
			suffix:   "",
			expected: `"${resource.example.id}"`,
		},
		{
			name: "Template with complex expression",
			exprTokens: hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("cloudflare_zone")},
				{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
				{Type: hclsyntax.TokenIdent, Bytes: []byte("main")},
				{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
				{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
			},
			suffix:   "/argo/smart_routing",
			expected: `"${cloudflare_zone.main.id}/argo/smart_routing"`,
		},
		{
			name: "Template with numeric suffix",
			exprTokens: hclwrite.Tokens{
				{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
				{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
				{Type: hclsyntax.TokenIdent, Bytes: []byte("account_id")},
			},
			suffix:   "/123",
			expected: `"${var.account_id}/123"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildTemplateStringTokens(tt.exprTokens, tt.suffix)

			// Convert tokens to string for comparison
			var builder strings.Builder
			for _, token := range result {
				builder.Write(token.Bytes)
			}
			output := builder.String()

			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestBuildTemplateStringTokens_Structure(t *testing.T) {
	// Test the token structure in detail
	exprTokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
		{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
	}

	result := BuildTemplateStringTokens(exprTokens, "/suffix")

	// Verify token structure
	assert.Equal(t, hclsyntax.TokenOQuote, result[0].Type, "First token should be opening quote")
	assert.Equal(t, []byte{'"'}, result[0].Bytes)

	assert.Equal(t, hclsyntax.TokenTemplateInterp, result[1].Type, "Second token should be template interpolation start")
	assert.Equal(t, []byte("${"), result[1].Bytes)

	// Next tokens should be the expression
	assert.Equal(t, hclsyntax.TokenIdent, result[2].Type)
	assert.Equal(t, []byte("var"), result[2].Bytes)

	// Find the template sequence end
	foundSeqEnd := false
	foundSuffix := false
	foundCQuote := false

	for i, token := range result {
		if token.Type == hclsyntax.TokenTemplateSeqEnd {
			foundSeqEnd = true
			assert.Equal(t, []byte("}"), token.Bytes)

			// Next should be the suffix
			if i+1 < len(result) {
				nextToken := result[i+1]
				if nextToken.Type == hclsyntax.TokenTemplateControl {
					foundSuffix = true
					assert.Equal(t, []byte("/suffix"), nextToken.Bytes)
				}
			}
		}
		if token.Type == hclsyntax.TokenCQuote {
			foundCQuote = true
			assert.Equal(t, []byte{'"'}, token.Bytes)
		}
	}

	assert.True(t, foundSeqEnd, "Should have template sequence end")
	assert.True(t, foundSuffix, "Should have suffix token")
	assert.True(t, foundCQuote, "Should have closing quote")
}

func TestBuildResourceReference(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		expected     string
	}{
		{
			name:         "Simple resource reference",
			resourceType: "cloudflare_zone",
			resourceName: "example",
			expected:     "cloudflare_zone.example",
		},
		{
			name:         "DNS record reference",
			resourceType: "cloudflare_dns_record",
			resourceName: "www",
			expected:     "cloudflare_dns_record.www",
		},
		{
			name:         "Argo smart routing reference",
			resourceType: "cloudflare_argo_smart_routing",
			resourceName: "main",
			expected:     "cloudflare_argo_smart_routing.main",
		},
		{
			name:         "Resource with underscore in name",
			resourceType: "cloudflare_access_policy",
			resourceName: "admin_policy",
			expected:     "cloudflare_access_policy.admin_policy",
		},
		{
			name:         "Resource with hyphen in name",
			resourceType: "cloudflare_zone_settings_override",
			resourceName: "production-zone",
			expected:     "cloudflare_zone_settings_override.production-zone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildResourceReference(tt.resourceType, tt.resourceName)

			// Convert tokens to string for comparison
			var builder strings.Builder
			for _, token := range result {
				builder.Write(token.Bytes)
			}
			output := builder.String()

			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestBuildResourceReference_Structure(t *testing.T) {
	// Test the token structure in detail
	result := BuildResourceReference("cloudflare_zone", "example")

	// Should have exactly 3 tokens: type, dot, name
	assert.Equal(t, 3, len(result), "Should have 3 tokens")

	// First token: resource type
	assert.Equal(t, hclsyntax.TokenIdent, result[0].Type)
	assert.Equal(t, []byte("cloudflare_zone"), result[0].Bytes)

	// Second token: dot
	assert.Equal(t, hclsyntax.TokenDot, result[1].Type)
	assert.Equal(t, []byte("."), result[1].Bytes)

	// Third token: resource name
	assert.Equal(t, hclsyntax.TokenIdent, result[2].Type)
	assert.Equal(t, []byte("example"), result[2].Bytes)
}

func TestBuildResourceReference_InMovedBlock(t *testing.T) {
	// Test that the reference can be used in a moved block context
	fromTokens := BuildResourceReference("cloudflare_argo", "main")
	toTokens := BuildResourceReference("cloudflare_argo_smart_routing", "main")

	// Verify the from tokens
	var fromBuilder strings.Builder
	for _, token := range fromTokens {
		fromBuilder.Write(token.Bytes)
	}
	assert.Equal(t, "cloudflare_argo.main", fromBuilder.String())

	// Verify the to tokens
	var toBuilder strings.Builder
	for _, token := range toTokens {
		toBuilder.Write(token.Bytes)
	}
	assert.Equal(t, "cloudflare_argo_smart_routing.main", toBuilder.String())
}

func TestBuildTemplateStringTokens_EmptySuffix(t *testing.T) {
	// Test with empty string suffix (should not add suffix token)
	exprTokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
		{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
	}

	result := BuildTemplateStringTokens(exprTokens, "")

	// Convert to string
	var builder strings.Builder
	for _, token := range result {
		builder.Write(token.Bytes)
	}
	output := builder.String()

	assert.Equal(t, `"${var.id}"`, output)

	// Verify there's no TokenTemplateControl when suffix is empty
	for _, token := range result {
		assert.NotEqual(t, hclsyntax.TokenTemplateControl, token.Type, "Should not have suffix token when suffix is empty")
	}
}

func TestBuildResourceReference_EmptyStrings(t *testing.T) {
	// Test edge case with empty strings (should still work, though not practical)
	result := BuildResourceReference("", "")

	assert.Equal(t, 3, len(result), "Should still have 3 tokens")

	var builder strings.Builder
	for _, token := range result {
		builder.Write(token.Bytes)
	}

	assert.Equal(t, ".", builder.String())
}

func TestTokensForSimpleValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "String value",
			value:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "Empty string",
			value:    "",
			expected: `""`,
		},
		{
			name:     "Integer value",
			value:    42,
			expected: "42",
		},
		{
			name:     "Zero integer",
			value:    0,
			expected: "0",
		},
		{
			name:     "Negative integer",
			value:    -10,
			expected: "-10",
		},
		{
			name:     "Int64 value",
			value:    int64(9223372036854775807),
			expected: "9223372036854775807",
		},
		{
			name:     "Float64 value",
			value:    3.14,
			expected: "3.14",
		},
		{
			name:     "Float64 zero",
			value:    0.0,
			expected: "0",
		},
		{
			name:     "Boolean true",
			value:    true,
			expected: "true",
		},
		{
			name:     "Boolean false",
			value:    false,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TokensForSimpleValue(tt.value)

			assert.NotNil(t, result, "Should return tokens for valid types")

			// Convert tokens to string
			var builder strings.Builder
			for _, token := range result {
				builder.Write(token.Bytes)
			}
			output := builder.String()

			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestTokensForSimpleValue_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "Nil value",
			value: nil,
		},
		{
			name:  "Struct",
			value: struct{ Name string }{Name: "test"},
		},
		{
			name:  "Slice",
			value: []string{"a", "b"},
		},
		{
			name:  "Map",
			value: map[string]string{"key": "value"},
		},
		{
			name:  "Pointer",
			value: new(int),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TokensForSimpleValue(tt.value)
			assert.Nil(t, result, "Should return nil for unsupported types")
		})
	}
}

func TestTokensForSimpleValue_TokenTypes(t *testing.T) {
	// Test that the correct token types are used
	t.Run("String produces quoted tokens", func(t *testing.T) {
		result := TokensForSimpleValue("test")

		// Should have opening quote, literal, closing quote
		assert.GreaterOrEqual(t, len(result), 2, "Should have at least opening and closing quotes")

		// Find opening and closing quotes
		hasOQuote := false
		hasCQuote := false
		for _, token := range result {
			if token.Type == hclsyntax.TokenOQuote {
				hasOQuote = true
			}
			if token.Type == hclsyntax.TokenCQuote {
				hasCQuote = true
			}
		}

		assert.True(t, hasOQuote, "Should have opening quote token")
		assert.True(t, hasCQuote, "Should have closing quote token")
	})

	t.Run("Number produces number literal", func(t *testing.T) {
		result := TokensForSimpleValue(42)

		// Should have a number literal token
		hasNumberLit := false
		for _, token := range result {
			if token.Type == hclsyntax.TokenNumberLit {
				hasNumberLit = true
			}
		}

		assert.True(t, hasNumberLit, "Should have number literal token")
	})

	t.Run("Boolean produces identifier", func(t *testing.T) {
		result := TokensForSimpleValue(true)

		// Should have an identifier token (true/false are identifiers in HCL)
		hasIdent := false
		for _, token := range result {
			if token.Type == hclsyntax.TokenIdent {
				hasIdent = true
			}
		}

		assert.True(t, hasIdent, "Should have identifier token for boolean")
	})
}

func TestAppendWarningComment(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectedComment string
	}{
		{
			name:            "Simple warning message",
			message:         "This resource needs manual review",
			expectedComment: "# MIGRATION WARNING: This resource needs manual review",
		},
		{
			name:            "Warning with special characters",
			message:         "Complex patterns like [0-9]+ are not supported",
			expectedComment: "# MIGRATION WARNING: Complex patterns like [0-9]+ are not supported",
		},
		{
			name:            "Long warning message",
			message:         "Unable to automatically merge cloudflare_list_item resources",
			expectedComment: "# MIGRATION WARNING: Unable to automatically merge cloudflare_list_item resources",
		},
		{
			name:            "Warning with punctuation",
			message:         "Cannot determine list kind - manual merge may be required",
			expectedComment: "# MIGRATION WARNING: Cannot determine list kind - manual merge may be required",
		},
		{
			name:            "Empty message",
			message:         "",
			expectedComment: "# MIGRATION WARNING: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new file and body
			file := hclwrite.NewEmptyFile()
			body := file.Body()

			// Add a test attribute to ensure body is not empty
			body.SetAttributeRaw("test", TokensForSimpleValue("value"))

			// Add the migration warning
			AppendWarningComment(body, tt.message)

			// Get the resulting HCL
			result := string(file.Bytes())

			// Verify the warning comment is present
			assert.Contains(t, result, tt.expectedComment, "Should contain the warning comment")

			// Verify the original attribute is still present
			assert.Contains(t, result, `test = "value"`, "Should preserve original content")
		})
	}
}

func TestAppendWarningComment_Structure(t *testing.T) {
	// Test the token structure in detail
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	message := "Test warning message"
	AppendWarningComment(body, message)

	result := string(file.Bytes())

	// Verify structure
	assert.Contains(t, result, "# MIGRATION WARNING: Test warning message\n")

	// Verify it starts with # (comment marker)
	lines := strings.Split(result, "\n")
	var commentLine string
	for _, line := range lines {
		if strings.Contains(line, "MIGRATION WARNING") {
			commentLine = line
			break
		}
	}

	assert.NotEmpty(t, commentLine, "Should have found comment line")
	assert.True(t, strings.HasPrefix(commentLine, "#"), "Comment should start with #")
	assert.Contains(t, commentLine, "MIGRATION WARNING:", "Should contain MIGRATION WARNING:")
	assert.Contains(t, commentLine, message, "Should contain the message")
}

func TestAppendWarningComment_MultipleWarnings(t *testing.T) {
	// Test adding multiple warnings to the same body
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	// Add an attribute first
	body.SetAttributeRaw("test", TokensForSimpleValue("value"))

	// Add multiple warnings
	AppendWarningComment(body, "First warning")
	AppendWarningComment(body, "Second warning")
	AppendWarningComment(body, "Third warning")

	result := string(file.Bytes())

	// Verify all warnings are present
	assert.Contains(t, result, "# MIGRATION WARNING: First warning")
	assert.Contains(t, result, "# MIGRATION WARNING: Second warning")
	assert.Contains(t, result, "# MIGRATION WARNING: Third warning")

	// Count occurrences of MIGRATION WARNING
	count := strings.Count(result, "MIGRATION WARNING")
	assert.Equal(t, 3, count, "Should have exactly 3 warnings")
}

func TestAppendWarningComment_WithResource(t *testing.T) {
	// Test adding a warning to a resource block
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	// Create a resource block
	block := body.AppendNewBlock("resource", []string{"cloudflare_list", "example"})
	blockBody := block.Body()
	blockBody.SetAttributeRaw("account_id", TokensForSimpleValue("abc123"))
	blockBody.SetAttributeRaw("name", TokensForSimpleValue("example_list"))
	blockBody.SetAttributeRaw("kind", TokensForSimpleValue("ip"))

	// Add warning to the block body
	AppendWarningComment(blockBody, "Cannot determine list kind for merging list_item resources")

	result := string(file.Bytes())

	// Verify the resource structure is preserved
	assert.Contains(t, result, `resource "cloudflare_list" "example"`)
	assert.Contains(t, result, `"abc123"`)
	assert.Contains(t, result, `"example_list"`)
	assert.Contains(t, result, `"ip"`)

	// Verify the warning is present inside the block
	assert.Contains(t, result, "# MIGRATION WARNING: Cannot determine list kind for merging list_item resources")
}

func TestAppendWarningComment_PreservesFormatting(t *testing.T) {
	// Test that adding a warning doesn't break existing formatting
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	// Add some attributes with specific formatting
	body.SetAttributeRaw("first", TokensForSimpleValue("value1"))
	body.SetAttributeRaw("second", TokensForSimpleValue("value2"))

	// Add warning
	AppendWarningComment(body, "Manual review required")

	// Add more attributes after warning
	body.SetAttributeRaw("third", TokensForSimpleValue("value3"))

	result := string(file.Bytes())

	// All attributes should still be present (check values, not exact formatting)
	assert.Contains(t, result, `"value1"`)
	assert.Contains(t, result, `"value2"`)
	assert.Contains(t, result, `"value3"`)
	assert.Contains(t, result, "# MIGRATION WARNING: Manual review required")
}
