package access_application

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "transform policies from list of strings to list of objects",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"

  policies = [
    cloudflare_zero_trust_access_policy.allow.id,
    cloudflare_zero_trust_access_policy.deny.id
  ]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"

  policies = [{ id = cloudflare_zero_trust_access_policy.allow.id }, { id = cloudflare_zero_trust_access_policy.deny.id }]
  type     = "self_hosted"
}`,
			},
			{
				Name: "transform policies with literal IDs",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"

  policies = ["policy-id-1", "policy-id-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"

  policies = [{ id = "policy-id-1" }, { id = "policy-id-2" }]
  type     = "self_hosted"
}`,
			},
			{
				Name: "no policies attribute but add default type",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
}`,
			},
			{
				Name: "remove domain_type attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id  = "abc123"
  name        = "Test App"
  domain      = "test.example.com"
  domain_type = "public"
  type        = "self_hosted"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
}`,
			},
			{
				Name: "convert single destinations block to list attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = "https://example.com"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {
      uri = "https://example.com"
    }
  ]
}`,
			},
			{
				Name: "convert multiple destinations blocks to list attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = "https://example.com"
  }

  destinations {
    uri = "tcp://db.example.com:5432"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"


  destinations = [
    {
      uri = "https://example.com"
    },
    {
      uri = "tcp://db.example.com:5432"
    }
  ]
}`,
			},
			{
				Name: "destinations block with multiple attributes",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri         = "https://app.example.com"
    description = "Main application"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {
      description = "Main application"
      uri         = "https://app.example.com"
    }
  ]
}`,
			},
			{
				Name: "remove skip_app_launcher_login_page when type is not app_launcher",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                   = "abc123"
  name                         = "Test App"
  domain                       = "test.example.com"
  type                         = "self_hosted"
  skip_app_launcher_login_page = false
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
}`,
			},
			{
				Name: "preserve skip_app_launcher_login_page when type is app_launcher",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                   = "abc123"
  name                         = "Test App"
  type                         = "app_launcher"
  skip_app_launcher_login_page = true
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                   = "abc123"
  name                         = "Test App"
  type                         = "app_launcher"
  skip_app_launcher_login_page = true
}`,
			},
			{
				Name: "transform toset to list for allowed_idps",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id   = "abc123"
  name         = "Test App"
  domain       = "test.example.com"
  allowed_idps = toset(["idp-1", "idp-2", "idp-3"])
  type         = "self_hosted"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id   = "abc123"
  name         = "Test App"
  domain       = "test.example.com"
  allowed_idps = ["idp-1", "idp-2", "idp-3"]
  type         = "self_hosted"
}`,
			},
			{
				Name: "empty policies array",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  policies   = []
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  policies   = []
  type       = "self_hosted"
}`,
			},
			{
				Name: "empty destinations block",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {}
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
