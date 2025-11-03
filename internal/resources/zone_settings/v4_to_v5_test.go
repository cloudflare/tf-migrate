package zone_settings

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "simple attributes",
				Input: `resource "cloudflare_zone_settings_override" "zone_settings" {
  zone_id = var.zone_id

  settings {
    automatic_https_rewrites = var.automatic_https_rewrites
    ssl                      = var.ssl
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "zone_settings_automatic_https_rewrites" {
  zone_id    = var.zone_id
  setting_id = "automatic_https_rewrites"
  value      = var.automatic_https_rewrites
}
resource "cloudflare_zone_setting" "zone_settings_ssl" {
  zone_id    = var.zone_id
  setting_id = "ssl"
  value      = var.ssl
}`,
			},
			{
				Name: "with security header block",
				Input: `resource "cloudflare_zone_settings_override" "zone_settings" {
  zone_id = var.zone_id

  settings {
    ssl = var.ssl

    security_header {
      enabled = var.security_header_enabled
      max_age = var.security_header_max_age
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "zone_settings_ssl" {
  zone_id    = var.zone_id
  setting_id = "ssl"
  value      = var.ssl
}
resource "cloudflare_zone_setting" "zone_settings_security_header" {
  zone_id    = var.zone_id
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled  =  var.security_header_enabled
      max_age  =  var.security_header_max_age
    }
  }
}`,
			},
			{
				Name: "with nel block",
				Input: `resource "cloudflare_zone_settings_override" "zone_settings" {
  zone_id = var.zone_id

  settings {
    nel {
      enabled = var.enable_network_error_logging
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "zone_settings_nel" {
  zone_id    = var.zone_id
  setting_id = "nel"
  value = {
    enabled  =  var.enable_network_error_logging
  }
}`,
			},
			{
				Name: "zero_rtt mapping",
				Input: `resource "cloudflare_zone_settings_override" "zone_settings" {
  zone_id = "abc123"

  settings {
    zero_rtt = "on"
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "zone_settings_zero_rtt" {
  zone_id    = "abc123"
  setting_id = "0rtt"
  value      = "on"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
