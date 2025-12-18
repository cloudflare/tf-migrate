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
