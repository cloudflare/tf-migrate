package zero_trust_local_fallback_domain

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic default profile - no policy_id",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "default profile with all domain fields",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix      = "example.com"
    description = "Example domain"
    dns_server  = ["1.1.1.1", "8.8.8.8"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "example.com"
      description = "Example domain"
      dns_server  = ["1.1.1.1", "8.8.8.8"]
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "default profile with multiple domains",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix      = "example.com"
    description = "Example domain"
  }

  domains {
    suffix     = "another.com"
    dns_server = ["1.1.1.1"]
  }

  domains {
    suffix = "third.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "example.com"
      description = "Example domain"
    },
    {
      suffix     = "another.com"
      dns_server = ["1.1.1.1"]
    },
    {
      suffix = "third.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "deprecated resource name - default profile",
			Input: `
resource "cloudflare_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "default profile with policy_id = null",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = null

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "basic custom profile - with policy_id",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "custom profile with all domain fields",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix      = "corp.example.com"
    description = "Corporate domain"
    dns_server  = ["10.0.0.1"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix      = "corp.example.com"
      description = "Corporate domain"
      dns_server  = ["10.0.0.1"]
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "custom profile with multiple domains",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "corp1.example.com"
  }

  domains {
    suffix      = "corp2.example.com"
    description = "Secondary corporate domain"
    dns_server  = ["10.0.0.2", "10.0.0.3"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "corp1.example.com"
    },
    {
      suffix      = "corp2.example.com"
      description = "Secondary corporate domain"
      dns_server  = ["10.0.0.2", "10.0.0.3"]
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "deprecated resource name - custom profile",
			Input: `
resource "cloudflare_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		// ----------------------------------------------------------------
		// Dynamic "domains" block test cases (ticket #002)
		// ----------------------------------------------------------------
		{
			Name: "dynamic domains block with toset - default profile",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  dynamic "domains" {
    for_each = toset(["intranet", "corp", "local"])
    content {
      suffix = domains.value
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [for value in toset(["intranet", "corp", "local"]) : { suffix = value }]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "dynamic domains block with toset - custom profile",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  dynamic "domains" {
    for_each = toset(["intranet", "corp", "local"])
    content {
      suffix = domains.value
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [for value in toset(["intranet", "corp", "local"]) : { suffix = value }]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "dynamic domains block with opaque for_each variable",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  dynamic "domains" {
    for_each = local.domain_list
    content {
      suffix      = domains.value.suffix
      description = domains.value.description
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [for value in local.domain_list : {
    suffix      = value.suffix
    description = value.description
  }]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "deprecated resource name with dynamic domains block",
			Input: `
resource "cloudflare_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  dynamic "domains" {
    for_each = toset(["intranet", "corp", "local"])
    content {
      suffix = domains.value
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [for value in toset(["intranet", "corp", "local"]) : { suffix = value }]
}

moved {
  from = cloudflare_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "multiple resources - mixed default and custom",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "default.example.com"
  }
}

resource "cloudflare_zero_trust_local_fallback_domain" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "custom.example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "default.example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.default
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.default
}

resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "custom.example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.custom
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.custom
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// TestDynamicDomainsDiagnostics verifies that an opaque for_each triggers a
// DiagWarning while still producing valid converted HCL output.
func TestDynamicDomainsDiagnostics(t *testing.T) {
	input := `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  dynamic "domains" {
    for_each = local.domain_list
    content {
      suffix = domains.value.suffix
    }
  }
}`

	migrator := NewV4ToV5Migrator()

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("Failed to parse input HCL: %v", diags)
	}

	ctx := &transform.Context{
		Content:  []byte(input),
		Filename: "test.tf",
		CFGFile:  file,
	}

	body := file.Body()
	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := block.Labels()[0]
			if migrator.CanHandle(resourceType) {
				result, err := migrator.TransformConfig(ctx, block)
				if err != nil {
					t.Fatalf("TransformConfig returned error: %v", err)
				}
				if result != nil && result.RemoveOriginal {
					body.RemoveBlock(block)
					for _, newBlock := range result.Blocks {
						body.AppendBlock(newBlock)
					}
				}
			}
		}
	}

	// Should have emitted exactly one DiagWarning for the opaque for_each.
	if len(ctx.Diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic for opaque for_each, got %d", len(ctx.Diagnostics))
	}
	if ctx.Diagnostics[0].Severity != hcl.DiagWarning {
		t.Errorf("Expected DiagWarning severity, got %v", ctx.Diagnostics[0].Severity)
	}
	if !strings.Contains(ctx.Diagnostics[0].Summary, "manual verification") {
		t.Errorf("Unexpected diagnostic summary: %s", ctx.Diagnostics[0].Summary)
	}

	// The HCL output must contain a for-expression (not a dynamic block).
	output := string(file.Bytes())
	if strings.Contains(output, `dynamic "domains"`) {
		t.Error("Output still contains dynamic block — expected for-expression conversion")
	}
	if !strings.Contains(output, "for value in local.domain_list") {
		t.Errorf("Expected for-expression referencing local.domain_list, got:\n%s", output)
	}
}

// TestMixedStaticAndDynamicDomains verifies that when both static domains blocks
// and a dynamic "domains" block are present, both are preserved via concat().
func TestMixedStaticAndDynamicDomains(t *testing.T) {
	input := `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "static.example.com"
  }

  dynamic "domains" {
    for_each = toset(["intranet", "corp"])
    content {
      suffix = domains.value
    }
  }
}`

	migrator := NewV4ToV5Migrator()

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("Failed to parse input HCL: %v", diags)
	}

	ctx := &transform.Context{
		Content:  []byte(input),
		Filename: "test.tf",
		CFGFile:  file,
	}

	body := file.Body()
	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := block.Labels()[0]
			if migrator.CanHandle(resourceType) {
				result, err := migrator.TransformConfig(ctx, block)
				if err != nil {
					t.Fatalf("TransformConfig returned error: %v", err)
				}
				if result != nil && result.RemoveOriginal {
					body.RemoveBlock(block)
					for _, newBlock := range result.Blocks {
						body.AppendBlock(newBlock)
					}
				}
			}
		}
	}

	output := string(file.Bytes())

	// The output should not contain any dynamic block.
	if strings.Contains(output, `dynamic "domains"`) {
		t.Error("Output still contains dynamic block — expected conversion")
	}

	// Both static and dynamic content should be preserved via concat.
	if !strings.Contains(output, "concat(") {
		t.Errorf("Expected concat() to merge static and dynamic domains, got:\n%s", output)
	}
	if !strings.Contains(output, "static.example.com") {
		t.Errorf("Expected static.example.com to be preserved in output, got:\n%s", output)
	}
	if !strings.Contains(output, "toset") {
		t.Errorf("Expected toset for-expression to be preserved in output, got:\n%s", output)
	}

	// Should have a mixed-merge diagnostic warning.
	foundMixedWarning := false
	for _, d := range ctx.Diagnostics {
		if strings.Contains(d.Summary, "Mixed static and dynamic") {
			foundMixedWarning = true
			break
		}
	}
	if !foundMixedWarning {
		t.Errorf("Expected a diagnostic warning about mixed static and dynamic domains, got: %v", ctx.Diagnostics)
	}
}

// TestMultipleDynamicDomainsBlocks verifies that when multiple dynamic "domains"
// blocks exist in a single resource, all of them are preserved via concat().
// This is the regression test for https://github.com/cloudflare/tf-migrate/issues/288.
func TestMultipleDynamicDomainsBlocks(t *testing.T) {
	input := `
resource "cloudflare_fallback_domain" "remote_region_profile_fallback_domains" {
  account_id = var.account_id
  policy_id  = cloudflare_zero_trust_device_custom_profile.remote_region_profile.id

  dynamic "domains" {
    for_each = local.primary_domain_fallback_entries
    content {
      suffix = domains.value
    }
  }

  dynamic "domains" {
    for_each = local.static_domain_fallback_entries
    content {
      suffix      = domains.value
      dns_server  = ["1.1.1.1"]
      description = "Static domains resolved via regional DNS"
    }
  }

  dynamic "domains" {
    for_each = local.dev_domain_fallback_entries
    content {
      suffix      = domains.value
      dns_server  = ["1.1.1.1"]
      description = "Dev domains resolved via regional DNS"
    }
  }
}
`

	migrator := NewV4ToV5Migrator()

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("Failed to parse input HCL: %v", diags)
	}

	ctx := &transform.Context{
		Content:  []byte(input),
		Filename: "test.tf",
		CFGFile:  file,
	}

	body := file.Body()
	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := block.Labels()[0]
			if migrator.CanHandle(resourceType) {
				result, err := migrator.TransformConfig(ctx, block)
				if err != nil {
					t.Fatalf("TransformConfig returned error: %v", err)
				}
				if result != nil && result.RemoveOriginal {
					body.RemoveBlock(block)
					for _, newBlock := range result.Blocks {
						body.AppendBlock(newBlock)
					}
				}
			}
		}
	}

	output := string(file.Bytes())

	// All dynamic blocks must be removed
	if strings.Contains(output, `dynamic "domains"`) {
		t.Error("Output still contains dynamic block — expected for-expression conversion")
	}

	// Must use concat() to merge all three dynamic blocks
	if !strings.Contains(output, "concat(") {
		t.Errorf("Expected concat() to merge multiple dynamic blocks, got:\n%s", output)
	}

	// All three for_each collections must be present — none silently dropped
	for _, expr := range []string{
		"local.primary_domain_fallback_entries",
		"local.static_domain_fallback_entries",
		"local.dev_domain_fallback_entries",
	} {
		if !strings.Contains(output, expr) {
			t.Errorf("Expected %q in output (was silently dropped!), got:\n%s", expr, output)
		}
	}

	// Verify resource was renamed to v5 custom type (has policy_id)
	if !strings.Contains(output, "cloudflare_zero_trust_device_custom_profile_local_domain_fallback") {
		t.Errorf("Expected v5 custom profile resource type in output, got:\n%s", output)
	}

	// Should have a diagnostic warning about multiple dynamic blocks
	foundMultiBlockWarning := false
	for _, d := range ctx.Diagnostics {
		if strings.Contains(d.Summary, "Multiple dynamic") {
			foundMultiBlockWarning = true
			break
		}
	}
	if !foundMultiBlockWarning {
		t.Error("Expected a diagnostic warning about multiple dynamic 'domains' blocks being merged")
	}
}

// TestMixedStaticAndDynamicDomainsMerge verifies that when both static domains
// blocks and dynamic "domains" blocks are present, they are merged correctly
// via concat() rather than the dynamic result being silently overwritten.
func TestMixedStaticAndDynamicDomainsMerge(t *testing.T) {
	input := `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "static.example.com"
  }

  dynamic "domains" {
    for_each = local.dynamic_entries
    content {
      suffix = domains.value
    }
  }

  domains {
    suffix      = "other.example.com"
    description = "other static"
  }
}
`

	migrator := NewV4ToV5Migrator()

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("Failed to parse input HCL: %v", diags)
	}

	ctx := &transform.Context{
		Content:  []byte(input),
		Filename: "test.tf",
		CFGFile:  file,
	}

	body := file.Body()
	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := block.Labels()[0]
			if migrator.CanHandle(resourceType) {
				result, err := migrator.TransformConfig(ctx, block)
				if err != nil {
					t.Fatalf("TransformConfig returned error: %v", err)
				}
				if result != nil && result.RemoveOriginal {
					body.RemoveBlock(block)
					for _, newBlock := range result.Blocks {
						body.AppendBlock(newBlock)
					}
				}
			}
		}
	}

	output := string(file.Bytes())

	// Must use concat to merge both static and dynamic
	if !strings.Contains(output, "concat(") {
		t.Errorf("Expected concat() to merge static and dynamic domains, got:\n%s", output)
	}

	// Dynamic for-expression must be present
	if !strings.Contains(output, "local.dynamic_entries") {
		t.Errorf("Expected dynamic for-expression in output, got:\n%s", output)
	}

	// Both static entries must be present
	if !strings.Contains(output, "static.example.com") {
		t.Errorf("Expected static.example.com in output, got:\n%s", output)
	}
	if !strings.Contains(output, "other.example.com") {
		t.Errorf("Expected other.example.com in output, got:\n%s", output)
	}

	// No dynamic blocks should remain
	if strings.Contains(output, `dynamic "domains"`) {
		t.Error("Output still contains dynamic block")
	}

	// Should have a mixed-merge diagnostic
	foundMixedWarning := false
	for _, d := range ctx.Diagnostics {
		if strings.Contains(d.Summary, "Mixed static and dynamic") {
			foundMixedWarning = true
			break
		}
	}
	if !foundMixedWarning {
		t.Error("Expected a diagnostic warning about mixed static and dynamic domains")
	}
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "handles current v4 name",
			resourceType: "cloudflare_zero_trust_local_fallback_domain",
			expected:     true,
		},
		{
			name:         "handles deprecated v4 name",
			resourceType: "cloudflare_fallback_domain",
			expected:     true,
		},
		{
			name:         "does not handle unrelated resource",
			resourceType: "cloudflare_other_resource",
			expected:     false,
		},
		{
			name:         "does not handle v5 default profile name",
			resourceType: "cloudflare_zero_trust_device_default_profile_local_domain_fallback",
			expected:     false,
		},
		{
			name:         "does not handle v5 custom profile name",
			resourceType: "cloudflare_zero_trust_device_custom_profile_local_domain_fallback",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}
