package custom_pages

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
				Name: "basic zone-scoped custom page",
				Input: `
resource "cloudflare_custom_pages" "test" {
  zone_id = var.cloudflare_zone_id
  type    = "500_errors"
  url     = "https://cf-tf-test.com/500.html"
  state   = "customized"
}`,
				Expected: `
resource "cloudflare_custom_pages" "test" {
  zone_id    = var.cloudflare_zone_id
  url        = "https://cf-tf-test.com/500.html"
  state      = "customized"
  identifier = "500_errors"
}`,
			},
			{
				Name: "account-scoped custom page",
				Input: `
resource "cloudflare_custom_pages" "test" {
  account_id = var.cloudflare_account_id
  type       = "basic_challenge"
  url        = "https://cf-tf-test.com/challenge.html"
  state      = "default"
}`,
				Expected: `
resource "cloudflare_custom_pages" "test" {
  account_id = var.cloudflare_account_id
  url        = "https://cf-tf-test.com/challenge.html"
  state      = "default"
  identifier = "basic_challenge"
}`,
			},
			{
				Name: "missing state field - default added",
				Input: `
resource "cloudflare_custom_pages" "test" {
  zone_id = var.cloudflare_zone_id
  type    = "500_errors"
  url     = "https://cf-tf-test.com/500.html"
}`,
				Expected: `
resource "cloudflare_custom_pages" "test" {
  zone_id    = var.cloudflare_zone_id
  url        = "https://cf-tf-test.com/500.html"
  identifier = "500_errors"
  state      = "default"
}`,
			},
			{
				Name: "multiple page types",
				Input: `
resource "cloudflare_custom_pages" "error_500" {
  zone_id = var.cloudflare_zone_id
  type    = "500_errors"
  url     = "https://cf-tf-test.com/500.html"
  state   = "customized"
}

resource "cloudflare_custom_pages" "error_1000" {
  zone_id = var.cloudflare_zone_id
  type    = "1000_errors"
  url     = "https://cf-tf-test.com/1000.html"
  state   = "customized"
}`,
				Expected: `
resource "cloudflare_custom_pages" "error_500" {
  zone_id    = var.cloudflare_zone_id
  url        = "https://cf-tf-test.com/500.html"
  state      = "customized"
  identifier = "500_errors"
}

resource "cloudflare_custom_pages" "error_1000" {
  zone_id    = var.cloudflare_zone_id
  url        = "https://cf-tf-test.com/1000.html"
  state      = "customized"
  identifier = "1000_errors"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// State transformation tests removed - state migration is now handled by provider's StateUpgraders
	t.Run("StateTransformation_Removed", func(t *testing.T) {
		t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
	})
}
