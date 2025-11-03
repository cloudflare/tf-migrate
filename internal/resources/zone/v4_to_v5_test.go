package zone

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test config transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic zone transformation",
				Input: `resource "cloudflare_zone" "example" {
  zone       = "example.com"
  account_id = "abc123"
}`,
				Expected: `resource "cloudflare_zone" "example" {
  name = "example.com"
  account = {
    id = "abc123"
  }
}`,
			},
			{
				Name: "zone with all v4 attributes",
				Input: `resource "cloudflare_zone" "example" {
  zone                = "example.com"
  account_id          = "abc123"
  paused              = true
  type                = "partial"
  jump_start          = true
  plan                = "enterprise"
  vanity_name_servers = ["ns1.example.com", "ns2.example.com"]
}`,
				Expected: `resource "cloudflare_zone" "example" {
  paused              = true
  type                = "partial"
  vanity_name_servers = ["ns1.example.com", "ns2.example.com"]
  name                = "example.com"
  account = {
    id = "abc123"
  }
}`,
			},
			{
				Name: "zone with only removed attributes",
				Input: `resource "cloudflare_zone" "example" {
  zone       = "example.com"
  account_id = "abc123"
  jump_start = false
  plan       = "free"
}`,
				Expected: `resource "cloudflare_zone" "example" {
  name = "example.com"
  account = {
    id = "abc123"
  }
}`,
			},
			{
				Name: "zone with variable reference in account_id",
				Input: `resource "cloudflare_zone" "complex" {
  zone       = "complex.example.com"
  account_id = var.account_id
  jump_start = var.enable_jump_start
}`,
				Expected: `resource "cloudflare_zone" "complex" {
  name = "complex.example.com"
  account = {
    id = var.account_id
  }
}`,
			},
			{
				Name: "multiple zones in same config",
				Input: `resource "cloudflare_zone" "primary" {
  zone       = "primary.example.com"
  account_id = "account1"
  plan       = "pro"
}

resource "cloudflare_zone" "secondary" {
  zone       = "secondary.example.com"
  account_id = "account2"
  type       = "partial"
  jump_start = true
}`,
				Expected: `resource "cloudflare_zone" "primary" {
  name = "primary.example.com"
  account = {
    id = "account1"
  }
}

resource "cloudflare_zone" "secondary" {
  type = "partial"
  name = "secondary.example.com"
  account = {
    id = "account2"
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
				Name: "basic zone state transformation",
				Input: `{
  "type": "cloudflare_zone",
  "name": "example",
  "instances": [
    {
      "attributes": {
        "zone": "example.com",
        "account_id": "abc123",
        "id": "zone123"
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_zone",
  "name": "example",
  "instances": [
    {
      "attributes": {
        "name": "example.com",
        "account": {
          "id": "abc123"
        },
        "id": "zone123"
      }
    }
  ]
}`,
			},
			{
				Name: "zone state with removed attributes",
				Input: `{
  "type": "cloudflare_zone",
  "name": "example",
  "instances": [
    {
      "attributes": {
        "zone": "example.com",
        "account_id": "abc123",
        "jump_start": true,
        "plan": "enterprise",
        "paused": false
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_zone",
  "name": "example",
  "instances": [
    {
      "attributes": {
        "name": "example.com",
        "account": {
          "id": "abc123"
        },
        "paused": false
      }
    }
  ]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
