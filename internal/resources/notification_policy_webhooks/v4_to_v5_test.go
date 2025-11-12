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

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "minimal state with url",
				Input: `{
					"schema_version": 1,
					"attributes": {
						"id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "test-webhook",
						"url": "https://example.com/webhook",
						"type": "generic"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "test-webhook",
						"url": "https://example.com/webhook",
						"type": "generic"
					}
				}`,
			},
			{
				Name: "full state with all fields",
				Input: `{
					"schema_version": 1,
					"attributes": {
						"id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "prod-webhook",
						"url": "https://alerts.example.com/notify",
						"secret": "webhook-secret",
						"type": "generic",
						"created_at": "2023-01-01T12:00:00Z",
						"last_success": "2023-01-15T14:30:00Z",
						"last_failure": null
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "prod-webhook",
						"url": "https://alerts.example.com/notify",
						"secret": "webhook-secret",
						"type": "generic",
						"created_at": "2023-01-01T12:00:00Z",
						"last_success": "2023-01-15T14:30:00Z",
						"last_failure": null
					}
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	// Note: No error cases for state transformation since we assume all v4 states
	// have the url field. While it was optional in v4 schema, the Cloudflare API
	// requires it, so all real resources should have it.
}
