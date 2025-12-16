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

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state transformation",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "500_errors",
    "zone_id": "14e48053f1836c10cb86ba178633826a",
    "account_id": null,
    "type": "500_errors",
    "url": "https://cf-tf-test.com/500.html",
    "state": "customized"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "500_errors",
    "zone_id": "14e48053f1836c10cb86ba178633826a",
    "account_id": null,
    "identifier": "500_errors",
    "url": "https://cf-tf-test.com/500.html",
    "state": "customized"
  }
}`,
			},
			{
				Name: "state transformation with missing state field",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "500_errors",
    "zone_id": "14e48053f1836c10cb86ba178633826a",
    "type": "500_errors",
    "url": "https://cf-tf-test.com/500.html"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "500_errors",
    "zone_id": "14e48053f1836c10cb86ba178633826a",
    "identifier": "500_errors",
    "url": "https://cf-tf-test.com/500.html",
    "state": "default"
  }
}`,
			},
			{
				Name: "account-scoped state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "basic_challenge",
    "account_id": "dbb2ef7988061049c4bd32bf6883fde0",
    "zone_id": null,
    "type": "basic_challenge",
    "url": "https://cf-tf-test.com/challenge.html",
    "state": "default"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "basic_challenge",
    "account_id": "dbb2ef7988061049c4bd32bf6883fde0",
    "zone_id": null,
    "identifier": "basic_challenge",
    "url": "https://cf-tf-test.com/challenge.html",
    "state": "default"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
