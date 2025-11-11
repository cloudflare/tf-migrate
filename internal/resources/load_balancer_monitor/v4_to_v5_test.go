package load_balancer_monitor

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	expected := "cloudflare_load_balancer_monitor"
	if got := migrator.GetResourceType(); got != expected {
		t.Errorf("GetResourceType() = %v, want %v", got, expected)
	}
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		want         bool
	}{
		{"handles correct type", "cloudflare_load_balancer_monitor", true},
		{"rejects other types", "cloudflare_load_balancer", false},
		{"rejects empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := migrator.CanHandle(tt.resourceType); got != tt.want {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestPreprocess(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no header blocks - unchanged",
			input:    `resource "cloudflare_load_balancer_monitor" "test" { account_id = "abc" }`,
			expected: `resource "cloudflare_load_balancer_monitor" "test" { account_id = "abc" }`,
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name: "content without load_balancer_monitor resources",
			input: `resource "cloudflare_other" "test" {
  header {
    header = "Host"
    values = ["example.com"]
  }
}`,
			expected: `resource "cloudflare_other" "test" {
  header {
    header = "Host"
    values = ["example.com"]
  }
}`,
		},
		{
			name: "header block with malformed header name - unchanged",
			input: `resource "cloudflare_load_balancer_monitor" "test" {
  account_id = "abc"
  header {
    badfield = "Host"
    values = ["example.com"]
  }
}`,
			expected: `resource "cloudflare_load_balancer_monitor" "test" {
  account_id = "abc"
  header {
    badfield = "Host"
    values = ["example.com"]
  }
}`,
		},
		{
			name: "header block with malformed values - unchanged",
			input: `resource "cloudflare_load_balancer_monitor" "test" {
  account_id = "abc"
  header {
    header = "Host"
    badfield = ["example.com"]
  }
}`,
			expected: `resource "cloudflare_load_balancer_monitor" "test" {
  account_id = "abc"
  header {
    header = "Host"
    badfield = ["example.com"]
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := migrator.Preprocess(tt.input)
			if got != tt.expected {
				t.Errorf("Preprocess() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "minimal monitor - no transformation needed",
				Input: `
resource "cloudflare_load_balancer_monitor" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			},
			{
				Name: "HTTP monitor with all fields",
				Input: `
resource "cloudflare_load_balancer_monitor" "http" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  description      = "Production HTTP monitor"
  type             = "http"
  method           = "GET"
  path             = "/health"
  interval         = 30
  retries          = 3
  timeout          = 10
  expected_codes   = "2xx"
  expected_body    = "healthy"
  follow_redirects = true
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "http" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  description      = "Production HTTP monitor"
  type             = "http"
  method           = "GET"
  path             = "/health"
  interval         = 30
  retries          = 3
  timeout          = 10
  expected_codes   = "2xx"
  expected_body    = "healthy"
  follow_redirects = true
}`,
			},
			{
				Name: "monitor with single header block",
				Input: `
resource "cloudflare_load_balancer_monitor" "with_header" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "http"
  path       = "/health"

  header {
    header = "Host"
    values = ["example.com"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "with_header" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "http"
  path       = "/health"

  header = {
    "Host" = ["example.com"]
  }
}`,
			},
			{
				Name: "monitor with multiple header blocks",
				Input: `
resource "cloudflare_load_balancer_monitor" "with_headers" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "https"

  header {
    header = "Host"
    values = ["api.example.com"]
  }

  header {
    header = "Authorization"
    values = ["Bearer token123"]
  }

  header {
    header = "X-Custom"
    values = ["value1", "value2", "value3"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "with_headers" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "https"

  header = {
    "Host" = ["api.example.com"]
    "Authorization" = ["Bearer token123"]
    "X-Custom" = ["value1", "value2", "value3"]
  }
}`,
			},
			{
				Name: "TCP monitor with port",
				Input: `
resource "cloudflare_load_balancer_monitor" "tcp" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "tcp"
  port       = 3306
  method     = "connection_established"
  interval   = 60
  timeout    = 5
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "tcp" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "tcp"
  port       = 3306
  method     = "connection_established"
  interval   = 60
  timeout    = 5
}`,
			},
			{
				Name: "HTTPS monitor with allow_insecure",
				Input: `
resource "cloudflare_load_balancer_monitor" "https_insecure" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  type           = "https"
  path           = "/api/status"
  allow_insecure = true
  expected_codes = "200"
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "https_insecure" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  type           = "https"
  path           = "/api/status"
  allow_insecure = true
  expected_codes = "200"
}`,
			},
			{
				Name: "monitor with consecutive checks",
				Input: `
resource "cloudflare_load_balancer_monitor" "consecutive" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  consecutive_up   = 2
  consecutive_down = 3
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "consecutive" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  consecutive_up   = 2
  consecutive_down = 3
}`,
			},
			{
				Name: "multiple monitors in one file",
				Input: `
resource "cloudflare_load_balancer_monitor" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "http"

  header {
    header = "Host"
    values = ["first.example.com"]
  }
}

resource "cloudflare_load_balancer_monitor" "second" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "https"

  header {
    header = "Host"
    values = ["second.example.com"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "http"

  header = {
    "Host" = ["first.example.com"]
  }
}

resource "cloudflare_load_balancer_monitor" "second" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "https"

  header = {
    "Host" = ["second.example.com"]
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "minimal monitor state",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe"
  }
}`,
			},
			{
				Name: "monitor with numeric fields - converted to float64",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "interval": 60,
    "port": 8080,
    "retries": 2,
    "timeout": 5,
    "consecutive_down": 3,
    "consecutive_up": 2
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "interval": 60.0,
    "port": 8080.0,
    "retries": 2.0,
    "timeout": 5.0,
    "consecutive_down": 3.0,
    "consecutive_up": 2.0
  }
}`,
			},
			{
				Name: "monitor with header array-of-objects - converted to map-of-arrays",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "header": [
      {
        "header": "Host",
        "values": ["example.com"]
      },
      {
        "header": "X-Custom",
        "values": ["value1", "value2"]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "header": {
      "Host": ["example.com"],
      "X-Custom": ["value1", "value2"]
    }
  }
}`,
			},
			{
				Name: "full monitor with all transformations",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "type": "https",
    "interval": 30,
    "port": 443,
    "retries": 3,
    "timeout": 10,
    "consecutive_down": 2,
    "consecutive_up": 1,
    "header": [
      {
        "header": "Host",
        "values": ["api.example.com"]
      },
      {
        "header": "Authorization",
        "values": ["Bearer token"]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "type": "https",
    "interval": 30.0,
    "port": 443.0,
    "retries": 3.0,
    "timeout": 10.0,
    "consecutive_down": 2.0,
    "consecutive_up": 1.0,
    "header": {
      "Host": ["api.example.com"],
      "Authorization": ["Bearer token"]
    }
  }
}`,
			},
			{
				Name: "monitor with empty header array - removed",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "header": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe"
  }
}`,
			},
			{
				Name: "monitor with single header",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "header": [
      {
        "header": "Host",
        "values": ["example.com", "www.example.com"]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "header": {
      "Host": ["example.com", "www.example.com"]
    }
  }
}`,
			},
			{
				Name: "state without attributes - unchanged except schema_version",
				Input: `{
  "schema_version": 1
}`,
				Expected: `{
  "schema_version": 1
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
