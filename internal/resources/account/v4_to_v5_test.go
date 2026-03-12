package account

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "minimum config - no enforce_twofactor",
				Input: `
resource "cloudflare_account" "example" {
  name = "My Account"
}`,
				Expected: `
resource "cloudflare_account" "example" {
  name = "My Account"
}`,
			},
			{
				Name: "enforce_twofactor true wraps into settings",
				Input: `
resource "cloudflare_account" "example" {
  name              = "My Account"
  enforce_twofactor = true
}`,
				Expected: `
resource "cloudflare_account" "example" {
  name = "My Account"
  settings = {
    enforce_twofactor = true
  }
}`,
			},
			{
				Name: "enforce_twofactor false wraps into settings",
				Input: `
resource "cloudflare_account" "example" {
  name              = "My Account"
  enforce_twofactor = false
}`,
				Expected: `
resource "cloudflare_account" "example" {
  name = "My Account"
  settings = {
    enforce_twofactor = false
  }
}`,
			},
			{
				Name: "full config with type",
				Input: `
resource "cloudflare_account" "example" {
  name              = "My Account"
  type              = "standard"
  enforce_twofactor = true
}`,
				Expected: `
resource "cloudflare_account" "example" {
  name = "My Account"
  type = "standard"
  settings = {
    enforce_twofactor = true
  }
}`,
			},
			{
				Name: "enforce_twofactor with variable reference",
				Input: `
resource "cloudflare_account" "example" {
  name              = "My Account"
  enforce_twofactor = var.enforce_2fa
}`,
				Expected: `
resource "cloudflare_account" "example" {
  name = "My Account"
  settings = {
    enforce_twofactor = var.enforce_2fa
  }
}`,
			},
			{
				Name: "enforce_twofactor with local reference",
				Input: `
resource "cloudflare_account" "example" {
  name              = var.account_name
  enforce_twofactor = local.enforce_2fa
}`,
				Expected: `
resource "cloudflare_account" "example" {
  name = var.account_name
  settings = {
    enforce_twofactor = local.enforce_2fa
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Note: StateTransformation test removed.
	// State migration is now handled by the provider's StateUpgrader (v5.19+).
	// The provider implements UpgradeState with slot 0 handling v4 SDKv2 state
	// (schema_version=0) and transforming enforce_twofactor into settings.enforce_twofactor.
	// tf-migrate's TransformState returns input unchanged (no-op).
}
