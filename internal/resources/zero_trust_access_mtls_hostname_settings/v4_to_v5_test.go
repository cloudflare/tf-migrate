package zero_trust_access_mtls_hostname_settings

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic resource with single settings block",
			Input: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings {
    hostname                      = "example.com"
    china_network                 = true
    client_certificate_forwarding = false
  }
}`,
			Expected: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = [{
    hostname                      = "example.com"
    china_network                 = true
    client_certificate_forwarding = false
  }]
}`,
		},
		{
			Name: "multiple settings blocks",
			Input: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    hostname                      = "example.com"
    china_network                 = true
    client_certificate_forwarding = false
  }

  settings {
    hostname                      = "api.example.com"
    china_network                 = false
    client_certificate_forwarding = true
  }
}`,
			Expected: `resource "cloudflare_zero_trust_access_mtls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings = [{
    hostname                      = "example.com"
    china_network                 = true
    client_certificate_forwarding = false
    }, {
    hostname                      = "api.example.com"
    china_network                 = false
    client_certificate_forwarding = true
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
