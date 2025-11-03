package workers_route

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test config transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "workers_route script_name to script",
				Input: `resource "cloudflare_workers_route" "example" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "example.com/*"
  script_name = "my-worker"
}`,
				Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "example.com/*"
  script  = "my-worker"
}`,
			},
			{
				Name: "workers_route with no script_name",
				Input: `resource "cloudflare_workers_route" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "example.com/*"
}`,
				Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "example.com/*"
}`,
			},
			{
				Name: "worker_route (singular) with script_name",
				Input: `resource "cloudflare_worker_route" "example" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "example.com/*"
  script_name = "my-worker"
}`,
				Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "example.com/*"
  script  = "my-worker"
}`,
			},
			{
				Name: "multiple workers routes with both singular and plural",
				Input: `resource "cloudflare_worker_route" "route1" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "api.example.com/*"
  script_name = "api-worker"
}

resource "cloudflare_workers_route" "route2" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "www.example.com/*"
  script_name = "web-worker"
}`,
				Expected: `resource "cloudflare_workers_route" "route1" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "api.example.com/*"
  script  = "api-worker"
}

resource "cloudflare_workers_route" "route2" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "www.example.com/*"
  script  = "web-worker"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "transforms script_name to script in state",
				Input: `{
  "type": "cloudflare_worker_route",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone",
        "pattern": "test.com/*",
        "script_name": "my-worker"
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_workers_route",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone",
        "pattern": "test.com/*",
        "script": "my-worker"
      }
    }
  ]
}`,
			},
			{
				Name: "handles missing script_name in state",
				Input: `{
  "type": "cloudflare_worker_route",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone",
        "pattern": "test.com/*"
      }
    }
  ]
}`,
				Expected: `{
  "type": "cloudflare_workers_route",
  "name": "test",
  "instances": [
    {
      "attributes": {
        "zone_id": "test-zone",
        "pattern": "test.com/*"
      }
    }
  ]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
