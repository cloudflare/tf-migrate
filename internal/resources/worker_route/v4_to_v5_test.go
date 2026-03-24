package worker_route

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5ConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

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
		{
			Name: "Singular resource name generates moved block",
			Input: `resource "cloudflare_worker_route" "example" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/*"
  script_name = "my-worker"
}`,
			Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
  script  = "my-worker"
}

moved {
  from = cloudflare_worker_route.example
  to   = cloudflare_workers_route.example
}`,
		},
		{
			Name: "Script reference rewrites workers_script name to id",
			Input: `resource "cloudflare_worker_route" "example" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/*"
  script_name = cloudflare_workers_script.my_worker.name
}`,
			Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
  script  = cloudflare_workers_script.my_worker.id
}
  
moved {
  from = cloudflare_worker_route.example
  to   = cloudflare_workers_route.example
}`,
		},
		{
			Name: "Indexed script reference rewrites workers_script name to id",
			Input: `resource "cloudflare_worker_route" "example" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "${each.key}.example.com/*"
  script_name = cloudflare_workers_script.my_worker[each.key].name
}`,
			Expected: `resource "cloudflare_workers_route" "example" {
  zone_id  = "d41d8cd98f00b204e9800998ecf8427e"
  pattern  = "${each.key}.example.com/*"
  script   = cloudflare_workers_script.my_worker[each.key].id
}
  
moved {
  from = cloudflare_worker_route.example
  to   = cloudflare_workers_route.example
}`,
		},
		{
			Name: "Singular worker_script reference rewrites name to id",
			Input: `resource "cloudflare_workers_route" "example" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/*"
  script_name = cloudflare_worker_script.my_worker.name
}`,
			Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
  script  = cloudflare_worker_script.my_worker.id
}`,
		},
		{
			Name: "Singular worker_route with singular worker_script reference",
			Input: `resource "cloudflare_worker_route" "datadog_rum_route" {
  zone_id     = local.prod_coalition_zone_id
  pattern     = "${local.datadog_rum_host}/*"
  script_name = cloudflare_worker_script.datadog_rum_proxy.name
}`,
			Expected: `resource "cloudflare_workers_route" "datadog_rum_route" {
  zone_id = local.prod_coalition_zone_id
  pattern = "${local.datadog_rum_host}/*"
  script  = cloudflare_worker_script.datadog_rum_proxy.id
}

moved {
  from = cloudflare_worker_route.datadog_rum_route
  to   = cloudflare_workers_route.datadog_rum_route
}`,
		},
		{
			Name: "Legacy dot-index script reference rewrites workers_script name to id",
			Input: `resource "cloudflare_worker_route" "example" {
  zone_id     = "d41d8cd98f00b204e9800998ecf8427e"
  pattern     = "example.com/*"
  script_name = cloudflare_workers_script.my_worker.0.name
}`,
			Expected: `resource "cloudflare_workers_route" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  pattern = "example.com/*"
  script  = cloudflare_workers_script.my_worker.0.id
}

moved {
  from = cloudflare_worker_route.example
  to   = cloudflare_workers_route.example
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
