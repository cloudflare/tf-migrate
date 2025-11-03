package tiered_cache

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
				Name: "cache_type smart to value on",
				Input: `resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "smart"
}`,
				Expected: `resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "on"
}`,
			},
			{
				Name: "cache_type generic creates argo_tiered_caching",
				Input: `resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "generic"
}`,
				Expected: `resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = "test-zone-id"
  value   = "on"
}
moved {
  from = cloudflare_tiered_cache.example
  to   = cloudflare_argo_tiered_caching.example
}`,
			},
			{
				Name: "cache_type off to value off",
				Input: `resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "off"
}`,
				Expected: `resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = "off"
}`,
			},
			{
				Name: "cache_type variable",
				Input: `resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = var.cache_type_value
}`,
				Expected: `resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = var.cache_type_value
}`,
			},
			{
				Name: "generic with lifecycle block",
				Input: `resource "cloudflare_tiered_cache" "example" {
  zone_id    = cloudflare_zone.example.id
  cache_type = "generic"

  lifecycle {
    create_before_destroy = true
  }
}`,
				Expected: `resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = cloudflare_zone.example.id
  value   = "on"
  lifecycle {
    create_before_destroy = true
  }
}
moved {
  from = cloudflare_tiered_cache.example
  to   = cloudflare_argo_tiered_caching.example
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "generic to argo_tiered_caching in state",
				Input: `{
  "type": "cloudflare_tiered_cache",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone-id",
        "cache_type": "generic",
        "id": "test-id"
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_tiered_cache",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone-id",
        "value": "on",
        "id": "test-id"
      }
    }
  ]
}`,
			},
			{
				Name: "smart to on in state",
				Input: `{
  "type": "cloudflare_tiered_cache",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone-id",
        "cache_type": "smart",
        "id": "test-id"
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_tiered_cache",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone-id",
        "value": "on",
        "id": "test-id"
      }
    }
  ]
}`,
			},
			{
				Name: "off to off in state",
				Input: `{
  "type": "cloudflare_tiered_cache",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone-id",
        "cache_type": "off",
        "id": "test-id"
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_tiered_cache",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone-id",
        "value": "off",
        "id": "test-id"
      }
    }
  ]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
