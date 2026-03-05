package mtls_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic CA certificate",
				Input: `
resource "cloudflare_mtls_certificate" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
  name         = "example-ca"
}`,
				Expected: `
resource "cloudflare_mtls_certificate" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
  name         = "example-ca"
}`,
			},
			{
				Name: "leaf certificate with private key",
				Input: `
resource "cloudflare_mtls_certificate" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = false
  certificates = "-----BEGIN CERTIFICATE-----\nMIICUTCC...\n-----END CERTIFICATE-----"
  private_key  = "-----BEGIN PRIVATE KEY-----\nMIICdgIB...\n-----END PRIVATE KEY-----"
  name         = "example-leaf"
}`,
				Expected: `
resource "cloudflare_mtls_certificate" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = false
  certificates = "-----BEGIN CERTIFICATE-----\nMIICUTCC...\n-----END CERTIFICATE-----"
  private_key  = "-----BEGIN PRIVATE KEY-----\nMIICdgIB...\n-----END PRIVATE KEY-----"
  name         = "example-leaf"
}`,
			},
			{
				Name: "minimal certificate without name",
				Input: `
resource "cloudflare_mtls_certificate" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
}`,
				Expected: `
resource "cloudflare_mtls_certificate" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
}`,
			},
			{
				Name: "multiple certificates",
				Input: `
resource "cloudflare_mtls_certificate" "ca1" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
  name         = "ca-1"
}

resource "cloudflare_mtls_certificate" "ca2" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIICUTCC...\n-----END CERTIFICATE-----"
  name         = "ca-2"
}`,
				Expected: `
resource "cloudflare_mtls_certificate" "ca1" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----"
  name         = "ca-1"
}

resource "cloudflare_mtls_certificate" "ca2" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIICUTCC...\n-----END CERTIFICATE-----"
  name         = "ca-2"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}
