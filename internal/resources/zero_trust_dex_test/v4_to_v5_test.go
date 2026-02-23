package zero_trust_dex_test

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic HTTP test",
			Input: `
resource "cloudflare_zero_trust_dex_test" "http_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "HTTP Test"
  description = "Test HTTP connectivity"
  interval    = "0h30m0s"
  enabled     = true

  data {
    kind   = "http"
    host   = "https://example.com"
    method = "GET"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_dex_test" "http_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "HTTP Test"
  description = "Test HTTP connectivity"
  interval    = "0h30m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://example.com"
    method = "GET"
  }
}

moved {
  from = cloudflare_device_dex_test.http_test
  to   = cloudflare_zero_trust_dex_test.http_test
}
`,
		},
		{
			Name: "traceroute test",
			Input: `
resource "cloudflare_zero_trust_dex_test" "traceroute_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Traceroute Test"
  description = "Test network path"
  interval    = "1h0m0s"
  enabled     = true

  data {
    kind = "traceroute"
    host = "8.8.8.8"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_dex_test" "traceroute_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Traceroute Test"
  description = "Test network path"
  interval    = "1h0m0s"
  enabled     = true

  data = {
    kind = "traceroute"
    host = "8.8.8.8"
  }
}

moved {
  from = cloudflare_device_dex_test.traceroute_test
  to   = cloudflare_zero_trust_dex_test.traceroute_test
}
`,
		},
		{
			Name: "disabled test",
			Input: `
resource "cloudflare_zero_trust_dex_test" "disabled" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Disabled Test"
  description = "Currently disabled"
  interval    = "0h15m0s"
  enabled     = false

  data {
    kind   = "http"
    host   = "https://internal.example.com"
    method = "GET"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_dex_test" "disabled" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Disabled Test"
  description = "Currently disabled"
  interval    = "0h15m0s"
  enabled     = false

  data = {
    kind   = "http"
    host   = "https://internal.example.com"
    method = "GET"
  }
}

moved {
  from = cloudflare_device_dex_test.disabled
  to   = cloudflare_zero_trust_dex_test.disabled
}
`,
		},
		{
			Name: "with variables",
			Input: `
resource "cloudflare_zero_trust_dex_test" "var_test" {
  account_id  = var.cloudflare_account_id
  name        = var.test_name
  description = var.test_description
  interval    = var.test_interval
  enabled     = var.test_enabled

  data {
    kind   = var.test_kind
    host   = var.test_host
    method = var.test_method
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_dex_test" "var_test" {
  account_id  = var.cloudflare_account_id
  name        = var.test_name
  description = var.test_description
  interval    = var.test_interval
  enabled     = var.test_enabled

  data = {
    kind   = var.test_kind
    host   = var.test_host
    method = var.test_method
  }
}

moved {
  from = cloudflare_device_dex_test.var_test
  to   = cloudflare_zero_trust_dex_test.var_test
}
`,
		},
		{
			Name: "deprecated resource name (cloudflare_device_dex_test)",
			Input: `
resource "cloudflare_device_dex_test" "renamed_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Test with Deprecated Name"
  description = "Using old resource name"
  interval    = "0h30m0s"
  enabled     = true

  data {
    kind   = "http"
    host   = "https://example.com"
    method = "GET"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_dex_test" "renamed_test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Test with Deprecated Name"
  description = "Using old resource name"
  interval    = "0h30m0s"
  enabled     = true

  data = {
    kind   = "http"
    host   = "https://example.com"
    method = "GET"
  }
}

moved {
  from = cloudflare_device_dex_test.renamed_test
  to   = cloudflare_zero_trust_dex_test.renamed_test
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
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
			resourceType: "cloudflare_zero_trust_dex_test",
			expected:     true,
		},
		{
			name:         "handles deprecated v4 name",
			resourceType: "cloudflare_device_dex_test",
			expected:     true,
		},
		{
			name:         "does not handle unrelated resource",
			resourceType: "cloudflare_other_resource",
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
