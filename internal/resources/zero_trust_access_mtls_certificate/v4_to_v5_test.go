package zero_trust_access_mtls_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Basic resource with account_id and associated_hostnames - no rename",
			Input: `
resource "cloudflare_zero_trust_access_mtls_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "example"
  certificate          = "-----BEGIN CERTIFICATE-----\nMIIFJDCCBAygAwIBAgIQB...\n-----END CERTIFICATE-----"
  associated_hostnames = ["example.com", "app.example.com"]
}`,
			Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "example"
  certificate          = "-----BEGIN CERTIFICATE-----\nMIIFJDCCBAygAwIBAgIQB...\n-----END CERTIFICATE-----"
  associated_hostnames = ["example.com", "app.example.com"]
}`,
		},
		{
			Name: "Basic resource with zone_id - no rename",
			Input: `
resource "cloudflare_zero_trust_access_mtls_certificate" "test" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "example"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIFJDCCBAygAwIBAgIQB...\n-----END CERTIFICATE-----"
}`,
			Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "test" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "example"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIFJDCCBAygAwIBAgIQB...\n-----END CERTIFICATE-----"
}`,
		},
		{
			Name: "Minimal resource - no rename",
			Input: `
resource "cloudflare_zero_trust_access_mtls_certificate" "minimal" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "minimal"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT_DATA\n-----END CERTIFICATE-----"
}`,
			Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "minimal" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "minimal"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT_DATA\n-----END CERTIFICATE-----"
}`,
		},
		{
			Name: "Legacy resource name - rename with moved block",
			Input: `
resource "cloudflare_access_mutual_tls_certificate" "legacy" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "legacy"
  certificate          = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
  associated_hostnames = ["legacy.example.com"]
}`,
			Expected: `
resource "cloudflare_zero_trust_access_mtls_certificate" "legacy" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "legacy"
  certificate          = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
  associated_hostnames = ["legacy.example.com"]
}
moved {
  from = cloudflare_access_mutual_tls_certificate.legacy
  to   = cloudflare_zero_trust_access_mtls_certificate.legacy
}
`,
		},
		{
			Name: "Resource with empty associated_hostnames - no rename",
			Input: `
resource "cloudflare_zero_trust_access_mtls_certificate" "empty" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "empty"
  certificate          = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
  associated_hostnames = []
}`,
			Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "empty" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "empty"
  certificate          = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
  associated_hostnames = []
}`,
		},
		{
			Name: "Mixed legacy and new resource names",
			Input: `
resource "cloudflare_access_mutual_tls_certificate" "old" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "old"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}

resource "cloudflare_zero_trust_access_mtls_certificate" "new" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "new"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}`,
			Expected: `
resource "cloudflare_zero_trust_access_mtls_certificate" "old" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "old"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}
moved {
  from = cloudflare_access_mutual_tls_certificate.old
  to   = cloudflare_zero_trust_access_mtls_certificate.old
}

resource "cloudflare_zero_trust_access_mtls_certificate" "new" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "new"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
