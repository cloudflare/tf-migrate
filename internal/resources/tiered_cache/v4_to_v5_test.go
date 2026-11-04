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
}
import {
  to = cloudflare_argo_tiered_caching.example
  id = "<zone_id>"
}
 `,
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
}
import {
  to = cloudflare_argo_tiered_caching.example
  id = "<zone_id>"
}
 `,
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
}
import {
  to = cloudflare_argo_tiered_caching.example
  id = "<zone_id>"
}
`,
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
import {
  to = cloudflare_argo_tiered_caching.off
  id = "<zone_id>"
}

resource "cloudflare_tiered_cache" "smart" {
  zone_id = "zone2"
  value   = "on"
}
resource "cloudflare_argo_tiered_caching" "smart" {
  zone_id = "zone2"
  value   = "on"
}
import {
  to = cloudflare_argo_tiered_caching.smart
  id = "<zone_id>"
}

resource "cloudflare_tiered_cache" "generic" {
  zone_id = "zone3"
  value   = "off"
}
resource "cloudflare_argo_tiered_caching" "generic" {
  zone_id = "zone3"
  value   = "on"
}
import {
  to = cloudflare_argo_tiered_caching.generic
  id = "<zone_id>"
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
}
import {
  to = cloudflare_argo_tiered_caching.example
  id = "<zone_id>"
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
