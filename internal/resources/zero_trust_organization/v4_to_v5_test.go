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
}`,
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
}`,
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
}`,
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
}`,
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
}`,
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
}`,
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

resource "cloudflare_zero_trust_organization" "zone_org" {
  zone_id     = "023e105f4ecef8ad9ca31a8372d0c353"
  auth_domain = "zone.cloudflareaccess.com"
  name        = "Zone Organization"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization"
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "allow_authenticate_via_warp": false,
    "auto_redirect_to_identity": false,
    "is_ui_read_only": false
  }
}`,
			},
			{
				Name: "With login_design array (MaxItems:1)",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "login_design": [{
      "background_color": "#000000",
      "text_color": "#FFFFFF",
      "logo_path": "https://example.com/logo.png",
      "header_text": "Welcome",
      "footer_text": "Powered by Cloudflare"
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "login_design": {
      "background_color": "#000000",
      "text_color": "#FFFFFF",
      "logo_path": "https://example.com/logo.png",
      "header_text": "Welcome",
      "footer_text": "Powered by Cloudflare"
    },
    "allow_authenticate_via_warp": false,
    "auto_redirect_to_identity": false,
    "is_ui_read_only": false
  }
}`,
			},
			{
				Name: "Empty login_design array",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "login_design": []
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "allow_authenticate_via_warp": false,
    "auto_redirect_to_identity": false,
    "is_ui_read_only": false
  }
}`,
			},
			{
				Name: "Zone-scoped organization (zone_id instead of account_id)",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization"
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "allow_authenticate_via_warp": false,
    "auto_redirect_to_identity": false,
    "is_ui_read_only": false
  }
}`,
			},
			{
				Name: "With custom_pages array (MaxItems:1)",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "custom_pages": [{
      "forbidden": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "identity_denied": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "custom_pages": {
      "forbidden": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "identity_denied": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
    },
    "allow_authenticate_via_warp": false,
    "auto_redirect_to_identity": false,
    "is_ui_read_only": false
  }
}`,
			},
			{
				Name: "With both login_design and custom_pages arrays",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "login_design": [{
      "background_color": "#000000",
      "text_color": "#FFFFFF"
    }],
    "custom_pages": [{
      "forbidden": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "login_design": {
      "background_color": "#000000",
      "text_color": "#FFFFFF"
    },
    "custom_pages": {
      "forbidden": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    },
    "allow_authenticate_via_warp": false,
    "auto_redirect_to_identity": false,
    "is_ui_read_only": false
  }
}`,
			},
			{
				Name: "With existing boolean values (should be preserved)",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "allow_authenticate_via_warp": true,
    "auto_redirect_to_identity": true,
    "is_ui_read_only": true
  }
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "My Organization",
    "allow_authenticate_via_warp": true,
    "auto_redirect_to_identity": true,
    "is_ui_read_only": true
  }
}`,
			},
			{
				Name: "Complete state with all fields",
				Input: `{
  "type": "cloudflare_zero_trust_access_organization",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "Complete Organization",
    "session_duration": "24h",
    "user_seat_expiration_inactive_time": "730h",
    "warp_auth_session_duration": "12h",
    "ui_read_only_toggle_reason": "Managed by Terraform",
    "is_ui_read_only": true,
    "auto_redirect_to_identity": true,
    "allow_authenticate_via_warp": true,
    "login_design": [{
      "background_color": "#000000",
      "text_color": "#FFFFFF",
      "logo_path": "https://example.com/logo.png",
      "header_text": "Welcome",
      "footer_text": "Powered by Cloudflare"
    }],
    "custom_pages": [{
      "forbidden": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "identity_denied": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_access_organization",
  "name": "example",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "auth_domain": "example.cloudflareaccess.com",
    "name": "Complete Organization",
    "session_duration": "24h",
    "user_seat_expiration_inactive_time": "730h",
    "warp_auth_session_duration": "12h",
    "ui_read_only_toggle_reason": "Managed by Terraform",
    "is_ui_read_only": true,
    "auto_redirect_to_identity": true,
    "allow_authenticate_via_warp": true,
    "login_design": {
      "background_color": "#000000",
      "text_color": "#FFFFFF",
      "logo_path": "https://example.com/logo.png",
      "header_text": "Welcome",
      "footer_text": "Powered by Cloudflare"
    },
    "custom_pages": {
      "forbidden": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "identity_denied": "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
    }
  }
}`,
			},
			{
				Name: "Invalid instance (no attributes) - should still set schema_version",
				Input: `{
  "type": "cloudflare_access_organization",
  "name": "example"
}`,
				Expected: `{
  "type": "cloudflare_access_organization",
  "name": "example",
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
