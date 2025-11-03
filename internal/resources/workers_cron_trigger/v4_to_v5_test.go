package workers_cron_trigger

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
				Name: "worker_cron_trigger singular resource rename",
				Input: `resource "cloudflare_worker_cron_trigger" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "my-worker"
  cron        = "0 0 * * *"
}`,
				Expected: `resource "cloudflare_workers_cron_trigger" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "my-worker"
  cron        = "0 0 * * *"
}`,
			},
			{
				Name: "workers_cron_trigger plural stays same",
				Input: `resource "cloudflare_workers_cron_trigger" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "my-worker"
  cron        = "*/5 * * * *"
}`,
				Expected: `resource "cloudflare_workers_cron_trigger" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "my-worker"
  cron        = "*/5 * * * *"
}`,
			},
			{
				Name: "multiple cron triggers with different patterns",
				Input: `resource "cloudflare_worker_cron_trigger" "daily" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "daily-worker"
  cron        = "0 0 * * *"
}

resource "cloudflare_worker_cron_trigger" "hourly" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "hourly-worker"
  cron        = "0 * * * *"
}`,
				Expected: `resource "cloudflare_workers_cron_trigger" "daily" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "daily-worker"
  cron        = "0 0 * * *"
}

resource "cloudflare_workers_cron_trigger" "hourly" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "hourly-worker"
  cron        = "0 * * * *"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
