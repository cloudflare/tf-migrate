package logpull_retention

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Basic resource with enabled true",
			Input: `resource "cloudflare_logpull_retention" "example" {
  zone_id = "zone123"
  enabled = true
}`,
			Expected: `resource "cloudflare_logpull_retention" "example" {
  zone_id = "zone123"
  flag    = true
}`,
		},
		{
			Name: "Basic resource with enabled false",
			Input: `resource "cloudflare_logpull_retention" "example" {
  zone_id = "abc123def456"
  enabled = false
}`,
			Expected: `resource "cloudflare_logpull_retention" "example" {
  zone_id = "abc123def456"
  flag    = false
}`,
		},
		{
			Name: "Multiple resources",
			Input: `resource "cloudflare_logpull_retention" "zone1" {
  zone_id = "zone111"
  enabled = true
}

resource "cloudflare_logpull_retention" "zone2" {
  zone_id = "zone222"
  enabled = false
}`,
			Expected: `resource "cloudflare_logpull_retention" "zone1" {
  zone_id = "zone111"
  flag    = true
}

resource "cloudflare_logpull_retention" "zone2" {
  zone_id = "zone222"
  flag    = false
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestV4ToV5StateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "Basic state with enabled true",
			Input: `{
  "type": "cloudflare_logpull_retention",
  "name": "example",
  "attributes": {
    "zone_id": "zone123",
    "enabled": true
  }
}`,
			Expected: `{
  "type": "cloudflare_logpull_retention",
  "name": "example",
  "attributes": {
    "zone_id": "zone123",
    "flag": true
  }
}`,
		},
		{
			Name: "Basic state with enabled false",
			Input: `{
  "type": "cloudflare_logpull_retention",
  "name": "example",
  "attributes": {
    "zone_id": "abc123def456",
    "enabled": false
  }
}`,
			Expected: `{
  "type": "cloudflare_logpull_retention",
  "name": "example",
  "attributes": {
    "zone_id": "abc123def456",
    "flag": false
  }
}`,
		},
		{
			Name: "State with additional computed fields",
			Input: `{
  "type": "cloudflare_logpull_retention",
  "name": "example",
  "attributes": {
    "zone_id": "zone999",
    "enabled": true,
    "id": "some-checksum-value"
  }
}`,
			Expected: `{
  "type": "cloudflare_logpull_retention",
  "name": "example",
  "attributes": {
    "zone_id": "zone999",
    "flag": true,
    "id": "some-checksum-value"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
