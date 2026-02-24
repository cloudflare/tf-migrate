package zone_dnssec

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic zone_dnssec with minimal fields",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
			},
			{
				Name: "Multiple zone_dnssec resources in one file",
				Input: `
resource "cloudflare_zone_dnssec" "example1" {
  zone_id = "abc123"
}

resource "cloudflare_zone_dnssec" "example2" {
  zone_id = "def456"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example1" {
  zone_id = "abc123"
}

resource "cloudflare_zone_dnssec" "example2" {
  zone_id = "def456"
}`,
			},
			{
				Name: "Zone DNSSEC with modified_on field (should be removed)",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id     = "abc123"
  modified_on = "2024-01-15T10:30:00Z"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
