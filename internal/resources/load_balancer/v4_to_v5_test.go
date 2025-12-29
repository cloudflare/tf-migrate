package load_balancer

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic load balancer",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]
  fallback_pool_id = "example-fallback-pool-id"
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "example-zone-id"
  name          = "example-lb.example.com"
  default_pools = ["example-pool-id"]
  fallback_pool = "example-fallback-pool-id"
}`,
			},
			{
				Name: "Load balancer with TTL",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]
  fallback_pool_id = "example-fallback-pool-id"
  ttl              = 30
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "example-zone-id"
  name          = "example-lb.example.com"
  ttl           = 30
  default_pools = ["example-pool-id"]
  fallback_pool = "example-fallback-pool-id"
}`,
			},
			{
				Name: "Load balancer with steering policy",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id           = "example-zone-id"
  name              = "example-lb.example.com"
  default_pool_ids  = ["example-pool-id"]
  fallback_pool_id  = "example-fallback-pool-id"
  steering_policy   = "geo"
  session_affinity  = "cookie"
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  steering_policy  = "geo"
  session_affinity = "cookie"
  default_pools    = ["example-pool-id"]
  fallback_pool    = "example-fallback-pool-id"
}`,
			},
			{
				Name: "Load balancer with adaptive_routing block",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  adaptive_routing {
    failover_across_pools = false
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  adaptive_routing = {
    failover_across_pools = false
  }
}`,
			},
			{
				Name: "Load balancer with location_strategy block",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  location_strategy {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
}`,
			},
			{
				Name: "Load balancer with random_steering block",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  random_steering {
    default_weight = 0.5
    pool_weights = {
      pool1 = 0.3
      pool2 = 0.7
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  random_steering = {
    default_weight = 0.5
    pool_weights = {
      pool1 = 0.3
      pool2 = 0.7
    }
  }
}`,
			},
			{
				Name: "Load balancer with all optional single-object blocks",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  adaptive_routing {
    failover_across_pools = false
  }

  location_strategy {
    prefer_ecs = "proximity"
    mode       = "pop"
  }

  random_steering {
    default_weight = 0.5
  }

  session_affinity_attributes {
    samesite = "Lax"
    secure   = "Always"
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  session_affinity_attributes = {
    samesite = "Lax"
    secure   = "Always"
  }
  adaptive_routing = {
    failover_across_pools = false
  }
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
  random_steering = {
    default_weight = 0.5
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic load balancer state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pool_ids": ["pool-123"],
    "fallback_pool_id": "pool-456"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pools": ["pool-123"],
    "fallback_pool": "pool-456"
  }
}`,
			},
			{
				Name: "Load balancer with adaptive_routing array to object",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pool_ids": ["pool-123"],
    "adaptive_routing": [{"failover_across_pools": false}]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pools": ["pool-123"],
    "adaptive_routing": {"failover_across_pools": false}
  }
}`,
			},
			{
				Name: "Load balancer with location_strategy array to object",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pool_ids": ["pool-123"],
    "location_strategy": [{"prefer_ecs": "proximity", "mode": "pop"}]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pools": ["pool-123"],
    "location_strategy": {"prefer_ecs": "proximity", "mode": "pop"}
  }
}`,
			},
			{
				Name: "Load balancer with random_steering array to object",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pool_ids": ["pool-123"],
    "random_steering": [{"default_weight": 0.5}]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pools": ["pool-123"],
    "random_steering": {"default_weight": 0.5}
  }
}`,
			},
			{
				Name: "Load balancer with empty arrays converted to null",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pool_ids": ["pool-123"],
    "adaptive_routing": [],
    "location_strategy": [],
    "random_steering": [],
    "region_pools": [],
    "pop_pools": [],
    "country_pools": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "lb-123",
    "zone_id": "zone-123",
    "name": "example-lb.example.com",
    "default_pools": ["pool-123"],
    "adaptive_routing": null,
    "location_strategy": null,
    "random_steering": null,
    "region_pools": null,
    "pop_pools": null,
    "country_pools": null
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
