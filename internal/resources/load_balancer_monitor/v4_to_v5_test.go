package load_balancer_monitor

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Resource with single header",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header {
    header = "Host"
    values = ["cf-tf-test.com"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header = {
    "Host" = ["cf-tf-test.com"]
  }
}`,
			},
			{
				Name: "Resource with multiple headers",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header {
    header = "Host"
    values = ["cf-tf-test.com"]
  }

  header {
    header = "User-Agent"
    values = ["Mozilla/5.0"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header = {
    "Host"       = ["cf-tf-test.com"]
    "User-Agent" = ["Mozilla/5.0"]
  }
}`,
			},
			{
				Name: "Complete resource with all fields",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id       = "abc123"
  type             = "https"
  interval         = 60
  retries          = 2
  timeout          = 5
  method           = "GET"
  path             = "/health"
  expected_codes   = "2xx"
  expected_body    = "alive"
  follow_redirects = true
  allow_insecure   = false
  description      = "Test monitor"
  port             = 443

  header {
    header = "Host"
    values = ["cf-tf-test.com"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id       = "abc123"
  type             = "https"
  interval         = 60
  retries          = 2
  timeout          = 5
  method           = "GET"
  path             = "/health"
  expected_codes   = "2xx"
  expected_body    = "alive"
  follow_redirects = true
  allow_insecure   = false
  description      = "Test monitor"
  port             = 443

  header = {
    "Host" = ["cf-tf-test.com"]
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "id": "monitor123"
  }
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "id": "monitor123",
    "allow_insecure": null,
    "description": null,
    "expected_body": null,
    "expected_codes": null,
    "follow_redirects": null,
    "probe_zone": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with numeric conversions",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "interval": 60,
    "retries": 2,
    "timeout": 5,
    "port": 443,
    "consecutive_down": 3,
    "consecutive_up": 2
  }
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "interval": 60.0,
    "retries": 2.0,
    "timeout": 5.0,
    "port": 443.0,
    "consecutive_down": 3.0,
    "consecutive_up": 2.0,
    "allow_insecure": null,
    "description": null,
    "expected_body": null,
    "expected_codes": null,
    "follow_redirects": null,
    "probe_zone": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with header transformation",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "header": [
      {
        "header": "Host",
        "values": ["cf-tf-test.com"]
      },
      {
        "header": "User-Agent",
        "values": ["Mozilla/5.0"]
      }
    ]
  }
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "header": {
      "Host": ["cf-tf-test.com"],
      "User-Agent": ["Mozilla/5.0"]
    },
    "allow_insecure": null,
    "description": null,
    "expected_body": null,
    "expected_codes": null,
    "follow_redirects": null,
    "probe_zone": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with empty header array",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "header": []
  }
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "account_id": "abc123",
    "allow_insecure": null,
    "description": null,
    "expected_body": null,
    "expected_codes": null,
    "follow_redirects": null,
    "probe_zone": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Complete state with all fields",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "id": "monitor123",
    "account_id": "abc123",
    "type": "https",
    "interval": 60,
    "retries": 2,
    "timeout": 5,
    "method": "GET",
    "path": "/health",
    "expected_codes": "2xx",
    "expected_body": "alive",
    "follow_redirects": true,
    "allow_insecure": false,
    "description": "Test monitor",
    "port": 443,
    "consecutive_down": 3,
    "consecutive_up": 2,
    "created_on": "2023-01-01T00:00:00Z",
    "modified_on": "2023-01-02T00:00:00Z",
    "header": [
      {
        "header": "Host",
        "values": ["cf-tf-test.com"]
      }
    ]
  }
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "example",
  "attributes": {
    "id": "monitor123",
    "account_id": "abc123",
    "type": "https",
    "interval": 60.0,
    "retries": 2.0,
    "timeout": 5.0,
    "method": "GET",
    "path": "/health",
    "expected_codes": "2xx",
    "expected_body": "alive",
    "follow_redirects": true,
    "allow_insecure": null,
    "description": "Test monitor",
    "port": 443.0,
    "consecutive_down": 3.0,
    "consecutive_up": 2.0,
    "created_on": "2023-01-01T00:00:00Z",
    "modified_on": "2023-01-02T00:00:00Z",
    "header": {
      "Host": ["cf-tf-test.com"]
    },
    "probe_zone": null
  },
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
