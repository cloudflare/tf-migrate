package zero_trust_access_applications

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource with old name",
				Input: `resource "cloudflare_access_application" "rename" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "rename" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			},
			{
				Name: "Minimal resource with new name - no change",
				Input: `resource "cloudflare_zero_trust_access_application" "no_rename" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "no_rename" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			},
			{
				Name: "Minimal resource with basic, unchanged fields",
				Input: `resource "cloudflare_access_application" "basic" {
  account_id                    = "1234"
  allow_authenticate_via_warp   = true
  app_launcher_logo_url         = true
  app_launcher_visible          = true
  auto_redirect_to_identity     = true
  bg_color                      = "#000000"
  custom_deny_message           = "message"
  custom_deny_url               = "www.example.com"
  custom_non_identity_deny_url  = "www.example.com"
  domain                        = "test.example.com/admin"
  enable_binding_cookie         = true
  header_bg_color               = "#000000"
  http_only_cookie_attribute    = true
  logo_url                      = "www.example.com"
  name                          = "test"
  options_preflight_bypass      = true
  same_site_cookie_attribute    = "strict"
  service_auth_401_redirect     = true
  session_duration              = "24h"
  skip_app_launcher_login_page  = true
  skip_interstitial             = true
  type                          = "self_hosted"
  zone_id                       = "1234"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "basic" {
  account_id                    = "1234"
  allow_authenticate_via_warp   = true
  app_launcher_logo_url         = true
  app_launcher_visible          = true
  auto_redirect_to_identity     = true
  bg_color                      = "#000000"
  custom_deny_message           = "message"
  custom_deny_url               = "www.example.com"
  custom_non_identity_deny_url  = "www.example.com"
  domain                        = "test.example.com/admin"
  enable_binding_cookie         = true
  header_bg_color               = "#000000"
  http_only_cookie_attribute    = true
  logo_url                      = "www.example.com"
  name                          = "test"
  options_preflight_bypass      = true
  same_site_cookie_attribute    = "strict"
  service_auth_401_redirect     = true
  session_duration              = "24h"
  skip_app_launcher_login_page  = true
  skip_interstitial             = true
  type                          = "self_hosted"
  zone_id                       = "1234"
}`,
			},
			{
				Name: "Minimal resource with string set, unchanged fields",
				Input: `resource "cloudflare_zero_trust_access_application" "basic_string_set" {
  account_id          = "1234"
  allowed_idps        = ["1234", "5678"]
  custom_pages        = ["1234", "5678"]
  self_hosted_domains = ["1234", "5678"]
  tags                = ["1234", "5678"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "basic_string_set" {
  account_id          = "1234"
  allowed_idps        = ["1234", "5678"]
  custom_pages        = ["1234", "5678"]
  self_hosted_domains = ["1234", "5678"]
  tags                = ["1234", "5678"]
}`,
			},
			{
				Name: "Remove domain_type attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "remove_domain_type" {
  account_id  = "1234"
  domain_type = "public"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "remove_domain_type" {
  account_id = "1234"
}`,
			},
			{
				Name: "Resource with cors_headers",
				Input: `resource "cloudflare_zero_trust_access_application" "cors_headers" {
  account_id = "1234"
  cors_headers {
    allow_all_headers = true
    allow_all_methods = true
    allow_all_origins = true
    allow_credentials = true
    allowed_headers   = ["string"]
    allowed_methods   = ["GET"]
    allowed_origins   = ["https://example.com"]
    max_age           = 1
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "cors_headers" {
  account_id   = "1234"
  cors_headers = {
    allow_all_headers = true
    allow_all_methods = true
    allow_all_origins = true
    allow_credentials = true
    allowed_headers   = ["string"]
    allowed_methods   = ["GET"]
    allowed_origins   = ["https://example.com"]
    max_age           = 1
  }
}`,
			},
			{
				Name: "Resource with single destinations attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "single_destinations" {
  account_id = "1234"
  destinations {
    cidr        = "10.5.0.0/24"
    hostname    = "hostname"
    l4_protocol = "tcp"
    port_range  = "80-90"
    type        = "private"
    vnet_id     = "vnet_id"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "single_destinations" {
  account_id = "1234"
  destinations = [
    {
      cidr        = "10.5.0.0/24"
      hostname    = "hostname"
      l4_protocol = "tcp"
      port_range  = "80-90"
      type        = "private"
      vnet_id     = "vnet_id"
    }
  ]
}`,
			},
			{
				Name: "Resource with multiple destinations attributes",
				Input: `resource "cloudflare_zero_trust_access_application" "multiple_destinations" {
  account_id = "1234"
  destinations {
    type = "public"
    uri  = "test.example.com/admin"
  }
  destinations {
    cidr        = "10.5.0.0/24"
    hostname    = "hostname"
    l4_protocol = "tcp"
    port_range  = "80-90"
    type        = "private"
    vnet_id     = "vnet_id"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "multiple_destinations" {
  account_id = "1234"
  destinations = [
    {
      type = "public"
      uri  = "test.example.com/admin"
    },
    {
      cidr        = "10.5.0.0/24"
      hostname    = "hostname"
      l4_protocol = "tcp"
      port_range  = "80-90"
      type        = "private"
      vnet_id     = "vnet_id"
    }
  ]
}`,
			},
			{
				Name: "Resource with footer_links",
				Input: `resource "cloudflare_zero_trust_access_application" "footer_links" {
  account_id = "1234"
  footer_links {
    name = "Privacy Policy"
    url  = "https://example.com/privacy"
  }
  footer_links {
    name = "Terms"
    url  = "https://example.com/terms"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "footer_links" {
  account_id   = "1234"
  footer_links = [
    {
      name = "Privacy Policy"
      url  = "https://example.com/privacy"
    },
    {
      name = "Terms"
      url  = "https://example.com/terms"
    }
  ]
}`,
			},
			{
				Name: "Resource with landing_page_design",
				Input: `resource "cloudflare_zero_trust_access_application" "landing_page_design" {
  account_id = "1234"
  landing_page_design {
    button_color      = "#000000"
    button_text_color = "#000000"
    image_url         = "example.com"
    message           = "message"
    title             = "title"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "landing_page_design" {
  account_id          = "1234"
  landing_page_design = {
    button_color      = "#000000"
    button_text_color = "#000000"
    image_url         = "example.com"
    message           = "message"
    title             = "title"
  }
}`,
			},
			{
				Name: "Resource with saas_app",
				Input: `resource "cloudflare_zero_trust_access_application" "saas_app" {
  account_id = "1234"
  saas_app {
    access_token_lifetime            = "24h"
    allow_pkce_without_client_secret = true
    app_launcher_url                 = "www.example.com"
    auth_type                        = "saml"
    consumer_service_url             = "www.example.com"
    custom_attribute {
      source {
        name = "name"
        name_by_idp = {
          "idp1" = "1234"
          "idp2" = "5678"
        }
      }
      friendly_name = "friendly_name"
      name          = "name"
      name_format   = "name_format"
      required      = true
    }
    custom_claim {
      source {
        name = "name"
        name_by_idp = {
          "idp1" = "1234"
          "idp2" = "5678"
        }
      }
      name     = "name"
      required = true
      scope    = "scope"
    }
    default_relay_state = "default_relay_state"
    grant_types         = ["grant_1", "grant_2"]
    group_filter_regex  = "group_filter_regex"
    hybrid_and_implicit_options {
      return_access_token_from_authorization_endpoint = true
      return_id_token_from_authorization_endpoint     = true
    }
    idp_entity_id                        = "idp_entity_id"
    name_id_transform_jsonata            = "name_id_transform_jsonata"
    redirect_uris                        = ["uri_1", "uri_2"]
    refresh_token_options {
      lifetime = "10m"
    }
    saml_attribute_transform_jsonata = "saml_attribute_transform_jsonata"
    scopes                           = ["scope_1", "scope_2"]
    sp_entity_id                     = "sp_entity_id"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "saas_app" {
  account_id = "1234"
  saas_app = {
    access_token_lifetime            = "24h"
    allow_pkce_without_client_secret = true
    app_launcher_url                 = "www.example.com"
    auth_type                        = "saml"
    consumer_service_url             = "www.example.com"
    default_relay_state              = "default_relay_state"
    grant_types                      = ["grant_1", "grant_2"]
    group_filter_regex               = "group_filter_regex"
    idp_entity_id                    = "idp_entity_id"
    name_id_transform_jsonata        = "name_id_transform_jsonata"
    redirect_uris                    = ["uri_1", "uri_2"]
    saml_attribute_transform_jsonata = "saml_attribute_transform_jsonata"
    scopes                           = ["scope_1", "scope_2"]
    sp_entity_id                     = "sp_entity_id"
    custom_attributes = [
      {
        friendly_name = "friendly_name"
        name          = "name"
        name_format   = "name_format"
        required      = true
        source = {
          name = "name"
          name_by_idp = [
            {
              idp_id      = "idp1"
              source_name = "1234"
            },
            {
              idp_id      = "idp2"
              source_name = "5678"
            }
          ]
        }
      }
    ]
    custom_claims = [
      {
        name     = "name"
        required = true
        scope    = "scope"
        source = {
          name = "name"
          name_by_idp = {
            "idp1" = "1234"
            "idp2" = "5678"
          }
        }
      }
    ]
    hybrid_and_implicit_options = {
      return_access_token_from_authorization_endpoint = true
      return_id_token_from_authorization_endpoint     = true
    }
    refresh_token_options = {
      lifetime = "10m"
    }
  }
}`,
			},
			{
				Name: "Resource with policies",
				Input: `resource "cloudflare_zero_trust_access_application" "policies" {
  account_id = "1234"
  policies   = ["policy-1", "policy-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "policies" {
  account_id = "1234"
  policies = [
    {
      id         = "policy-1"
      precedence = 1
    },
    {
      id         = "policy-2"
      precedence = 2
    }
  ]
}`,
			},
			// TODO scim_config
			{
				Name: "Resource with scim_config",
				Input: `resource "cloudflare_zero_trust_access_application" "scim_config" {
  account_id = "1234"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "scim_config" {
  account_id = "1234"
}`,
			},
			// TODO target_criteria
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

func TestResourceNaming(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("CanHandle both names", func(t *testing.T) {
		if !migrator.CanHandle("cloudflare_access_application") {
			t.Error("Should handle cloudflare_access_application")
		}
		if !migrator.CanHandle("cloudflare_zero_trust_access_application") {
			t.Error("Should handle cloudflare_zero_trust_access_application")
		}
	})

	t.Run("GetResourceType returns v5 name", func(t *testing.T) {
		expected := "cloudflare_zero_trust_access_application"
		if migrator.GetResourceType() != expected {
			t.Errorf("Expected %s, got %s", expected, migrator.GetResourceType())
		}
	})

	t.Run("GetResourceRename returns correct mapping", func(t *testing.T) {
		// GetResourceRename is an optional interface
		type resourceRenamer interface {
			GetResourceRename() (string, string)
		}

		renamer, ok := migrator.(resourceRenamer)
		if !ok {
			t.Fatal("Migrator should implement GetResourceRename")
		}

		old, new := renamer.GetResourceRename()
		if old != "cloudflare_access_application" {
			t.Errorf("Expected old name cloudflare_access_application, got %s", old)
		}
		if new != "cloudflare_zero_trust_access_application" {
			t.Errorf("Expected new name cloudflare_zero_trust_access_application, got %s", new)
		}
	})
}
