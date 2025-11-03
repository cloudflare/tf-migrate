package zero_trust_access_identity_provider

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "resource name transformation",
				Input: `resource "cloudflare_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "onetimepin"
}`,
				Expected: `resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "onetimepin"
  config     = {}
}`,
			},
			{
				Name: "config block to object transformation",
				Input: `resource "cloudflare_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "github"
  config {
    client_id     = "test"
    client_secret = "secret"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "github"
  config = {
    client_id     = "test"
    client_secret = "secret"
  }
}`,
			},
			{
				Name: "certificate field transformation",
				Input: `resource "cloudflare_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "saml"
  config {
    idp_public_cert = "CERT123"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "saml"
  config = {
    idp_public_certs = ["CERT123"]
  }
}`,
			},
			{
				Name: "scim config block to object transformation",
				Input: `resource "cloudflare_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "azureAD"
  scim_config {
    enabled = true
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "azureAD"
  scim_config = {
    enabled = true
  }
  config = {}
}`,
			},
			{
				Name: "with both config and scim_config",
				Input: `resource "cloudflare_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "okta"
  config {
    client_id     = "test"
    client_secret = "secret"
  }
  scim_config {
    enabled = true
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "test"
  name       = "test"
  type       = "okta"
  config = {
    client_id     = "test"
    client_secret = "secret"
  }
  scim_config = {
    enabled = true
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
