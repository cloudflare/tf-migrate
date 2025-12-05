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

func TestV4ToV5StateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "HTTP healthcheck with all fields",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "http-check",
    "address": "example.com",
    "type": "HTTP",
    "port": 80,
    "path": "/health",
    "method": "GET",
    "expected_codes": ["200", "201"],
    "expected_body": "OK",
    "follow_redirects": false,
    "allow_insecure": false,
    "consecutive_fails": 3,
    "consecutive_successes": 2,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "suspended": false,
    "check_regions": ["WNAM", "ENAM"]
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "http-check",
    "address": "example.com",
    "type": "HTTP",
    "consecutive_fails": 3,
    "consecutive_successes": 2,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "suspended": false,
    "check_regions": ["WNAM", "ENAM"],
    "http_config": {
      "port": 80,
      "path": "/health",
      "method": "GET",
      "expected_codes": ["200", "201"],
      "expected_body": "OK",
      "follow_redirects": false,
      "allow_insecure": false
    }
  },
  "schema_version": 0
}`,
		},
		{
			Name: "HTTP healthcheck minimal",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "minimal",
  "attributes": {
    "zone_id": "abc123",
    "name": "minimal-check",
    "address": "example.com",
    "type": "HTTP"
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "minimal",
  "attributes": {
    "zone_id": "abc123",
    "name": "minimal-check",
    "address": "example.com",
    "type": "HTTP"
  },
  "schema_version": 0
}`,
		},
		{
			Name: "HTTPS healthcheck",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "https",
  "attributes": {
    "zone_id": "abc123",
    "name": "https-check",
    "address": "example.com",
    "type": "HTTPS",
    "port": 443,
    "path": "/api/health",
    "method": "HEAD",
    "allow_insecure": true,
    "expected_codes": ["200"]
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "https",
  "attributes": {
    "zone_id": "abc123",
    "name": "https-check",
    "address": "example.com",
    "type": "HTTPS",
    "http_config": {
      "port": 443,
      "path": "/api/health",
      "method": "HEAD",
      "allow_insecure": true,
      "expected_codes": ["200"]
    }
  },
  "schema_version": 0
}`,
		},
		{
			Name: "TCP healthcheck",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "tcp",
  "attributes": {
    "zone_id": "abc123",
    "name": "tcp-check",
    "address": "10.0.0.1",
    "type": "TCP",
    "port": 8080,
    "method": "connection_established",
    "consecutive_fails": 2,
    "timeout": 10
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "tcp",
  "attributes": {
    "zone_id": "abc123",
    "name": "tcp-check",
    "address": "10.0.0.1",
    "type": "TCP",
    "consecutive_fails": 2,
    "timeout": 10,
    "tcp_config": {
      "port": 8080,
      "method": "connection_established"
    }
  },
  "schema_version": 0
}`,
		},
		{
			Name: "Header transformation - single header",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "header": [
      {
        "header": "Host",
        "values": ["example.com"]
      }
    ]
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "http_config": {
      "header": {
        "Host": ["example.com"]
      }
    }
  },
  "schema_version": 0
}`,
		},
		{
			Name: "Header transformation - multiple headers",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTPS",
    "header": [
      {
        "header": "Host",
        "values": ["example.com"]
      },
      {
        "header": "User-Agent",
        "values": ["HealthChecker/1.0"]
      },
      {
        "header": "Accept",
        "values": ["application/json", "text/plain"]
      }
    ]
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTPS",
    "http_config": {
      "header": {
        "Host": ["example.com"],
        "User-Agent": ["HealthChecker/1.0"],
        "Accept": ["application/json", "text/plain"]
      }
    }
  },
  "schema_version": 0
}`,
		},
		{
			Name: "Computed fields preserved",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "id": "health123",
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "created_on": "2024-01-01T00:00:00Z",
    "modified_on": "2024-01-01T00:00:00Z",
    "status": "healthy",
    "failure_reason": ""
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "id": "health123",
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "created_on": "2024-01-01T00:00:00Z",
    "modified_on": "2024-01-01T00:00:00Z",
    "status": "healthy",
    "failure_reason": ""
  },
  "schema_version": 0
}`,
		},
		{
			Name: "Type conversions - integers to float64",
			Input: `{
  "type": "cloudflare_healthcheck",
  "name": "conversions",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "consecutive_fails": 1,
    "consecutive_successes": 1,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "port": 80
  }
}`,
			Expected: `{
  "type": "cloudflare_healthcheck",
  "name": "conversions",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "consecutive_fails": 1.0,
    "consecutive_successes": 1.0,
    "retries": 2.0,
    "timeout": 5.0,
    "interval": 60.0,
    "http_config": {
      "port": 80.0
    }
  },
  "schema_version": 0
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
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
