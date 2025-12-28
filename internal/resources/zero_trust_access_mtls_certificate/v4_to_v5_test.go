package zero_trust_access_mtls_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic resource with account_id and associated_hostnames",
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
				Name: "Basic resource with zone_id",
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
				Name: "Minimal resource with only required fields",
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
				Name: "Legacy resource name - cloudflare_access_mutual_tls_certificate",
				Input: `
resource "cloudflare_access_mutual_tls_certificate" "legacy" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "legacy"
  certificate          = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
  associated_hostnames = ["legacy.example.com"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "legacy" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "legacy"
  certificate          = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
  associated_hostnames = ["legacy.example.com"]
}`,
			},
			{
				Name: "Resource with empty associated_hostnames",
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
				Name: "Multiple resources in one file",
				Input: `
resource "cloudflare_zero_trust_access_mtls_certificate" "first" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "first"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT1\n-----END CERTIFICATE-----"
}

resource "cloudflare_zero_trust_access_mtls_certificate" "second" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "second"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT2\n-----END CERTIFICATE-----"
}`,
				Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "first" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "first"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT1\n-----END CERTIFICATE-----"
}

resource "cloudflare_zero_trust_access_mtls_certificate" "second" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "second"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT2\n-----END CERTIFICATE-----"
}`,
			},
			{
				Name: "Resource without associated_hostnames (should be unchanged in config)",
				Input: `
resource "cloudflare_zero_trust_access_mtls_certificate" "no_hostnames" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "no_hostnames"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}`,
				Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "no_hostnames" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "no_hostnames"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
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
				Expected: `resource "cloudflare_zero_trust_access_mtls_certificate" "old" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "old"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}

resource "cloudflare_zero_trust_access_mtls_certificate" "new" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "new"
  certificate = "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Complete resource with all fields",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "cert-123",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "example",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "associated_hostnames": ["example.com", "app.example.com"],
        "fingerprint": "MD5_FINGERPRINT_HERE"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "cert-123",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "example",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "associated_hostnames": ["example.com", "app.example.com"],
        "fingerprint": "MD5_FINGERPRINT_HERE"
      }
    }]
  }]
}`,
			},
			{
				Name: "Resource with missing associated_hostnames - should add empty array",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "cert-456",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "test",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "fingerprint": "MD5_FINGERPRINT"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "cert-456",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "test",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "fingerprint": "MD5_FINGERPRINT",
        "associated_hostnames": []
      }
    }]
  }]
}`,
			},
			{
				Name: "Resource with empty associated_hostnames - should keep as empty array",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "cert-789",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "name": "empty",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "associated_hostnames": []
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "cert-789",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "name": "empty",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "associated_hostnames": []
      }
    }]
  }]
}`,
			},
			{
				Name: "Legacy resource name - should be renamed in state",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_access_mutual_tls_certificate",
    "name": "legacy",
    "instances": [{
      "attributes": {
        "id": "cert-legacy",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "legacy",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "fingerprint": "MD5_FINGERPRINT"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "legacy",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "cert-legacy",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "legacy",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "fingerprint": "MD5_FINGERPRINT",
        "associated_hostnames": []
      }
    }]
  }]
}`,
			},
			{
				Name: "Multiple instances",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [
      {
        "attributes": {
          "id": "cert-1",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "first",
          "certificate": "-----BEGIN CERTIFICATE-----\nCERT1\n-----END CERTIFICATE-----"
        }
      },
      {
        "attributes": {
          "id": "cert-2",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "second",
          "certificate": "-----BEGIN CERTIFICATE-----\nCERT2\n-----END CERTIFICATE-----",
          "associated_hostnames": ["example.com"]
        }
      }
    ]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "test",
    "instances": [
      {
        "schema_version": 0,
        "attributes": {
          "id": "cert-1",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "first",
          "certificate": "-----BEGIN CERTIFICATE-----\nCERT1\n-----END CERTIFICATE-----",
          "associated_hostnames": []
        }
      },
      {
        "schema_version": 0,
        "attributes": {
          "id": "cert-2",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "second",
          "certificate": "-----BEGIN CERTIFICATE-----\nCERT2\n-----END CERTIFICATE-----",
          "associated_hostnames": ["example.com"]
        }
      }
    ]
  }]
}`,
			},
			{
				Name: "Resource with zone_id instead of account_id",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "zone_cert",
    "instances": [{
      "attributes": {
        "id": "cert-zone",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "name": "zone_certificate",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "fingerprint": "MD5_FINGERPRINT"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_mtls_certificate",
    "name": "zone_cert",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "cert-zone",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "name": "zone_certificate",
        "certificate": "-----BEGIN CERTIFICATE-----\nCERT\n-----END CERTIFICATE-----",
        "fingerprint": "MD5_FINGERPRINT",
        "associated_hostnames": []
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
