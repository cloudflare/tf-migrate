package worker_route

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic route with script_name",
				Input: `resource "cloudflare_workers_route" "example" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/*"
  script_name = "my-worker"
}`,
				Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
  script  = "my-worker"
}`,
			},
			{
				Name: "Route without script_name (optional field)",
				Input: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
}`,
				Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
}`,
			},
			{
				Name: "Multiple routes in one file",
				Input: `resource "cloudflare_workers_route" "route1" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/api/*"
  script_name = "api-worker"
}

resource "cloudflare_workers_route" "route2" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/admin/*"
  script_name = "admin-worker"
}`,
				Expected: `resource "cloudflare_workers_route" "route1" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/api/*"
  script  = "api-worker"
}

resource "cloudflare_workers_route" "route2" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/admin/*"
  script  = "admin-worker"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic state with script_name",
				Input: `{
  "type": "cloudflare_workers_route",
  "name": "example",
  "attributes": {
    "id": "route123",
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "pattern": "example.com/*",
    "script_name": "my-worker"
  }
}`,
				Expected: `{
  "type": "cloudflare_workers_route",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "id": "route123",
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "pattern": "example.com/*",
    "script": "my-worker"
  }
}`,
			},
			{
				Name: "State without script_name (optional field)",
				Input: `{
  "type": "cloudflare_workers_route",
  "name": "example",
  "attributes": {
    "id": "route123",
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "pattern": "example.com/*"
  }
}`,
				Expected: `{
  "type": "cloudflare_workers_route",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "id": "route123",
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "pattern": "example.com/*"
  }
}`,
			},
			{
				Name: "State with empty attributes",
				Input: `{
  "type": "cloudflare_workers_route",
  "name": "example",
  "attributes": {}
}`,
				Expected: `{
  "type": "cloudflare_workers_route",
  "name": "example",
  "schema_version": 0,
  "attributes": {}
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
