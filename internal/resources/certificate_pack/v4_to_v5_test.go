package certificate_pack

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
				Name: "basic certificate pack",
				Input: `
resource "cloudflare_certificate_pack" "example" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com", "*.cf-tftest.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}`,
				Expected: `resource "cloudflare_certificate_pack" "example" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com", "*.cf-tftest.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}`,
			},
			{
				Name: "wait_for_active_status removed",
				Input: `
resource "cloudflare_certificate_pack" "example" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com", "*.cf-tftest.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
  wait_for_active_status = true
}`,
				Expected: `resource "cloudflare_certificate_pack" "example" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com", "*.cf-tftest.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}`,
			},
			{
				Name: "certificate pack with all fields including optional",
				Input: `
resource "cloudflare_certificate_pack" "full" {
  zone_id                = "0da42c8d2132a9ddaf714f9e7c920711"
  type                   = "advanced"
  hosts                  = ["cf-tftest.com", "*.cf-tftest.com", "www.cf-tftest.com"]
  validation_method      = "http"
  validity_days          = 365
  certificate_authority  = "google"
  cloudflare_branding    = true
  wait_for_active_status = false
}`,
				Expected: `resource "cloudflare_certificate_pack" "full" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com", "*.cf-tftest.com", "www.cf-tftest.com"]
  validation_method     = "http"
  validity_days         = 365
  certificate_authority = "google"
  cloudflare_branding   = true
}`,
			},
			{
				Name: "computed fields removed from config",
				Input: `
resource "cloudflare_certificate_pack" "computed" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
  validation_records    = []
  validation_errors     = []
}`,
				Expected: `resource "cloudflare_certificate_pack" "computed" {
  zone_id               = "0da42c8d2132a9ddaf714f9e7c920711"
  type                  = "advanced"
  hosts                 = ["cf-tftest.com"]
  validation_method     = "txt"
  validity_days         = 90
  certificate_authority = "lets_encrypt"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic certificate pack state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-basic-123",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "lets_encrypt",
    "validation_records": [],
    "validation_errors": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-basic-123",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "lets_encrypt",
    "validation_records": [],
    "validation_errors": []
  }
}`,
			},
			{
				Name: "wait_for_active_status removed from state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-wait-456",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "lets_encrypt",
    "wait_for_active_status": true,
    "validation_records": [],
    "validation_errors": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-wait-456",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "lets_encrypt",
    "validation_records": [],
    "validation_errors": []
  }
}`,
			},
			{
				Name: "certificate pack with all fields including optional - state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-full-789",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com", "www.cf-tftest.com"],
    "validation_method": "http",
    "validity_days": 365,
    "certificate_authority": "google",
    "cloudflare_branding": true,
    "wait_for_active_status": false,
    "validation_records": [
      {
        "cname_target": "validation.cloudflare.com",
        "cname_name": "_acme-challenge.cf-tftest.com",
        "txt_name": "_acme-challenge.cf-tftest.com",
        "txt_value": "token-12345",
        "http_url": "http://cf-tftest.com/.well-known/acme-challenge/token",
        "http_body": "verification-content",
        "emails": ["admin@cf-tftest.com", "webmaster@cf-tftest.com"]
      }
    ],
    "validation_errors": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-full-789",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com", "www.cf-tftest.com"],
    "validation_method": "http",
    "validity_days": 365,
    "certificate_authority": "google",
    "cloudflare_branding": true,
    "validation_records": [
      {
        "txt_name": "_acme-challenge.cf-tftest.com",
        "txt_value": "token-12345",
        "http_url": "http://cf-tftest.com/.well-known/acme-challenge/token",
        "http_body": "verification-content",
        "emails": ["admin@cf-tftest.com", "webmaster@cf-tftest.com"]
      }
    ],
    "validation_errors": []
  }
}`,
			},
			{
				Name: "computed fields remain in state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-computed-890",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "lets_encrypt",
    "validation_records": [],
    "validation_errors": []
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-computed-890",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "lets_encrypt",
    "validation_records": [],
    "validation_errors": []
  }
}`,
			},
			{
				Name: "null validation_records and validation_errors converted to empty arrays",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-null-fields-999",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "google"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "cert-null-fields-999",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "type": "advanced",
    "hosts": ["cf-tftest.com", "*.cf-tftest.com"],
    "validation_method": "txt",
    "validity_days": 90,
    "certificate_authority": "google",
    "validation_records": [],
    "validation_errors": []
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

func TestMigratorInterface(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("GetResourceType", func(t *testing.T) {
		expected := "cloudflare_certificate_pack"
		if got := migrator.GetResourceType(); got != expected {
			t.Errorf("GetResourceType() = %v, want %v", got, expected)
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		tests := []struct {
			name         string
			resourceType string
			want         bool
		}{
			{"handles v4 name", "cloudflare_certificate_pack", true},
			{"doesn't handle other types", "cloudflare_other_resource", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := migrator.CanHandle(tt.resourceType); got != tt.want {
					t.Errorf("CanHandle(%v) = %v, want %v", tt.resourceType, got, tt.want)
				}
			})
		}
	})
}
