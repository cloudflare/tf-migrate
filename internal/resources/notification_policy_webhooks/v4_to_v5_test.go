package notification_policy_webhooks

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "minimal config with url",
				Input: `
resource "cloudflare_notification_policy_webhooks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-webhook"
  url        = "https://example.com/webhook"
}`,
				Expected: `
resource "cloudflare_notification_policy_webhooks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-webhook"
  url        = "https://example.com/webhook"
}`,
			},
			{
				Name: "full config with all fields",
				Input: `
resource "cloudflare_notification_policy_webhooks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "production-webhook"
  url        = "https://alerts.example.com/notify"
  secret     = "webhook-secret-token"
}`,
				Expected: `
resource "cloudflare_notification_policy_webhooks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "production-webhook"
  url        = "https://alerts.example.com/notify"
  secret     = "webhook-secret-token"
}`,
			},
			{
				Name: "multiple resources",
				Input: `
resource "cloudflare_notification_policy_webhooks" "primary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "primary-webhook"
  url        = "https://primary.example.com/webhook"
}

resource "cloudflare_notification_policy_webhooks" "backup" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "backup-webhook"
  url        = "https://backup.example.com/webhook"
  secret     = "backup-secret"
}`,
				Expected: `
resource "cloudflare_notification_policy_webhooks" "primary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "primary-webhook"
  url        = "https://primary.example.com/webhook"
}

resource "cloudflare_notification_policy_webhooks" "backup" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "backup-webhook"
  url        = "https://backup.example.com/webhook"
  secret     = "backup-secret"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Note: No error cases for config transformation since we assume all v4 configs
	// have the url field. If missing, the v5 provider will validate during apply.
}
