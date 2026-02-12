package origin_ca_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource (no changes needed)",
				Input: `resource "cloudflare_origin_ca_certificate" "example" {
  csr          = "-----BEGIN CERTIFICATE REQUEST-----\nMIICvDCCAaQCAQAwdzELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx\n-----END CERTIFICATE REQUEST-----"
  request_type = "origin-rsa"
  hostnames    = ["example.com"]
}`,
				Expected: `resource "cloudflare_origin_ca_certificate" "example" {
  csr          = "-----BEGIN CERTIFICATE REQUEST-----\nMIICvDCCAaQCAQAwdzELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx\n-----END CERTIFICATE REQUEST-----"
  request_type = "origin-rsa"
  hostnames    = ["example.com"]
}`,
			},
			{
				Name: "Complete resource with all v4 fields (min_days_for_renewal removed)",
				Input: `resource "cloudflare_origin_ca_certificate" "example" {
  csr                  = "-----BEGIN CERTIFICATE REQUEST-----\nMIICvDCCAaQCAQAwdzELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx\n-----END CERTIFICATE REQUEST-----"
  request_type         = "origin-rsa"
  hostnames            = ["example.com", "*.example.com"]
  requested_validity   = 365
  min_days_for_renewal = 30
}`,
				Expected: `resource "cloudflare_origin_ca_certificate" "example" {
  csr                = "-----BEGIN CERTIFICATE REQUEST-----\nMIICvDCCAaQCAQAwdzELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx\n-----END CERTIFICATE REQUEST-----"
  request_type       = "origin-rsa"
  hostnames          = ["example.com", "*.example.com"]
  requested_validity = 365
}`,
			},
			{
				Name: "Multiple hostnames",
				Input: `resource "cloudflare_origin_ca_certificate" "example" {
  csr          = "-----BEGIN CERTIFICATE REQUEST-----\ntest\n-----END CERTIFICATE REQUEST-----"
  request_type = "origin-ecc"
  hostnames    = ["a.example.com", "b.example.com", "*.example.com"]
}`,
				Expected: `resource "cloudflare_origin_ca_certificate" "example" {
  csr          = "-----BEGIN CERTIFICATE REQUEST-----\ntest\n-----END CERTIFICATE REQUEST-----"
  request_type = "origin-ecc"
  hostnames    = ["a.example.com", "b.example.com", "*.example.com"]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
