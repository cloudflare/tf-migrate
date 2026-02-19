package healthcheck

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5ConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "HTTP healthcheck with all fields",
			Input: `resource "cloudflare_healthcheck" "http_check" {
  zone_id         = "abc123"
  name            = "http-check"
  address         = "example.com"
  type            = "HTTP"
  port            = 80
  path            = "/health"
  method          = "GET"
  expected_codes  = ["200", "201"]
  expected_body   = "OK"
  follow_redirects = false
  allow_insecure  = false
}`,
			Expected: `resource "cloudflare_healthcheck" "http_check" {
  zone_id = "abc123"
  name    = "http-check"
  address = "example.com"
  type    = "HTTP"
  http_config = {
    allow_insecure   = false
    expected_body    = "OK"
    expected_codes   = ["200", "201"]
    follow_redirects = false
    method           = "GET"
    path             = "/health"
    port             = 80
  }
}`,
		},
		{
			Name: "HTTPS healthcheck",
			Input: `resource "cloudflare_healthcheck" "https_check" {
  zone_id        = "abc123"
  name           = "https-check"
  address        = "example.com"
  type           = "HTTPS"
  port           = 443
  path           = "/api/health"
  method         = "HEAD"
  allow_insecure = true
}`,
			Expected: `resource "cloudflare_healthcheck" "https_check" {
  zone_id = "abc123"
  name    = "https-check"
  address = "example.com"
  type    = "HTTPS"
  http_config = {
    allow_insecure = true
    method         = "HEAD"
    path           = "/api/health"
    port           = 443
  }
}`,
		},
		{
			Name: "TCP healthcheck",
			Input: `resource "cloudflare_healthcheck" "tcp_check" {
  zone_id = "abc123"
  name    = "tcp-check"
  address = "10.0.0.1"
  type    = "TCP"
  port    = 8080
  method  = "connection_established"
}`,
			Expected: `resource "cloudflare_healthcheck" "tcp_check" {
  zone_id = "abc123"
  name    = "tcp-check"
  address = "10.0.0.1"
  type    = "TCP"
  tcp_config = {
    method = "connection_established"
    port   = 8080
  }
}`,
		},
		{
			Name: "HTTP with header blocks",
			Input: `resource "cloudflare_healthcheck" "with_headers" {
  zone_id = "abc123"
  name    = "header-check"
  address = "example.com"
  type    = "HTTP"

  header {
    header = "Host"
    values = ["example.com"]
  }

  header {
    header = "User-Agent"
    values = ["HealthChecker/1.0"]
  }
}`,
			Expected: `resource "cloudflare_healthcheck" "with_headers" {
  zone_id = "abc123"
  name    = "header-check"
  address = "example.com"
  type    = "HTTP"

  http_config = {
    header = {
      "Host"       = ["example.com"]
      "User-Agent" = ["HealthChecker/1.0"]
    }
  }
}`,
		},
		{
			Name: "Minimal HTTP healthcheck",
			Input: `resource "cloudflare_healthcheck" "minimal" {
  zone_id = "abc123"
  name    = "minimal-check"
  address = "example.com"
  type    = "HTTP"
}`,
			Expected: `resource "cloudflare_healthcheck" "minimal" {
  zone_id = "abc123"
  name    = "minimal-check"
  address = "example.com"
  type    = "HTTP"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// TestV4ToV5StateTransformation_Removed is a skip marker indicating state tests were removed.
// State transformation is now handled by the provider's StateUpgraders (UpgradeState).
func TestV4ToV5StateTransformation_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

// TestCanHandle verifies the migrator handles the correct resource type
func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"cloudflare_healthcheck", true},
		{"cloudflare_health_check", false},
		{"cloudflare_dns_record", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%s) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

// TestGetResourceType verifies the resource type returned
func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	expected := "cloudflare_healthcheck"
	result := migrator.GetResourceType()

	if result != expected {
		t.Errorf("GetResourceType() = %s, expected %s", result, expected)
	}
}
