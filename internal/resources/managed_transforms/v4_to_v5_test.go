package managed_transforms

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
				Name: "adds missing managed_response_headers",
				Input: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    }
  ]
}`,
				Expected: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    }
  ]
  managed_response_headers = []
}`,
			},
			{
				Name: "adds missing managed_request_headers",
				Input: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
  managed_response_headers = [
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
}`,
				Expected: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
  managed_response_headers = [
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
  managed_request_headers = []
}`,
			},
			{
				Name: "adds both missing attributes",
				Input: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
}`,
				Expected: `resource "cloudflare_managed_transforms" "test" {
  zone_id                  = "abc123"
  managed_request_headers  = []
  managed_response_headers = []
}`,
			},
			{
				Name: "preserves existing attributes",
				Input: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    }
  ]
  managed_response_headers = [
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
}`,
				Expected: `resource "cloudflare_managed_transforms" "test" {
  zone_id = "abc123"
  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    }
  ]
  managed_response_headers = [
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
