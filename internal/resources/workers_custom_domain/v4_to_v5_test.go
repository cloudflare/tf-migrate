package workers_custom_domain

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5ConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "rename resource type with attributes passthrough",
			Input: `resource "cloudflare_worker_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  zone_id    = "023e105f4ecef8ad9ca31a8372d0c353"
  hostname   = "api.example.com"
  service    = "my-worker"
  environment = "production"
}`,
			Expected: `resource "cloudflare_workers_custom_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  zone_id    = "023e105f4ecef8ad9ca31a8372d0c353"
  hostname   = "api.example.com"
  service    = "my-worker"
  environment = "production"
}

moved {
  from = cloudflare_worker_domain.example
  to   = cloudflare_workers_custom_domain.example
}`,
		},
		{
			Name: "preserve meta arguments",
			Input: `resource "cloudflare_worker_domain" "domains" {
  for_each = toset(["api.example.com", "admin.example.com"])

  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = each.value
  service    = "my-worker"
}`,
			Expected: `resource "cloudflare_workers_custom_domain" "domains" {
  for_each = toset(["api.example.com", "admin.example.com"])

  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = each.value
  service    = "my-worker"
}

moved {
  from = cloudflare_worker_domain.domains
  to   = cloudflare_workers_custom_domain.domains
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestUsesProviderStateUpgrader(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	if got := migrator.(*V4ToV5Migrator).UsesProviderStateUpgrader(); !got {
		t.Fatalf("UsesProviderStateUpgrader() = %v, want true", got)
	}
}
