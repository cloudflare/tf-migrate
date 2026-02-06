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

	// Note: State transformation tests removed since the provider handles all state migrations
	// via StateUpgraders. The TransformState function in v4_to_v5.go is a no-op.
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
