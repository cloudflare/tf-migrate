package regional_hostname

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
				Name: "removes timeouts block",
				Input: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

  timeouts {
    create = "30s"
    update = "30s"
  }
}`,
				Expected: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

}`,
			},
			{
				Name: "removes timeouts with other blocks",
				Input: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

  timeouts {
    create = "30s"
    update = "30s"
    delete = "30s"
  }
}`,
				Expected: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

}`,
			},
			{
				Name: "no change when no timeouts",
				Input: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"
}`,
				Expected: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"
}`,
			},
			{
				Name: "preserves other attributes",
				Input: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = var.zone_id
  hostname   = "regional.example.com"
  region_key = "eu"

  timeouts {
    create = "1m"
  }
}`,
				Expected: `resource "cloudflare_regional_hostname" "test" {
  zone_id    = var.zone_id
  hostname   = "regional.example.com"
  region_key = "eu"

}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
