package regional_tiered_cache

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic resource with value on",
				Input: `
resource "cloudflare_regional_tiered_cache" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}`,
				Expected: `
resource "cloudflare_regional_tiered_cache" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}`,
			},
			{
				Name: "basic resource with value off",
				Input: `
resource "cloudflare_regional_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "off"
}`,
				Expected: `
resource "cloudflare_regional_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "off"
}`,
			},
			{
				Name: "multiple resources",
				Input: `
resource "cloudflare_regional_tiered_cache" "first" {
  zone_id = "zone1"
  value   = "on"
}

resource "cloudflare_regional_tiered_cache" "second" {
  zone_id = "zone2"
  value   = "off"
}`,
				Expected: `
resource "cloudflare_regional_tiered_cache" "first" {
  zone_id = "zone1"
  value   = "on"
}

resource "cloudflare_regional_tiered_cache" "second" {
  zone_id = "zone2"
  value   = "off"
}`,
			},
			{
				Name: "with variable reference",
				Input: `
resource "cloudflare_regional_tiered_cache" "example" {
  zone_id = var.cloudflare_zone_id
  value   = var.cache_value
}`,
				Expected: `
resource "cloudflare_regional_tiered_cache" "example" {
  zone_id = var.cloudflare_zone_id
  value   = var.cache_value
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
