package zero_trust_organization

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource - cloudflare_access_organization",
				Input: `resource "cloudflare_access_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"
}

moved {
  from = cloudflare_access_organization.example
  to   = cloudflare_zero_trust_organization.example
}
`,
			},
			{
				Name: "Minimal resource - cloudflare_zero_trust_access_organization",
				Input: `resource "cloudflare_zero_trust_access_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"
}

moved {
  from = cloudflare_zero_trust_access_organization.example
  to   = cloudflare_zero_trust_organization.example
}
`,
			},
			{
				Name: "With login_design block",
				Input: `resource "cloudflare_access_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"

  login_design {
    background_color = "#000000"
    text_color       = "#FFFFFF"
    logo_path        = "https://example.com/logo.png"
    header_text      = "Welcome"
    footer_text      = "Powered by Cloudflare"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"

  login_design = {
    background_color = "#000000"
    text_color       = "#FFFFFF"
    logo_path        = "https://example.com/logo.png"
    header_text      = "Welcome"
    footer_text      = "Powered by Cloudflare"
  }
}

moved {
  from = cloudflare_access_organization.example
  to   = cloudflare_zero_trust_organization.example
}
`,
			},
			{
				Name: "With custom_pages block",
				Input: `resource "cloudflare_access_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"

  custom_pages {
    forbidden       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    identity_denied = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"

  custom_pages = {
    forbidden       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    identity_denied = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
  }
}

moved {
  from = cloudflare_access_organization.example
  to   = cloudflare_zero_trust_organization.example
}
`,
			},
			{
				Name: "Complete organization with all fields",
				Input: `resource "cloudflare_access_organization" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  auth_domain = "example.cloudflareaccess.com"
  name        = "Complete Organization"

  is_ui_read_only            = true
  ui_read_only_toggle_reason = "Managed by Terraform"

  user_seat_expiration_inactive_time = "730h"
  auto_redirect_to_identity          = true
  session_duration                   = "24h"

  login_design {
    background_color = "#000000"
    text_color       = "#FFFFFF"
    logo_path        = "https://example.com/logo.png"
    header_text      = "Welcome"
    footer_text      = "Powered by Cloudflare"
  }

  custom_pages {
    forbidden       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    identity_denied = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
  }

  allow_authenticate_via_warp = true
  warp_auth_session_duration  = "12h"
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  auth_domain = "example.cloudflareaccess.com"
  name        = "Complete Organization"

  is_ui_read_only            = true
  ui_read_only_toggle_reason = "Managed by Terraform"

  user_seat_expiration_inactive_time = "730h"
  auto_redirect_to_identity          = true
  session_duration                   = "24h"

  login_design = {
    background_color = "#000000"
    text_color       = "#FFFFFF"
    logo_path        = "https://example.com/logo.png"
    header_text      = "Welcome"
    footer_text      = "Powered by Cloudflare"
  }

  custom_pages = {
    forbidden       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    identity_denied = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
  }

  allow_authenticate_via_warp = true
  warp_auth_session_duration  = "12h"
}

moved {
  from = cloudflare_access_organization.example
  to   = cloudflare_zero_trust_organization.example
}
`,
			},
			{
				Name: "Partial login_design block",
				Input: `resource "cloudflare_access_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"

  login_design {
    background_color = "#000000"
    text_color       = "#FFFFFF"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "example" {
  auth_domain = "example.cloudflareaccess.com"
  name        = "My Organization"

  login_design = {
    background_color = "#000000"
    text_color       = "#FFFFFF"
  }
}

moved {
  from = cloudflare_access_organization.example
  to   = cloudflare_zero_trust_organization.example
}
`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `resource "cloudflare_access_organization" "account_org" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  auth_domain = "account.cloudflareaccess.com"
  name        = "Account Organization"
}

resource "cloudflare_zero_trust_access_organization" "zone_org" {
  zone_id     = "023e105f4ecef8ad9ca31a8372d0c353"
  auth_domain = "zone.cloudflareaccess.com"
  name        = "Zone Organization"
}`,
				Expected: `resource "cloudflare_zero_trust_organization" "account_org" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  auth_domain = "account.cloudflareaccess.com"
  name        = "Account Organization"
}

moved {
  from = cloudflare_access_organization.account_org
  to   = cloudflare_zero_trust_organization.account_org
}

resource "cloudflare_zero_trust_organization" "zone_org" {
  zone_id     = "023e105f4ecef8ad9ca31a8372d0c353"
  auth_domain = "zone.cloudflareaccess.com"
  name        = "Zone Organization"
}

moved {
  from = cloudflare_zero_trust_access_organization.zone_org
  to   = cloudflare_zero_trust_organization.zone_org
}
`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
