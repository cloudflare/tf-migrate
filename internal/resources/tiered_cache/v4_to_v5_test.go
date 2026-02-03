package tiered_cache

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

// Config transformation tests
func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "tiered_cache with cache_type=smart",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "smart"
}`,
			Expected: `
resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "on"
}
resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = "test-zone-id"
  value   = "on"
} `,
		},
		{
			Name: "tiered_cache with cache_type=generic",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "generic"
}`,
			Expected: `
resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "off"
}
resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = "test-zone-id"
  value   = "on"
} `,
		},
		{
			Name: "tiered_cache with cache_type=off",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "off"
}`,
			Expected: `
resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "off"
}
resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = "test-zone-id"
  value   = "off"
}`,
		},
		{
			Name: "multiple tiered_cache resources with mixed types",
			Input: `
resource "cloudflare_tiered_cache" "off" {
  zone_id    = "zone1"
  cache_type = "off"
}
resource "cloudflare_tiered_cache" "smart" {
  zone_id    = "zone2"
  cache_type = "smart"
}
resource "cloudflare_tiered_cache" "generic" {
  zone_id    = "zone3"
  cache_type = "generic"
}`,
			Expected: `
resource "cloudflare_tiered_cache" "off" {
  zone_id = "zone1"
  value   = "off"
}
resource "cloudflare_argo_tiered_caching" "off" {
  zone_id = "zone1"
  value   = "off"
}
resource "cloudflare_tiered_cache" "smart" {
  zone_id = "zone2"
  value   = "on"
}
resource "cloudflare_argo_tiered_caching" "smart" {
  zone_id = "zone2"
  value   = "on"
}
resource "cloudflare_tiered_cache" "generic" {
  zone_id = "zone3"
  value   = "off"
}
resource "cloudflare_argo_tiered_caching" "generic" {
  zone_id = "zone3"
  value   = "on"
}
`,
		},
		{
			Name: "tiered_cache with cache_type=generic and other attributes",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = cloudflare_zone.example.id
  cache_type = "generic"

  lifecycle {
    create_before_destroy = true
  }
}`,
			Expected: `
resource "cloudflare_tiered_cache" "example" {
  zone_id = cloudflare_zone.example.id
  value   = "off"
  lifecycle {
    create_before_destroy = true
  }
}
resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = cloudflare_zone.example.id
  value   = "on"
  lifecycle {
    create_before_destroy = true
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// TestStateTransformation_Removed documents that state transformation tests were removed.
// State transformation is now handled by the provider's StateUpgraders (UpgradeState).
// The provider transforms cache_type to value when it sees schema_version=0.
func TestStateTransformation_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}
