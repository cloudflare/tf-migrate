package custom_pages

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "account-level custom page with all fields",
				Input: `
resource "cloudflare_custom_pages" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "500_errors"
  state      = "customized"
  url        = "https://example.workers.dev/"
}`,
				Expected: `
resource "cloudflare_custom_pages" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  state      = "customized"
  url        = "https://example.workers.dev/"
  identifier = "500_errors"
}`,
			},
			{
				Name: "zone-level custom page",
				Input: `
resource "cloudflare_custom_pages" "zone_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  type    = "basic_challenge"
  state   = "default"
  url     = ""
}`,
				Expected: `
resource "cloudflare_custom_pages" "zone_test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  state      = "default"
  url        = ""
  identifier = "basic_challenge"
}`,
			},
			{
				Name: "multiple page types",
				Input: `
resource "cloudflare_custom_pages" "error_500" {
  account_id = "abc123"
  type       = "500_errors"
  state      = "customized"
  url        = "https://errors.example.com/500.html"
}

resource "cloudflare_custom_pages" "error_1000" {
  account_id = "abc123"
  type       = "1000_errors"
  state      = "customized"
  url        = "https://errors.example.com/1000.html"
}`,
				Expected: `
resource "cloudflare_custom_pages" "error_500" {
  account_id = "abc123"
  state      = "customized"
  url        = "https://errors.example.com/500.html"
  identifier = "500_errors"
}

resource "cloudflare_custom_pages" "error_1000" {
  account_id = "abc123"
  state      = "customized"
  url        = "https://errors.example.com/1000.html"
  identifier = "1000_errors"
}`,
			},
			{
				Name: "waf_block type",
				Input: `
resource "cloudflare_custom_pages" "waf" {
  account_id = "abc123"
  type       = "waf_block"
  state      = "customized"
  url        = "https://example.workers.dev/waf"
}`,
				Expected: `
resource "cloudflare_custom_pages" "waf" {
  account_id = "abc123"
  state      = "customized"
  url        = "https://example.workers.dev/waf"
  identifier = "waf_block"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "account-level custom page",
				Input: `{
					"schema_version": 1,
					"attributes": {
						"id": "500_errors",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"type": "500_errors",
						"state": "customized",
						"url": "https://example.workers.dev/"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"id": "500_errors",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"identifier": "500_errors",
						"state": "customized",
						"url": "https://example.workers.dev/"
					}
				}`,
			},
			{
				Name: "zone-level custom page",
				Input: `{
					"schema_version": 1,
					"attributes": {
						"id": "basic_challenge",
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"type": "basic_challenge",
						"state": "default",
						"url": ""
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"id": "basic_challenge",
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"identifier": "basic_challenge",
						"state": "default",
						"url": ""
					}
				}`,
			},
			{
				Name: "various page types",
				Input: `{
					"schema_version": 1,
					"attributes": {
						"id": "waf_block",
						"account_id": "abc123",
						"type": "waf_block",
						"state": "customized",
						"url": "https://example.workers.dev/waf"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"id": "waf_block",
						"account_id": "abc123",
						"identifier": "waf_block",
						"state": "customized",
						"url": "https://example.workers.dev/waf"
					}
				}`,
			},
			{
				Name: "managed_challenge type",
				Input: `{
					"schema_version": 1,
					"attributes": {
						"id": "managed_challenge",
						"account_id": "abc123",
						"type": "managed_challenge",
						"state": "customized",
						"url": "https://challenge.example.com/"
					}
				}`,
				Expected: `{
					"schema_version": 0,
					"attributes": {
						"id": "managed_challenge",
						"account_id": "abc123",
						"identifier": "managed_challenge",
						"state": "customized",
						"url": "https://challenge.example.com/"
					}
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
