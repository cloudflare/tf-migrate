package zone

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "zone_id only - no changes",
			Input: `
data "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
			Expected: `
data "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
		},
		{
			Name: "name only - wrap in filter",
			Input: `
data "cloudflare_zone" "example" {
  name = "example.com"
}`,
			Expected: `
data "cloudflare_zone" "example" {
  filter = {
    name = "example.com"
  }
}`,
		},
		{
			Name: "account_id only - wrap in filter",
			Input: `
data "cloudflare_zone" "example" {
  account_id = "abc123"
}`,
			Expected: `
data "cloudflare_zone" "example" {
  filter = {
    account = {
      id = "abc123"
    }
  }
}`,
		},
		{
			Name: "name and account_id - both in filter",
			Input: `
data "cloudflare_zone" "example" {
  name       = "example.com"
  account_id = "abc123"
}`,
			Expected: `
data "cloudflare_zone" "example" {
  filter = {
    name = "example.com"
    account = {
      id = "abc123"
    }
  }
}`,
		},
		{
			Name: "with variable references",
			Input: `
data "cloudflare_zone" "example" {
  name       = var.zone_name
  account_id = var.account_id
}`,
			Expected: `
data "cloudflare_zone" "example" {
  filter = {
    name = var.zone_name
    account = {
      id = var.account_id
    }
  }
}`,
		},
		{
			Name: "multiple datasources in one file",
			Input: `
data "cloudflare_zone" "example1" {
  zone_id = "abc123"
}

data "cloudflare_zone" "example2" {
  name = "example.com"
}`,
			Expected: `
data "cloudflare_zone" "example1" {
  zone_id = "abc123"
}

data "cloudflare_zone" "example2" {
  filter = {
    name = "example.com"
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "minimal state",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "abc123",
    "name": "example.com"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "name": "example.com"
  }
}`,
		},
		{
			Name: "full state with all v4 fields",
			Input: `{
  "schema_version": 1,
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
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
