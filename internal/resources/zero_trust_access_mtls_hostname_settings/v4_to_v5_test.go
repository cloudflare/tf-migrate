package zero_trust_access_mtls_hostname_settings

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "single settings block",
				Input: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings {
    hostname = "example.com"
    client_certificate_forwarding = true
    china_network = false
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = [{
    china_network                 = false
    client_certificate_forwarding = true
    hostname                      = "example.com"
  }]
}`,
			},
			{
				Name: "multiple settings blocks",
				Input: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  settings {
    hostname = "app1.example.com"
    client_certificate_forwarding = true
    china_network = false
  }
  settings {
    hostname = "app2.example.com"
    client_certificate_forwarding = false
    china_network = false
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  settings = [{
    china_network                 = false
    client_certificate_forwarding = true
    hostname                      = "app1.example.com"
  }, {
    china_network                 = false
    client_certificate_forwarding = false
    hostname                      = "app2.example.com"
  }]
}`,
			},
			{
				Name: "with boolean defaults",
				Input: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings {
    hostname = "example.com"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = [{
    china_network                 = false
    client_certificate_forwarding = false
    hostname                      = "example.com"
  }]
}`,
			},
			{
				Name: "dynamic settings blocks converted to for expression",
				Input: `locals {
  mtls_domains = ["app1.example.com", "app2.example.com"]
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  dynamic "settings" {
    for_each = local.mtls_domains
    content {
      hostname = settings.value
      client_certificate_forwarding = true
      china_network = false
    }
  }
}`,
				Expected: `locals {
  mtls_domains = ["app1.example.com", "app2.example.com"]
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = [for value in local.mtls_domains : {
    hostname                      = value
    china_network                 = false
    client_certificate_forwarding = true
  }]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}

func TestAccessMutualTLSHostnameSettingsTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation_OldResourceName", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "transform cloudflare_access_mutual_tls_hostname_settings to v5 format",
				Input: `resource "cloudflare_access_mutual_tls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings {
    hostname = "example.com"
    client_certificate_forwarding = true
    china_network = false
  }
}`,
				Expected: `resource "cloudflare_access_mutual_tls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = [{
    china_network                 = false
    client_certificate_forwarding = true
    hostname                      = "example.com"
  }]
}`,
			},
			{
				Name: "transform multiple settings blocks for old resource name",
				Input: `resource "cloudflare_access_mutual_tls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  settings {
    hostname = "app1.example.com"
    client_certificate_forwarding = true
    china_network = false
  }
  settings {
    hostname = "app2.example.com"
    client_certificate_forwarding = false
    china_network = false
  }
}`,
				Expected: `resource "cloudflare_access_mutual_tls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  settings = [{
    china_network                 = false
    client_certificate_forwarding = true
    hostname                      = "app1.example.com"
  }, {
    china_network                 = false
    client_certificate_forwarding = false
    hostname                      = "app2.example.com"
  }]
}`,
			},
			{
				Name: "handle dynamic blocks for old resource name",
				Input: `resource "cloudflare_access_mutual_tls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  dynamic "settings" {
    for_each = local.domains
    content {
      hostname = settings.value
      client_certificate_forwarding = true
      china_network = false
    }
  }
}`,
				Expected: `resource "cloudflare_access_mutual_tls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = [for value in local.domains : {
    hostname                      = value
    china_network                 = false
    client_certificate_forwarding = true
  }]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
