package zero_trust_gateway_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "basic unchanged fields",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1
  activate             = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1
  activate             = true
}`,
		},
		{
			Name: "removes custom field",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  custom     = true
  activate   = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  activate   = true
}`,
		},
		{
			Name: "removes gateway_managed field",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 1826
  activate             = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1826
  activate             = true
}`,
		},
		{
			Name: "removes id field from custom certificate config",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  custom     = true
  id         = "existing-cert-uuid-123"
  activate   = false
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  activate   = false
}`,
		},
		{
			Name: "multiple resources in one file",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  custom     = true
  id         = "cert-1"
}

resource "cloudflare_zero_trust_gateway_certificate" "second" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 3650
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}

resource "cloudflare_zero_trust_gateway_certificate" "second" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 3650
}`,
		},
		{
			Name: "comprehensive - all user-defined fields",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "comprehensive" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  custom               = true
  gateway_managed      = false
  id                   = "test-cert-uuid-789"
  validity_period_days = 1826
  activate             = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "comprehensive" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1826
  activate             = true
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "handles cloudflare_zero_trust_gateway_certificate",
			resourceType: "cloudflare_zero_trust_gateway_certificate",
			expected:     true,
		},
		{
			name:         "does not handle other resources",
			resourceType: "cloudflare_teams_list",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	expected := "cloudflare_zero_trust_gateway_certificate"
	result := migrator.GetResourceType()

	if result != expected {
		t.Errorf("GetResourceType() = %q, want %q", result, expected)
	}
}
