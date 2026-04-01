package custom_ssl

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

var migrator = NewV4ToV5Migrator()

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "full config with all custom_ssl_options fields including geo_restrictions",
				Input: `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate      = "-----BEGIN CERTIFICATE-----"
    private_key      = "-----BEGIN RSA PRIVATE KEY-----"
    bundle_method    = "ubiquitous"
    type             = "legacy_custom"
    geo_restrictions = "us"
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "example" {
  zone_id          = "abc123"
  certificate      = "-----BEGIN CERTIFICATE-----"
  private_key      = "-----BEGIN RSA PRIVATE KEY-----"
  bundle_method    = "ubiquitous"
  type             = "legacy_custom"
  geo_restrictions = { label = "us" }
}`,
			},
			{
				Name: "minimal config with only certificate and private_key",
				Input: `
resource "cloudflare_custom_ssl" "minimal" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate = "-----BEGIN CERTIFICATE-----"
    private_key = "-----BEGIN RSA PRIVATE KEY-----"
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "minimal" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----"
  private_key = "-----BEGIN RSA PRIVATE KEY-----"
}`,
			},
			{
				Name: "config without geo_restrictions",
				Input: `
resource "cloudflare_custom_ssl" "no_geo" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate   = "-----BEGIN CERTIFICATE-----"
    private_key   = "-----BEGIN RSA PRIVATE KEY-----"
    bundle_method = "ubiquitous"
    type          = "legacy_custom"
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "no_geo" {
  zone_id       = "abc123"
  certificate   = "-----BEGIN CERTIFICATE-----"
  private_key   = "-----BEGIN RSA PRIVATE KEY-----"
  bundle_method = "ubiquitous"
  type          = "legacy_custom"
}`,
			},
			{
				Name: "config with custom_ssl_priority block must be removed",
				Input: `
resource "cloudflare_custom_ssl" "with_priority" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate = "-----BEGIN CERTIFICATE-----"
    private_key = "-----BEGIN RSA PRIVATE KEY-----"
  }

  custom_ssl_priority {
    id       = "abc123"
    priority = 1
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "with_priority" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----"
  private_key = "-----BEGIN RSA PRIVATE KEY-----"
}`,
			},
			{
				Name: "config with variable references preserved",
				Input: `
resource "cloudflare_custom_ssl" "vars" {
  zone_id = var.zone_id

  custom_ssl_options {
    certificate   = var.certificate
    private_key   = var.private_key
    bundle_method = var.bundle_method
    type          = "legacy_custom"
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "vars" {
  zone_id       = var.zone_id
  certificate   = var.certificate
  private_key   = var.private_key
  bundle_method = var.bundle_method
  type          = "legacy_custom"
}`,
			},
			{
				Name: "multiple resources in one file",
				Input: `
resource "cloudflare_custom_ssl" "first" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate = "-----BEGIN CERTIFICATE-----"
    private_key = "-----BEGIN RSA PRIVATE KEY-----"
    type        = "legacy_custom"
  }
}

resource "cloudflare_custom_ssl" "second" {
  zone_id = "def456"

  custom_ssl_options {
    certificate      = "-----BEGIN CERTIFICATE-----"
    private_key      = "-----BEGIN RSA PRIVATE KEY-----"
    geo_restrictions = "eu"
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "first" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----"
  private_key = "-----BEGIN RSA PRIVATE KEY-----"
  type        = "legacy_custom"
}

resource "cloudflare_custom_ssl" "second" {
  zone_id          = "def456"
  certificate      = "-----BEGIN CERTIFICATE-----"
  private_key      = "-----BEGIN RSA PRIVATE KEY-----"
  geo_restrictions = { label = "eu" }
}`,
			},
			{
				Name: "multiple custom_ssl_priority blocks all removed",
				Input: `
resource "cloudflare_custom_ssl" "multi_priority" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate = "-----BEGIN CERTIFICATE-----"
    private_key = "-----BEGIN RSA PRIVATE KEY-----"
  }

  custom_ssl_priority {
    id       = "abc123"
    priority = 1
  }

  custom_ssl_priority {
    id       = "def456"
    priority = 2
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "multi_priority" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----"
  private_key = "-----BEGIN RSA PRIVATE KEY-----"
}`,
			},
			{
				Name: "geo_restrictions with hyphenated value preserved",
				Input: `
resource "cloudflare_custom_ssl" "hyphen_geo" {
  zone_id = "abc123"

  custom_ssl_options {
    certificate      = "-----BEGIN CERTIFICATE-----"
    private_key      = "-----BEGIN RSA PRIVATE KEY-----"
    geo_restrictions = "us-t5"
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "hyphen_geo" {
  zone_id          = "abc123"
  certificate      = "-----BEGIN CERTIFICATE-----"
  private_key      = "-----BEGIN RSA PRIVATE KEY-----"
  geo_restrictions = { label = "us-t5" }
}`,
			},

			// --- Placeholder path: missing write-only fields ---
			{
				Name: "missing certificate and private_key — placeholders and lifecycle added",
				Input: `
resource "cloudflare_custom_ssl" "no_creds" {
  zone_id = "abc123"
}`,
				Expected: `resource "cloudflare_custom_ssl" "no_creds" {
  zone_id     = "abc123"
  certificate = "PLACEHOLDER - actual certificate already deployed"
  private_key = "PLACEHOLDER - actual private key already deployed"
  lifecycle {
    ignore_changes = [certificate, private_key]
  }
}`,
			},
			{
				Name: "empty custom_ssl_options block — placeholders and lifecycle added",
				Input: `
resource "cloudflare_custom_ssl" "empty_opts" {
  zone_id = "abc123"
  custom_ssl_options {}
}`,
				Expected: `resource "cloudflare_custom_ssl" "empty_opts" {
  zone_id     = "abc123"
  certificate = "PLACEHOLDER - actual certificate already deployed"
  private_key = "PLACEHOLDER - actual private key already deployed"
  lifecycle {
    ignore_changes = [certificate, private_key]
  }
}`,
			},
			{
				Name: "existing lifecycle block — ignore_changes merged, other attrs preserved",
				Input: `
resource "cloudflare_custom_ssl" "existing_lc" {
  zone_id = "abc123"
  lifecycle {
    create_before_destroy = true
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "existing_lc" {
  zone_id     = "abc123"
  certificate = "PLACEHOLDER - actual certificate already deployed"
  private_key = "PLACEHOLDER - actual private key already deployed"
  lifecycle {
    create_before_destroy = true
    ignore_changes        = [certificate, private_key]
  }
}`,
			},
			{
				Name: "existing ignore_changes custom_ssl_options is normalized",
				Input: `
resource "cloudflare_custom_ssl" "existing_ignore" {
  zone_id = "abc123"
  lifecycle {
    ignore_changes = [custom_ssl_options]
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "existing_ignore" {
  zone_id     = "abc123"
  certificate = "PLACEHOLDER - actual certificate already deployed"
  private_key = "PLACEHOLDER - actual private key already deployed"
  lifecycle {
    ignore_changes = [certificate, private_key]
  }
}`,
			},
			{
				Name: "existing ignore_changes all stays exclusive",
				Input: `
resource "cloudflare_custom_ssl" "ignore_all" {
  zone_id = "abc123"
  lifecycle {
    ignore_changes = all
  }
}`,
				Expected: `resource "cloudflare_custom_ssl" "ignore_all" {
  zone_id     = "abc123"
  certificate = "PLACEHOLDER - actual certificate already deployed"
  private_key = "PLACEHOLDER - actual private key already deployed"
  lifecycle {
    ignore_changes = all
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
