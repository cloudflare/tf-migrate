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

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "complete state with all fields",
				Input: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "name": "example-ca",
    "private_key": null,
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "name": "example-ca",
    "private_key": null,
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
			},
			{
				Name: "minimal state with required fields only",
				Input: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
			},
			{
				Name: "state with null optional fields",
				Input: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "name": null,
    "private_key": null,
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "name": null,
    "private_key": null,
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
			},
			{
				Name: "state missing attributes",
				Input: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example"
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example"
}`,
			},
			{
				Name: "remove new v5 computed fields if present",
				Input: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "id": "cert-id-123",
    "updated_at": "2024-01-01T00:00:00Z",
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "type": "cloudflare_mtls_certificate",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ca": true,
    "certificates": "-----BEGIN CERTIFICATE-----\nMIIBkTCB...\n-----END CERTIFICATE-----",
    "id": "cert-id-123",
    "issuer": "CN=Test CA",
    "signature": "SHA256withRSA",
    "serial_number": "00:aa:bb:cc:dd",
    "uploaded_on": "2024-01-01T00:00:00Z",
    "expires_on": "2025-01-01T00:00:00Z"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
