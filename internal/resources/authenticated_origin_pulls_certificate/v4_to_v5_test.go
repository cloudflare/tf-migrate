package authenticated_origin_pulls_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation_PerZone(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "per-zone certificate with all fields",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
  type        = "per-zone"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
}`,
		},
		{
			Name: "per-zone certificate with variable references",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("cert.pem")
  private_key = file("key.pem")
  type        = "per-zone"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("cert.pem")
  private_key = file("key.pem")
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestConfigTransformation_PerHostname(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "per-hostname certificate with all fields",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
  type        = "per-hostname"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
}

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.test
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.test
}`,
		},
		{
			Name: "per-hostname certificate with variable references",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("hostname_cert.pem")
  private_key = file("hostname_key.pem")
  type        = "per-hostname"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("hostname_cert.pem")
  private_key = file("hostname_key.pem")
}

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.test
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.test
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestConfigTransformation_Mixed(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "both per-zone and per-hostname in same file",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "per_zone" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nzone cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nzone key\n-----END PRIVATE KEY-----"
  type        = "per-zone"
}

resource "cloudflare_authenticated_origin_pulls_certificate" "per_hostname" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nhost cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nhost key\n-----END PRIVATE KEY-----"
  type        = "per-hostname"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_certificate" "per_zone" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nzone cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nzone key\n-----END PRIVATE KEY-----"
}

resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "per_hostname" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nhost cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nhost key\n-----END PRIVATE KEY-----"
}

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.per_hostname
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.per_hostname
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformation_PerZone(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// State transformation is now handled by provider-side StateUpgraders
	// tf-migrate only handles resource type routing, not field transformations
	tests := []testhelpers.StateTestCase{
		{
			Name: "per-zone certificate state passes through unchanged",
			Input: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-zone",
					"serial_number": "12345",
					"issuer": "CN=Example CA",
					"signature": "SHA256WithRSA",
					"expires_on": "2025-12-31T23:59:59Z",
					"status": "active",
					"uploaded_on": "2024-01-01T00:00:00Z"
				}
			}`,
			// State unchanged - provider StateUpgraders will handle field removal
			Expected: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-zone",
					"serial_number": "12345",
					"issuer": "CN=Example CA",
					"signature": "SHA256WithRSA",
					"expires_on": "2025-12-31T23:59:59Z",
					"status": "active",
					"uploaded_on": "2024-01-01T00:00:00Z"
				}
			}`,
		},
		{
			Name: "per-zone certificate state with minimal fields passes through unchanged",
			Input: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-zone"
				}
			}`,
			// State unchanged - provider will handle field transformations
			Expected: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-zone"
				}
			}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func TestStateTransformation_PerHostname(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// State transformation is now handled by provider-side StateUpgraders
	// tf-migrate only handles resource type routing, not field transformations
	tests := []testhelpers.StateTestCase{
		{
			Name: "per-hostname certificate state passes through unchanged",
			Input: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-hostname",
					"serial_number": "67890",
					"issuer": "CN=Example CA",
					"signature": "SHA256WithRSA",
					"expires_on": "2025-12-31T23:59:59Z",
					"status": "active",
					"uploaded_on": "2024-01-01T00:00:00Z"
				}
			}`,
			// State unchanged - provider StateUpgraders will handle field removal
			Expected: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-hostname",
					"serial_number": "67890",
					"issuer": "CN=Example CA",
					"signature": "SHA256WithRSA",
					"expires_on": "2025-12-31T23:59:59Z",
					"status": "active",
					"uploaded_on": "2024-01-01T00:00:00Z"
				}
			}`,
		},
		{
			Name: "per-hostname certificate state with minimal fields passes through unchanged",
			Input: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-hostname"
				}
			}`,
			// State unchanged - provider will handle field transformations
			Expected: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-hostname"
				}
			}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func TestStateTransformation_EdgeCases(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// State transformation is now handled by provider-side StateUpgraders
	// tf-migrate passes state through unchanged
	tests := []testhelpers.StateTestCase{
		{
			Name: "missing type field - state passes through unchanged",
			Input: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
				}
			}`,
			// State unchanged - provider will default to per-zone
			Expected: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
				}
			}`,
		},
		{
			Name: "null computed fields are preserved including type field",
			Input: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-zone",
					"issuer": null,
					"signature": null,
					"expires_on": null,
					"status": null,
					"uploaded_on": null
				}
			}`,
			// State unchanged - provider will handle removal of type field
			Expected: `{
				"schema_version": 0,
				"attributes": {
					"zone_id": "abc123",
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
					"type": "per-zone",
					"issuer": null,
					"signature": null,
					"expires_on": null,
					"status": null,
					"uploaded_on": null
				}
			}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
