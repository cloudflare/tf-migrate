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
}`,
		},
		{
			Name: "tiered_cache with cache_type=generic creates argo_tiered_caching and moved block",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = "generic"
}`,
			Expected: `
resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = "test-zone-id"
  value   = "on"
}
moved {
  from = cloudflare_tiered_cache.example
  to   = cloudflare_argo_tiered_caching.example
}`,
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
}`,
		},
		{
			Name: "tiered_cache with variable cache_type",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = var.cache_type_value
}`,
			Expected: `
resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = var.cache_type_value
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
resource "cloudflare_tiered_cache" "smart" {
  zone_id = "zone2"
  value   = "on"
}
resource "cloudflare_argo_tiered_caching" "generic" {
  zone_id = "zone3"
  value   = "on"
}
moved {
  from = cloudflare_tiered_cache.generic
  to   = cloudflare_argo_tiered_caching.generic
}`,
		},
		{
			Name: "tiered_cache with dynamic cache_type should not generate moved block",
			Input: `
resource "cloudflare_tiered_cache" "example" {
  zone_id    = "test-zone-id"
  cache_type = var.cache_type
}`,
			Expected: `
resource "cloudflare_tiered_cache" "example" {
  zone_id = "test-zone-id"
  value   = var.cache_type
}`,
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
resource "cloudflare_argo_tiered_caching" "example" {
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
}

// State transformation tests
func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "transforms_tiered_cache_generic_to_argo_tiered_caching",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "cache_type": "generic",
    "id": "test-id"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "value": "on",
    "id": "test-id"
  }
}`,
		},
		{
			Name: "transforms_tiered_cache_smart_value",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "cache_type": "smart",
    "id": "test-id"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "value": "on",
    "id": "test-id"
  }
}`,
		},
		{
			Name: "transforms_tiered_cache_off_value",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "cache_type": "off",
    "id": "test-id"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "value": "off",
    "id": "test-id"
  }
}`,
		},
		{
			Name: "handles_tiered_cache_without_cache_type",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "id": "test-id"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "id": "test-id"
  }
}`,
		},
		{
			Name: "preserves_non_tiered_cache_resources",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "name": "example.com",
    "id": "test-id"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "name": "example.com",
    "id": "test-id"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
