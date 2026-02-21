package load_balancer_monitor

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"
}`,
			},
			{
				Name: "Resource with single header",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header {
    header = "Host"
    values = ["cf-tf-test.com"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header = {
    "Host" = ["cf-tf-test.com"]
  }
}`,
			},
			{
				Name: "Resource with multiple headers",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header {
    header = "Host"
    values = ["cf-tf-test.com"]
  }

  header {
    header = "User-Agent"
    values = ["Mozilla/5.0"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id = "abc123"

  header = {
    "Host"       = ["cf-tf-test.com"]
    "User-Agent" = ["Mozilla/5.0"]
  }
}`,
			},
			{
				Name: "Complete resource with all fields",
				Input: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id       = "abc123"
  type             = "https"
  interval         = 60
  retries          = 2
  timeout          = 5
  method           = "GET"
  path             = "/health"
  expected_codes   = "2xx"
  expected_body    = "alive"
  follow_redirects = true
  allow_insecure   = false
  description      = "Test monitor"
  port             = 443

  header {
    header = "Host"
    values = ["cf-tf-test.com"]
  }
}`,
				Expected: `resource "cloudflare_load_balancer_monitor" "example" {
  account_id       = "abc123"
  type             = "https"
  interval         = 60
  retries          = 2
  timeout          = 5
  method           = "GET"
  path             = "/health"
  expected_codes   = "2xx"
  expected_body    = "alive"
  follow_redirects = true
  allow_insecure   = false
  description      = "Test monitor"
  port             = 443

  header = {
    "Host" = ["cf-tf-test.com"]
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}

// TestV4ToV5TransformationState_Removed is a marker indicating state transformation tests were removed.
// State migration is now handled by the provider's StateUpgraders.
func TestV4ToV5TransformationState_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}
