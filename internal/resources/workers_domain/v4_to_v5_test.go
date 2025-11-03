package workers_domain

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
				Name: "rename cloudflare_worker_domain to cloudflare_workers_custom_domain",
				Input: `resource "cloudflare_worker_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "subdomain.example.com"
  service    = "my-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
				Expected: `resource "cloudflare_workers_custom_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "subdomain.example.com"
  service    = "my-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			},
			{
				Name: "already renamed resource should not change",
				Input: `resource "cloudflare_workers_custom_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "app.example.com"
  service    = "worker-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
				Expected: `resource "cloudflare_workers_custom_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "app.example.com"
  service    = "worker-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			},
			{
				Name: "multiple worker domain resources",
				Input: `resource "cloudflare_worker_domain" "primary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "primary.example.com"
  service    = "primary-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}

resource "cloudflare_worker_domain" "secondary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "secondary.example.com"
  service    = "secondary-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
				Expected: `resource "cloudflare_workers_custom_domain" "primary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "primary.example.com"
  service    = "primary-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}

resource "cloudflare_workers_custom_domain" "secondary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "secondary.example.com"
  service    = "secondary-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			},
			{
				Name: "worker domain with environment attribute",
				Input: `resource "cloudflare_worker_domain" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  hostname    = "subdomain.example.com"
  service     = "my-service"
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  environment = "production"
}`,
				Expected: `resource "cloudflare_workers_custom_domain" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  hostname    = "subdomain.example.com"
  service     = "my-service"
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  environment = "production"
}`,
			},
			{
				Name: "mixed old and new resource types",
				Input: `resource "cloudflare_worker_domain" "old_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "old.example.com"
  service    = "old-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}

resource "cloudflare_workers_custom_domain" "new_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "new.example.com"
  service    = "new-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
				Expected: `resource "cloudflare_workers_custom_domain" "old_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "old.example.com"
  service    = "old-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}

resource "cloudflare_workers_custom_domain" "new_style" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "new.example.com"
  service    = "new-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
