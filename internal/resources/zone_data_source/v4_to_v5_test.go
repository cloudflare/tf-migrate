package zone_data_source

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "zone_id only - no changes",
				Input: `data "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
				Expected: `data "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
			},
			{
				Name: "zone_id with account_id - remove account_id",
				Input: `data "cloudflare_zone" "example" {
  zone_id    = "abc123"
  account_id = "def456"
}`,
				Expected: `data "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
			},
			{
				Name: "name field moves to filter",
				Input: `data "cloudflare_zone" "example" {
  name = "example.com"
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter = {
    name = "example.com"
  }
}`,
			},
			{
				Name: "name and account_id move to filter",
				Input: `data "cloudflare_zone" "example" {
  name       = "example.com"
  account_id = "abc123"
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter = {
    name = "example.com"
    account = {
      id = "abc123"
    }
  }
}`,
			},
			{
				Name: "account_id only moves to filter",
				Input: `data "cloudflare_zone" "example" {
  account_id = "abc123"
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter = {
    account = {
      id = "abc123"
    }
  }
}`,
			},
			{
				Name: "name with variable",
				Input: `data "cloudflare_zone" "example" {
  name = var.zone_name
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter = {
    name = var.zone_name
  }
}`,
			},
			{
				Name: "multiple datasources in one test",
				Input: `data "cloudflare_zone" "by_id" {
  zone_id = "abc123"
}

data "cloudflare_zone" "by_name" {
  name = "example.com"
}`,
				Expected: `data "cloudflare_zone" "by_id" {
  zone_id = "abc123"
}

data "cloudflare_zone" "by_name" {
  filter = {
    name = "example.com"
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic zone state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "account_id": "def456",
    "name": "example.com",
    "status": "active",
    "paused": false,
    "plan": "free",
    "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"],
    "vanity_name_servers": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "account_id": "def456",
    "name": "example.com",
    "status": "active",
    "paused": false,
    "plan": "free",
    "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"],
    "vanity_name_servers": []
  }
}`,
			},
			{
				Name: "Zone state with null values",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "name": "example.com",
    "status": "active",
    "paused": false,
    "plan": null,
    "name_servers": [],
    "vanity_name_servers": null
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "name": "example.com",
    "status": "active",
    "paused": false,
    "plan": null,
    "name_servers": [],
    "vanity_name_servers": null
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
