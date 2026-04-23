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
// and a dynamic "domains" block are present, the dynamic block is converted first
// and the static blocks are converted afterwards.  Because both set the same
// attribute name, the static array takes precedence (last write wins) and a
// DiagWarning is emitted so the user knows about the mixed case.
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

	// No diagnostics expected — toset is a literal, so no opaque warning.
	if len(ctx.Diagnostics) != 0 {
		t.Errorf("Expected 0 diagnostics for toset case, got %d: %v", len(ctx.Diagnostics), ctx.Diagnostics)
	}

	// The output should not contain any dynamic block.
	output := string(file.Bytes())
	if strings.Contains(output, `dynamic "domains"`) {
		t.Error("Output still contains dynamic block — expected conversion")
	}
	// The output should contain a 'domains' attribute.
	if !strings.Contains(output, "domains") {
		t.Errorf("Expected 'domains' attribute in output, got:\n%s", output)
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
