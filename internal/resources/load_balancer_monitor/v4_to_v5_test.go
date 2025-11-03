package load_balancer_monitor

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "single header transformation",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "header": [
          {
            "header": "Host",
            "values": ["api.example.com"]
          }
        ]
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "header": {
          "Host": ["api.example.com"]
        }
      }
    }
  ]
}`,
			},
			{
				Name: "multiple headers transformation",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "header": [
          {
            "header": "Host",
            "values": ["api.example.com"]
          },
          {
            "header": "X-App-ID",
            "values": ["abc123", "def456"]
          }
        ]
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "header": {
          "Host": ["api.example.com"],
          "X-App-ID": ["abc123", "def456"]
        }
      }
    }
  ]
}`,
			},
			{
				Name: "empty header array removes attribute",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "header": []
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123"
      }
    }
  ]
}`,
			},
			{
				Name: "full monitor with all attributes",
				Input: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "api-healthcheck",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "expected_body": "healthy",
        "expected_codes": "200",
        "method": "GET",
        "timeout": 5,
        "path": "/health",
        "interval": 60,
        "retries": 2,
        "port": 443,
        "description": "API health check",
        "header": [
          {
            "header": "Host",
            "values": ["api.example.com"]
          },
          {
            "header": "X-Custom-Header",
            "values": ["value1", "value2"]
          }
        ]
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_load_balancer_monitor",
  "name": "api-healthcheck",
  "instances": [
    {
      "attributes": {
        "id": "monitor123",
        "account_id": "account123",
        "expected_body": "healthy",
        "expected_codes": "200",
        "method": "GET",
        "timeout": 5,
        "path": "/health",
        "interval": 60,
        "retries": 2,
        "port": 443,
        "description": "API health check",
        "header": {
          "Host": ["api.example.com"],
          "X-Custom-Header": ["value1", "value2"]
        }
      }
    }
  ]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
