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
		{
			// No type attribute: defaults to HTTP path, no http_config created (no HTTP fields present)
			Name: "No type attribute defaults to HTTP path",
			Input: `resource "cloudflare_healthcheck" "no_type" {
  zone_id = "abc123"
  name    = "no-type-check"
  address = "example.com"
}`,
			Expected: `resource "cloudflare_healthcheck" "no_type" {
  zone_id = "abc123"
  name    = "no-type-check"
  address = "example.com"
}`,
		},
		{
			// TCP with only type (no method/port): tcp_config not created since no fields to move
			Name: "TCP healthcheck with no optional fields",
			Input: `resource "cloudflare_healthcheck" "tcp_minimal" {
  zone_id = "abc123"
  name    = "tcp-minimal"
  address = "10.0.0.1"
  type    = "TCP"
}`,
			Expected: `resource "cloudflare_healthcheck" "tcp_minimal" {
  zone_id = "abc123"
  name    = "tcp-minimal"
  address = "10.0.0.1"
  type    = "TCP"
}`,
		},
		{
			// Common fields (check_regions, consecutive_fails, etc.) stay at top-level in v5
			Name: "Common fields stay top-level",
			Input: `resource "cloudflare_healthcheck" "common_fields" {
  zone_id               = "abc123"
  name                  = "common-fields"
  address               = "example.com"
  type                  = "HTTP"
  description           = "My healthcheck"
  consecutive_fails     = 3
  consecutive_successes = 2
  retries               = 2
  timeout               = 5
  interval              = 60
  suspended             = false
  check_regions         = ["WNAM", "ENAM"]
  expected_codes        = ["200"]
}`,
			Expected: `resource "cloudflare_healthcheck" "common_fields" {
  zone_id               = "abc123"
  name                  = "common-fields"
  address               = "example.com"
  type                  = "HTTP"
  description           = "My healthcheck"
  consecutive_fails     = 3
  consecutive_successes = 2
  retries               = 2
  timeout               = 5
  interval              = 60
  suspended             = false
  check_regions         = ["WNAM", "ENAM"]
  http_config = {
    expected_codes = ["200"]
  }
}`,
		},
		{
			// Suspended healthcheck: suspended stays top-level, http fields go into http_config
			Name: "Suspended HTTP healthcheck",
			Input: `resource "cloudflare_healthcheck" "suspended" {
  zone_id   = "abc123"
  name      = "suspended-check"
  address   = "example.com"
  type      = "HTTP"
  suspended = true
  path      = "/health"
  method    = "GET"
}`,
			Expected: `resource "cloudflare_healthcheck" "suspended" {
  zone_id   = "abc123"
  name      = "suspended-check"
  address   = "example.com"
  type      = "HTTP"
  suspended = true
  http_config = {
    method = "GET"
    path   = "/health"
  }
}`,
		},
		{
			// Lifecycle meta-arguments are preserved unchanged
			Name: "HTTP healthcheck with lifecycle",
			Input: `resource "cloudflare_healthcheck" "with_lifecycle" {
  zone_id = "abc123"
  name    = "lifecycle-check"
  address = "example.com"
  type    = "HTTP"
  path    = "/health"
  method  = "GET"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}`,
			Expected: `resource "cloudflare_healthcheck" "with_lifecycle" {
  zone_id = "abc123"
  name    = "lifecycle-check"
  address = "example.com"
  type    = "HTTP"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
  http_config = {
    method = "GET"
    path   = "/health"
  }
}`,
		},
		{
			// for_each resources are transformed correctly
			Name: "HTTP healthcheck with for_each",
			Input: `resource "cloudflare_healthcheck" "multi" {
  for_each = toset(["api", "web"])

  zone_id = "abc123"
  name    = "check-${each.value}"
  address = "example.com"
  type    = "HTTP"

  port   = 80
  path   = "/${each.value}/health"
  method = "GET"

  check_regions  = ["WNAM"]
  expected_codes = ["200"]
}`,
			Expected: `resource "cloudflare_healthcheck" "multi" {
  for_each = toset(["api", "web"])

  zone_id = "abc123"
  name    = "check-${each.value}"
  address = "example.com"
  type    = "HTTP"

  check_regions = ["WNAM"]
  http_config = {
    expected_codes = ["200"]
    method         = "GET"
    path           = "/${each.value}/health"
    port           = 80
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
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
