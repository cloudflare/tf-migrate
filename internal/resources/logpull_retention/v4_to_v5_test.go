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
