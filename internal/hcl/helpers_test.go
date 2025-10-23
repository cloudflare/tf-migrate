package hcl

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func TestAttributesOrdered(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		expected []string
	}{
		{
			name: "Attributes in specific order",
			hcl: `resource "test" "example" {
  name = "test"
  zone_id = "abc123"
  type = "A"
  value = "192.0.2.1"
}`,
			expected: []string{"name", "zone_id", "type", "value"},
		},
		{
			name: "Mixed attributes and blocks",
			hcl: `resource "test" "example" {
  first = "value1"
  data {
    nested = "value"
  }
  second = "value2"
  third = "value3"
}`,
			expected: []string{"first", "second", "third"},
		},
		{
			name:     "Empty body",
			hcl:      `resource "test" "example" {}`,
			expected: []string{},
		},
		{
			name: "Attributes with complex values",
			hcl: `resource "test" "example" {
  list = ["a", "b", "c"]
  number = 42
  boolean = true
  object = {
    key = "value"
  }
}`,
			expected: []string{"list", "number", "boolean", "object"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.hcl), "", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("Failed to parse HCL: %v", diags)
			}

			block := file.Body().Blocks()[0]
			attrs := AttributesOrdered(block.Body())

			if len(attrs) != len(tt.expected) {
				t.Errorf("Expected %d attributes, got %d", len(tt.expected), len(attrs))
			}

			for i, attrInfo := range attrs {
				if i >= len(tt.expected) {
					break
				}
				if attrInfo.Name != tt.expected[i] {
					t.Errorf("Expected attribute %d to be %s, got %s", i, tt.expected[i], attrInfo.Name)
				}
				if attrInfo.Attribute == nil {
					t.Errorf("Attribute %s has nil Attribute field", attrInfo.Name)
				}
			}
		})
	}
}

func TestBuildTemplateStringTokens(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		suffix   string
		expected string
	}{
		{
			name:     "Simple variable interpolation",
			expr:     "var.zone_id",
			suffix:   "/record1",
			expected: `"${var.zone_id}/record1"`,
		},
		{
			name:     "No suffix",
			expr:     "local.id",
			suffix:   "",
			expected: `"${local.id}"`,
		},
		{
			name:     "Resource reference with suffix",
			expr:     "cloudflare_zone.main.id",
			suffix:   "/dns_record",
			expected: `"${cloudflare_zone.main.id}/dns_record"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the expression to get tokens
			exprTokens := hclwrite.TokensForIdentifier(tt.expr)
			
			tokens := BuildTemplateStringTokens(exprTokens, tt.suffix)
			result := string(tokens.Bytes())
			
			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			
			if result != expected {
				t.Errorf("Expected %s, got %s", expected, result)
			}
		})
	}
}

func TestBuildResourceReference(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		expected     string
	}{
		{
			name:         "Standard resource reference",
			resourceType: "cloudflare_dns_record",
			resourceName: "example",
			expected:     "cloudflare_dns_record.example",
		},
		{
			name:         "Resource with underscores",
			resourceType: "cloudflare_record",
			resourceName: "test_record",
			expected:     "cloudflare_record.test_record",
		},
		{
			name:         "Short names",
			resourceType: "aws_s3_bucket",
			resourceName: "b1",
			expected:     "aws_s3_bucket.b1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := BuildResourceReference(tt.resourceType, tt.resourceName)
			result := string(tokens.Bytes())
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCreateMovedBlock(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected string
	}{
		{
			name: "Simple resource move",
			from: "cloudflare_record.example",
			to:   "cloudflare_dns_record.example",
			expected: `moved {
  from = cloudflare_record.example
  to   = cloudflare_dns_record.example
}`,
		},
		{
			name: "Module resource move",
			from: "module.old.cloudflare_record.test",
			to:   "module.new.cloudflare_dns_record.test",
			expected: `moved {
  from = module.old.cloudflare_record.test
  to   = module.new.cloudflare_dns_record.test
}`,
		},
		{
			name: "Resource with index",
			from: "cloudflare_record.example[0]",
			to:   "cloudflare_dns_record.example[0]",
			expected: `moved {
  from = cloudflare_record.example[0]
  to   = cloudflare_dns_record.example[0]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := CreateMovedBlock(tt.from, tt.to)
			
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(block)
			
			result := string(file.Bytes())
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			
			// Compare normalized (remove extra spaces)
			result = normalizeHCL(result)
			expected = normalizeHCL(expected)
			
			if result != expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestCreateImportBlock(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		importID     string
		expected     string
	}{
		{
			name:         "Simple import block",
			resourceType: "cloudflare_dns_record",
			resourceName: "example",
			importID:     "zone123/record456",
			expected: `import {
  to = cloudflare_dns_record.example
  id = "zone123/record456"
}`,
		},
		{
			name:         "Import with special characters",
			resourceType: "aws_s3_bucket",
			resourceName: "my_bucket",
			importID:     "arn:aws:s3:::my-bucket-name",
			expected: `import {
  to = aws_s3_bucket.my_bucket
  id = "arn:aws:s3:::my-bucket-name"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := CreateImportBlock(tt.resourceType, tt.resourceName, tt.importID)
			
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(block)
			
			result := string(file.Bytes())
			result = normalizeHCL(result)
			expected := normalizeHCL(tt.expected)
			
			if result != expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestCreateImportBlockWithTokens(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		idExpr       string
		expected     string
	}{
		{
			name:         "Import with variable ID",
			resourceType: "cloudflare_dns_record",
			resourceName: "example",
			idExpr:       "var.import_id",
			expected: `import {
  to = cloudflare_dns_record.example
  id = var.import_id
}`,
		},
		{
			name:         "Import with concatenated ID",
			resourceType: "aws_instance",
			resourceName: "web",
			idExpr:       "local.instance_id",
			expected: `import {
  to = aws_instance.web
  id = local.instance_id
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idTokens := hclwrite.TokensForIdentifier(tt.idExpr)
			block := CreateImportBlockWithTokens(tt.resourceType, tt.resourceName, idTokens)
			
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(block)
			
			result := string(file.Bytes())
			result = normalizeHCL(result)
			expected := normalizeHCL(tt.expected)
			
			if result != expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestBuildObjectFromBlock(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		expected string
	}{
		{
			name: "Simple block to object",
			hcl: `data {
  priority = 10
  target = "mail.example.com"
}`,
			expected: `{
  priority = 10
  target   = "mail.example.com"
}`,
		},
		{
			name: "Block with various types",
			hcl: `settings {
  enabled = true
  count = 5
  name = "test"
  tags = ["a", "b"]
}`,
			expected: `{
  enabled = true
  count   = 5
  name    = "test"
  tags    = ["a", "b"]
}`,
		},
		{
			name: "Empty block",
			hcl:  `empty {}`,
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.hcl), "", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("Failed to parse HCL: %v", diags)
			}

			block := file.Body().Blocks()[0]
			tokens := BuildObjectFromBlock(block)
			result := string(tokens.Bytes())
			
			// Normalize for comparison
			result = normalizeHCL(result)
			expected := normalizeHCL(tt.expected)
			
			if result != expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestSetAttributeValue(t *testing.T) {
	tests := []struct {
		name     string
		attrName string
		value    interface{}
		expected string
	}{
		{
			name:     "String value",
			attrName: "name",
			value:    "test-value",
			expected: `name = "test-value"`,
		},
		{
			name:     "Integer value",
			attrName: "count",
			value:    42,
			expected: `count = 42`,
		},
		{
			name:     "Int64 value",
			attrName: "bignum",
			value:    int64(9999999999),
			expected: `bignum = 9999999999`,
		},
		{
			name:     "Float value",
			attrName: "ratio",
			value:    3.14,
			expected: `ratio = 3.14`,
		},
		{
			name:     "Boolean true",
			attrName: "enabled",
			value:    true,
			expected: `enabled = true`,
		},
		{
			name:     "Boolean false",
			attrName: "disabled",
			value:    false,
			expected: `disabled = false`,
		},
		{
			name:     "String slice",
			attrName: "tags",
			value:    []string{"web", "production", "critical"},
			expected: `tags = ["web", "production", "critical"]`,
		},
		{
			name:     "Empty string slice",
			attrName: "empty_list",
			value:    []string{},
			expected: `empty_list = []`,
		},
		{
			name:     "String map",
			attrName: "labels",
			value:    map[string]string{"env": "prod", "team": "backend"},
			expected: `labels = {
  env  = "prod"
  team = "backend"
}`,
		},
		{
			name:     "Empty map",
			attrName: "empty_map",
			value:    map[string]string{},
			expected: `empty_map = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := hclwrite.NewEmptyFile().Body()
			SetAttributeValue(body, tt.attrName, tt.value)
			
			result := string(hclwrite.Format(body.BuildTokens(nil).Bytes()))
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			
			// For maps, the order might vary, so normalize
			if strings.Contains(expected, "{") && strings.Contains(expected, "}") {
				result = normalizeHCL(result)
				expected = normalizeHCL(expected)
			}
			
			if result != expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestCopyAttribute(t *testing.T) {
	tests := []struct {
		name        string
		sourceHCL   string
		attrToCopy  string
		expectCopy  bool
		expected    string
	}{
		{
			name: "Copy simple attribute",
			sourceHCL: `resource "test" "source" {
  name = "test-name"
  value = 123
}`,
			attrToCopy:  "name",
			expectCopy:  true,
			expected:    `name = "test-name"`,
		},
		{
			name: "Copy complex attribute",
			sourceHCL: `resource "test" "source" {
  tags = ["a", "b", "c"]
}`,
			attrToCopy:  "tags",
			expectCopy:  true,
			expected:    `tags = ["a", "b", "c"]`,
		},
		{
			name: "Copy non-existent attribute",
			sourceHCL: `resource "test" "source" {
  name = "test"
}`,
			attrToCopy:  "missing",
			expectCopy:  false,
			expected:    ``,
		},
		{
			name: "Copy expression attribute",
			sourceHCL: `resource "test" "source" {
  zone_id = var.zone_id
}`,
			attrToCopy:  "zone_id",
			expectCopy:  true,
			expected:    `zone_id = var.zone_id`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.sourceHCL), "", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("Failed to parse HCL: %v", diags)
			}

			sourceBlock := file.Body().Blocks()[0]
			targetBody := hclwrite.NewEmptyFile().Body()
			
			CopyAttribute(sourceBlock.Body(), targetBody, tt.attrToCopy)
			
			result := string(hclwrite.Format(targetBody.BuildTokens(nil).Bytes()))
			result = strings.TrimSpace(result)
			
			if tt.expectCopy {
				if result != tt.expected {
					t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
				}
			} else {
				if result != "" {
					t.Errorf("Expected no copy, but got: %s", result)
				}
			}
		})
	}
}

func TestRemoveEmptyBlocks(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		blockType string
		expected  string
	}{
		{
			name: "Remove empty data blocks",
			hcl: `resource "test" "example" {
  name = "test"
  data {}
  value = "123"
  data {}
}`,
			blockType: "data",
			expected: `resource "test" "example" {
  name  = "test"
  value = "123"
}`,
		},
		{
			name: "Keep non-empty blocks",
			hcl: `resource "test" "example" {
  data {
    key = "value"
  }
  data {}
  data {
    another = "field"
  }
}`,
			blockType: "data",
			expected: `resource "test" "example" {
  data {
    key = "value"
  }
  data {
    another = "field"
  }
}`,
		},
		{
			name: "No blocks to remove",
			hcl: `resource "test" "example" {
  name = "test"
  data {
    field = "value"
  }
}`,
			blockType: "settings",
			expected: `resource "test" "example" {
  name = "test"
  data {
    field = "value"
  }
}`,
		},
		{
			name: "Remove all empty blocks of type",
			hcl: `resource "test" "example" {
  lifecycle {}
  lifecycle {}
  lifecycle {}
}`,
			blockType: "lifecycle",
			expected: `resource "test" "example" {
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.hcl), "", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("Failed to parse HCL: %v", diags)
			}

			block := file.Body().Blocks()[0]
			RemoveEmptyBlocks(block.Body(), tt.blockType)
			
			newFile := hclwrite.NewEmptyFile()
			newFile.Body().AppendBlock(block)
			
			result := string(hclwrite.Format(newFile.Bytes()))
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			
			if result != expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestTokensForSimpleValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		isNil    bool
	}{
		{
			name:     "String value",
			value:    "hello world",
			expected: `"hello world"`,
		},
		{
			name:     "Integer value",
			value:    42,
			expected: `42`,
		},
		{
			name:     "Int64 value",
			value:    int64(9876543210),
			expected: `9876543210`,
		},
		{
			name:     "Float value",
			value:    3.14159,
			expected: `3.14159`,
		},
		{
			name:     "Boolean true",
			value:    true,
			expected: `true`,
		},
		{
			name:     "Boolean false",
			value:    false,
			expected: `false`,
		},
		{
			name:  "Unsupported type returns nil",
			value: []string{"unsupported"},
			isNil: true,
		},
		{
			name:  "Nil value returns nil",
			value: nil,
			isNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := TokensForSimpleValue(tt.value)
			
			if tt.isNil {
				if tokens != nil {
					t.Errorf("Expected nil, got tokens: %s", string(tokens.Bytes()))
				}
			} else {
				if tokens == nil {
					t.Fatal("Expected tokens, got nil")
				}
				
				result := string(tokens.Bytes())
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestSetAttributeValueComplexType(t *testing.T) {
	// Test that complex types are not handled
	body := hclwrite.NewEmptyFile().Body()
	
	// Try to set a complex type (struct)
	type complexType struct {
		Field string
	}
	
	SetAttributeValue(body, "complex", complexType{Field: "value"})
	
	// Should not have added any attribute
	if len(body.Attributes()) != 0 {
		t.Error("Expected no attributes for unsupported type, but got some")
	}
}

func TestAttributeValueWithCtyTypes(t *testing.T) {
	// Test using cty.Value directly
	body := hclwrite.NewEmptyFile().Body()
	
	// Set a cty.Value directly (should use the regular SetAttributeValue method)
	body.SetAttributeValue("direct_cty", cty.StringVal("test"))
	
	attr := body.GetAttribute("direct_cty")
	if attr == nil {
		t.Fatal("Expected attribute to be set")
	}
	
	tokens := attr.Expr().BuildTokens(nil)
	result := string(tokens.Bytes())
	expected := `"test"`
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// Helper function to normalize HCL for comparison
func normalizeHCL(hcl string) string {
	// Remove extra whitespace and normalize
	lines := strings.Split(hcl, "\n")
	var normalized []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			normalized = append(normalized, line)
		}
	}
	return strings.Join(normalized, "\n")
}