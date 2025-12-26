package zero_trust_access_application

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			// Migration V1 Tests
			{
				Name: "transform policies from list of strings to list of objects",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    cloudflare_zero_trust_access_policy.allow.id,
    cloudflare_zero_trust_access_policy.deny.id
  ]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.allow.id
      precedence = 1
    },
    {
      id         = cloudflare_zero_trust_access_policy.deny.id
      precedence = 2
    }
  ]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "transform policies with literal IDs",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = ["policy-id-1", "policy-id-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    {
      id         = "policy-id-1"
      precedence = 1
    },
    {
      id         = "policy-id-2"
      precedence = 2
    }
  ]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "mixed references and literals",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    cloudflare_zero_trust_access_policy.allow.id,
    "literal-policy-id",
    cloudflare_zero_trust_access_policy.deny.id
  ]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.allow.id
      precedence = 1
    },
    {
      id         = "literal-policy-id"
      precedence = 2
    },
    {
      id         = cloudflare_zero_trust_access_policy.deny.id
      precedence = 3
    }
  ]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "handle old resource name references",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    cloudflare_access_policy.old_style.id
  ]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"

  policies = [
    {
      id         = cloudflare_access_policy.old_style.id
      precedence = 1
    }
  ]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "remove domain_type attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id  = "abc123"
  name        = "Test App"
  domain      = "test.example.com"
  type        = "self_hosted"
  domain_type = "public"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "remove domain_type with other attributes preserved",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id    = "abc123"
  name          = "Test App"
  domain        = "test.example.com"
  domain_type   = "public"
  session_duration = "24h"

  cors_headers {
    allow_all_origins = true
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id       = "abc123"
  name             = "Test App"
  domain           = "test.example.com"
  session_duration = "24h"

  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
  cors_headers = {
    allow_all_origins = true
  }
}`,
			},
			{
				Name: "no domain_type to remove",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
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
      uri         = "https://app.example.com"
      description = "Main application"
    }
  ]
}`,
			},
			{
				Name: "no destinations blocks - no change",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "destinations blocks with variable references",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = var.app_uri
  }

  destinations {
    uri = local.db_connection
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"


  destinations = [
    {
      uri = var.app_uri
    },
    {
      uri = local.db_connection
    }
  ]
}`,
			},
			{
				Name: "combined domain_type removal and destinations conversion",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id  = "abc123"
  name        = "Test App"
  type        = "warp"
  domain_type = "public"

  destinations {
    uri = "https://example.com"
  }

  policies = ["policy-id-1", "policy-id-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"


  policies = [
    {
      id         = "policy-id-1"
      precedence = 1
    },
    {
      id         = "policy-id-2"
      precedence = 2
    }
  ]
  destinations = [
    {
      uri = "https://example.com"
    }
  ]
}`,
			},
			{
				Name: "all transformations together with allowed_idps",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id    = "abc123"
  name          = "Test App"
  type          = "warp"
  domain_type   = "public"
  allowed_idps  = toset(["idp-1", "idp-2"])

  destinations {
    uri = "https://example.com"
  }

  destinations {
    uri = "tcp://db.example.com:5432"
  }

  policies = [
    cloudflare_zero_trust_access_policy.allow.id,
    "literal-policy-id"
  ]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id   = "abc123"
  name         = "Test App"
  type         = "warp"
  allowed_idps = ["idp-1", "idp-2"]



  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.allow.id
      precedence = 1
    },
    {
      id         = "literal-policy-id"
      precedence = 2
    }
  ]
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
				Name: "transform toset to list for allowed_idps",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id   = "abc123"
  name         = "Test App"
  domain       = "test.example.com"
  type         = "self_hosted"
  allowed_idps = toset(["idp-1", "idp-2", "idp-3"])
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  allowed_idps               = ["idp-1", "idp-2", "idp-3"]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "handle already list format for allowed_idps",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id   = "abc123"
  name         = "Test App"
  domain       = "test.example.com"
  type         = "self_hosted"
  allowed_idps = ["idp-1", "idp-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  allowed_idps               = ["idp-1", "idp-2"]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "transform toset for custom_pages",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id    = "abc123"
  name          = "Test App"
  domain        = "test.example.com"
  type          = "self_hosted"
  custom_pages  = toset(["page1", "page2"])
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  custom_pages               = ["page1", "page2"]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "transform toset for self_hosted_domains",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id            = "abc123"
  name                  = "Test App"
  domain                = "test.example.com"
  type                  = "self_hosted"
  self_hosted_domains  = toset(["page1", "page2"])
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  self_hosted_domains        = ["page1", "page2"]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "empty policies array",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
  policies   = []
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  policies                   = []
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "destinations with expressions",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = format("https://%s.example.com", var.subdomain)
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {
      uri = format("https://%s.example.com", var.subdomain)
    }
  ]
}`,
			},
			{
				Name: "destinations with conditional expression",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = var.use_ssl ? "https://app.example.com" : "http://app.example.com"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {
      uri = var.use_ssl ? "https://app.example.com" : "http://app.example.com"
    }
  ]
}`,
			},
			{
				Name: "destinations block without uri",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    description = "Test destination"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {
      description = "Test destination"
    }
  ]
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
			{
				Name: "multiple destinations with mixed content",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = "https://app1.example.com"
    description = "Primary app"
  }

  destinations {
  }

  destinations {
    uri = "tcp://db.example.com:3306"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"



  destinations = [
    {
      uri         = "https://app1.example.com"
      description = "Primary app"
    },
    {},
    {
      uri = "tcp://db.example.com:3306"
    }
  ]
}`,
			},
			// New Tests
			{
				Name: "Minimal resource with old name",
				Input: `resource "cloudflare_access_application" "rename" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "rename" {
  account_id                 = "f037e56e89293a057740de681ac9abbe"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "Minimal resource with new name - no change",
				Input: `resource "cloudflare_zero_trust_access_application" "no_rename" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "no_rename" {
  account_id                 = "f037e56e89293a057740de681ac9abbe"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
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
  account_id                   = "1234"
  allow_authenticate_via_warp  = true
  app_launcher_logo_url        = true
  app_launcher_visible         = true
  auto_redirect_to_identity    = true
  bg_color                     = "#000000"
  custom_deny_message          = "message"
  custom_deny_url              = "www.example.com"
  custom_non_identity_deny_url = "www.example.com"
  domain                       = "test.example.com/admin"
  enable_binding_cookie        = true
  header_bg_color              = "#000000"
  http_only_cookie_attribute   = true
  logo_url                     = "www.example.com"
  name                         = "test"
  options_preflight_bypass     = true
  same_site_cookie_attribute   = "strict"
  service_auth_401_redirect    = true
  session_duration             = "24h"
  skip_app_launcher_login_page = true
  skip_interstitial            = true
  type                         = "self_hosted"
  zone_id                      = "1234"
}`,
			},
			{
				Name: "Minimal resource with string set, unchanged fields",
				Input: `resource "cloudflare_zero_trust_access_application" "basic_string_set" {
  account_id          = "1234"
  type                = "self_hosted"
  allowed_idps        = ["1234", "5678"]
  custom_pages        = ["1234", "5678"]
  self_hosted_domains = ["1234", "5678"]
  tags                = ["1234", "5678"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "basic_string_set" {
  account_id                 = "1234"
  type                       = "self_hosted"
  allowed_idps               = ["1234", "5678"]
  custom_pages               = ["1234", "5678"]
  self_hosted_domains        = ["1234", "5678"]
  tags                       = ["1234", "5678"]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "Remove domain_type attribute",
				Input: `resource "cloudflare_zero_trust_access_application" "remove_domain_type" {
  account_id  = "1234"
  type        = "self_hosted"
  domain_type = "public"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "remove_domain_type" {
  account_id                 = "1234"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "Resource with cors_headers",
				Input: `resource "cloudflare_zero_trust_access_application" "cors_headers" {
  account_id = "1234"
  type       = "self_hosted"
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
  account_id                 = "1234"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
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
  type       = "self_hosted"
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
  account_id                 = "1234"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
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
  type       = "self_hosted"
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
  account_id                 = "1234"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
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
  type       = "app_launcher"
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
  account_id = "1234"
  type       = "app_launcher"
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
  type       = "self_hosted"
  landing_page_design {
    button_color      = "#000000"
    button_text_color = "#000000"
    image_url         = "example.com"
    message           = "message"
    title             = "title"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "landing_page_design" {
  account_id                 = "1234"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
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
  type       = "saas"
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
  type       = "saas"
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
  type       = "self_hosted"
  policies   = ["policy-1", "policy-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "policies" {
  account_id = "1234"
  type       = "self_hosted"
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
  http_only_cookie_attribute = "false"
}`,
			},
			// TODO scim_config
			{
				Name: "Resource with scim_config",
				Input: `resource "cloudflare_zero_trust_access_application" "scim_config" {
  account_id = "1234"
  name       = "SCIM App"
  type       = "saas"

  scim_config {
    enabled             = true
    remote_uri          = "https://example.com/scim/v2"
    idp_uid             = "idp-123"
    deactivate_on_delete = true

    authentication {
      scheme          = "oauth2"
      client_id       = "client-123"
      client_secret   = "secret-456"
      authorization_url = "https://auth.example.com/authorize"
      token_url       = "https://auth.example.com/token"
      scopes          = toset(["read", "write"])
    }

    mappings {
      schema             = "urn:ietf:params:scim:schemas:core:2.0:User"
      enabled            = true
      filter             = "userName sw \"test\""
      transform_jsonata  = "$"

      operations {
        create = true
        update = true
        delete = false
      }
    }

    mappings {
      schema    = "urn:ietf:params:scim:schemas:core:2.0:Group"
      enabled   = false
      strictness = "strict"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "scim_config" {
  account_id = "1234"
  name       = "SCIM App"
  type       = "saas"

  scim_config = {
    enabled              = true
    remote_uri           = "https://example.com/scim/v2"
    idp_uid              = "idp-123"
    deactivate_on_delete = true
    authentication = {
      scheme            = "oauth2"
      client_id         = "client-123"
      client_secret     = "secret-456"
      authorization_url = "https://auth.example.com/authorize"
      token_url         = "https://auth.example.com/token"
      scopes            = ["read", "write"]
    }
    mappings = [
      {
        schema            = "urn:ietf:params:scim:schemas:core:2.0:User"
        enabled           = true
        filter            = "userName sw \"test\""
        transform_jsonata = "$"
        operations = {
          create = true
          update = true
          delete = false
        }
      },
      {
        schema     = "urn:ietf:params:scim:schemas:core:2.0:Group"
        enabled    = false
        strictness = "strict"
      }
    ]
  }
}`,
			},
			{
				Name: "Resource with target_criteria",
				Input: `resource "cloudflare_zero_trust_access_application" "target_criteria" {
  account_id = "1234"
  name       = "SSH App"
  type       = "ssh"

  target_criteria {
    port     = 22
    protocol = "SSH"

    target_attributes {
      name   = "hostname"
      values = ["server1.example.com", "server2.example.com"]
    }

    target_attributes {
      name   = "username"
      values = ["admin", "root"]
    }
  }

  target_criteria {
    port     = 3389
    protocol = "RDP"

    target_attributes {
      name   = "hostname"
      values = ["windows-server.example.com"]
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "target_criteria" {
  account_id = "1234"
  name       = "SSH App"
  type       = "ssh"


  http_only_cookie_attribute = "false"
  target_criteria = [
    {
      port     = 22
      protocol = "SSH"
      target_attributes = {
        "hostname" = ["server1.example.com", "server2.example.com"]
        "username" = ["admin", "root"]
      }
    },
    {
      port     = 3389
      protocol = "RDP"
      target_attributes = {
        "hostname" = ["windows-server.example.com"]
      }
    }
  ]
}`,
			},
			// P0/P1 Gap Tests
			{
				Name: "destinations without type field - structural transform only",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "self_hosted"

  destinations {
    uri = "https://example.com"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "self_hosted"

  http_only_cookie_attribute = "false"
  destinations = [
    {
      uri = "https://example.com"
    }
  ]
}`,
			},
			{
				Name: "landing_page_design without title field - structural transform only",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "self_hosted"

  landing_page_design {
    message = "Welcome to our app"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "self_hosted"

  http_only_cookie_attribute = "false"
  landing_page_design = {
    message = "Welcome to our app"
  }
}`,
			},
			{
				Name: "saas_app without auth_type field - structural transform only",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "saas"

  saas_app {
    consumer_service_url = "https://example.com/saml/consume"
    sp_entity_id = "example-entity"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "saas"

  saas_app = {
    consumer_service_url = "https://example.com/saml/consume"
    sp_entity_id         = "example-entity"
  }
}`,
			},
			{
				Name: "cors_headers with max_age - structural transform only",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "self_hosted"

  cors_headers {
    allowed_methods = ["GET", "POST"]
    max_age = 3600
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  type       = "self_hosted"

  http_only_cookie_attribute = "false"
  cors_headers = {
    allowed_methods = ["GET", "POST"]
    max_age         = 3600
  }
}`,
			},
			// Issue #2: Add explicit type when missing but required by type-specific attributes
			{
				Name: "add type=self_hosted when session_duration present but type missing",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id       = "1234"
  name             = "Test App"
  domain           = "test.example.com"
  session_duration = "12h"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "1234"
  name                       = "Test App"
  domain                     = "test.example.com"
  session_duration           = "12h"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "add type=self_hosted when cors_headers present but type missing",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  name       = "Test App"
  domain     = "test.example.com"

  cors_headers {
    allowed_methods = ["GET", "POST"]
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  name       = "Test App"
  domain     = "test.example.com"

  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
  cors_headers = {
    allowed_methods = ["GET", "POST"]
  }
}`,
			},
			{
				Name: "preserve existing type even with type-specific attributes",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id       = "1234"
  name             = "Test App"
  domain           = "test.example.com"
  type             = "ssh"
  session_duration = "8h"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "1234"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "ssh"
  session_duration           = "8h"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "add type=self_hosted when domain present but type missing",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "add type=self_hosted when self_hosted_domains present but type missing",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id          = "abc123"
  name                = "Test App"
  self_hosted_domains = ["app1.example.com", "app2.example.com"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "abc123"
  name                       = "Test App"
  self_hosted_domains        = ["app1.example.com", "app2.example.com"]
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "add type=self_hosted as default when type is not present",
				Input: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "1234"
  name       = "My Application"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "test" {
  account_id                 = "1234"
  name                       = "My Application"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			// Migration V1 Tests
			{
				Name: "transforms_cors_headers_from_array_to_object",
				Input: `{
	"version": 4,
	"terraform_version": "1.12.2",
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"identity_schema_version": 0,
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"cors_headers": [{
					"allowed_methods": ["GET", "POST", "OPTIONS"],
					"allowed_origins": ["https://example.com"],
					"allow_credentials": true,
					"max_age": 600
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"cors_headers": {
							"allow_credentials": true,
							"allowed_methods": [
								"GET",
								"POST",
								"OPTIONS"
							],
							"allowed_origins": [
								"https://example.com"
							],
							"max_age": 600
						},
						"id": "app-id-123",
						"name": "Test App",
						"type": "self_hosted"
					},
					"identity_schema_version": 0,
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"terraform_version": "1.12.2",
	"version": 4
}`,
			},
			{
				Name: "handles_empty_cors_headers_array",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"cors_headers": []
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"cors_headers": null,
						"id": "app-id-123",
						"name": "Test App",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "preserves_cors_headers_when_already_object",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"cors_headers": {
					"allowed_methods": ["GET", "POST"],
					"allowed_origins": ["https://test.com"]
				}
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"cors_headers": {
							"allowed_methods": [
								"GET",
								"POST"
							],
							"allowed_origins": [
								"https://test.com"
							]
						},
						"id": "app-id-123",
						"name": "Test App",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "transforms_landing_page_design_from_array_to_object",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"landing_page_design": [{
					"title": "Welcome",
					"message": "Please sign in",
					"image_url": "https://example.com/logo.png"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"landing_page_design": {
							"image_url": "https://example.com/logo.png",
							"message": "Please sign in",
							"title": "Welcome"
						},
						"name": "Test App",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "handles_empty_landing_page_design_array",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"landing_page_design": []
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"landing_page_design": null,
						"name": "Test App",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "transforms_saas_app_from_array_to_object",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test SAAS App",
				"type": "saas",
				"saas_app": [{
					"consumer_service_url": "https://example.com/sso/saml/consume",
					"sp_entity_id": "example.com",
					"name_id_format": "email",
					"auth_type": "saml"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"name": "Test SAAS App",
						"saas_app": {
							"auth_type": "saml",
							"consumer_service_url": "https://example.com/sso/saml/consume",
							"name_id_format": "email",
							"sp_entity_id": "example.com"
						},
						"type": "saas"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "handles_empty_saas_app_array",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "saas",
				"saas_app": []
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"name": "Test App",
						"saas_app": null,
						"type": "saas"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "transforms_scim_config_from_array_to_object",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "saas",
				"scim_config": [{
					"enabled": true,
					"remote_uri": "https://example.com/scim/v2",
					"idp_uid": "idp-123",
					"deactivate_on_delete": true,
					"authentication": [{
						"scheme": "oauth2",
						"client_id": "client-123",
						"client_secret": "secret-456",
						"authorization_url": "https://auth.example.com/authorize",
						"token_url": "https://auth.example.com/token",
						"scopes": ["read", "write"]
					}],
					"mappings": [
						{
							"schema": "urn:ietf:params:scim:schemas:core:2.0:User",
							"enabled": true,
							"filter": "userName sw \"test\"",
							"transform_jsonata": "$",
							"operations": [{
								"create": true,
								"update": true,
								"delete": true
							}]
						},
						{
							"schema": "urn:ietf:params:scim:schemas:core:2.0:Group",
							"enabled": true,
							"strictness": "strict"
						}
					]
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"name": "Test App",
						"scim_config": {
							"authentication": {
								"authorization_url": "https://auth.example.com/authorize",
								"client_id": "client-123",
								"client_secret": "secret-456",
								"scheme": "oauth2",
								"scopes": [
									"read",
									"write"
								],
								"token_url": "https://auth.example.com/token"
							},
							"deactivate_on_delete": true,
							"enabled": true,
							"idp_uid": "idp-123",
							"mappings": [
								{
									"enabled": true,
									"filter": "userName sw \"test\"",
									"operations": {
										"create": true,
										"delete": true,
										"update": true
									},
									"schema": "urn:ietf:params:scim:schemas:core:2.0:User",
									"transform_jsonata": "$"
								},
								{
									"enabled": true,
									"schema": "urn:ietf:params:scim:schemas:core:2.0:Group",
									"strictness": "strict"
								}
							],
							"remote_uri": "https://example.com/scim/v2"
						},
						"type": "saas"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "handles_empty_scim_config_array",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"scim_config": []
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"name": "Test App",
						"scim_config": null,
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "transforms_empty_hybrid_and_implicit_options_array_to_null",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test SAML App",
				"type": "saas",
				"saas_app": [{
					"consumer_service_url": "https://saml.example.com/sso/saml",
					"sp_entity_id": "saml-app-test",
					"name_id_format": "email",
					"auth_type": "saml",
					"hybrid_and_implicit_options": []
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"name": "Test SAML App",
						"saas_app": {
							"auth_type": "saml",
							"consumer_service_url": "https://saml.example.com/sso/saml",
							"hybrid_and_implicit_options": null,
							"name_id_format": "email",
							"sp_entity_id": "saml-app-test"
						},
						"type": "saas"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "transforms_empty_refresh_token_options_array_to_null",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test OIDC App",
				"type": "saas",
				"saas_app": [{
					"auth_type": "oidc",
					"app_launcher_url": "https://oidc.example.com/launch",
					"grant_types": ["authorization_code"],
					"scopes": ["openid", "email", "profile"],
					"redirect_uris": ["https://oidc.example.com/callback"],
					"refresh_token_options": []
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"id": "app-id-123",
						"name": "Test OIDC App",
						"saas_app": {
							"app_launcher_url": "https://oidc.example.com/launch",
							"auth_type": "oidc",
							"grant_types": [
								"authorization_code"
							],
							"redirect_uris": [
								"https://oidc.example.com/callback"
							],
							"refresh_token_options": null,
							"scopes": [
								"openid",
								"email",
								"profile"
							]
						},
						"type": "saas"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "transforms_multiple_attributes_together",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"name": "Test App",
				"type": "self_hosted",
				"cors_headers": [{
					"allowed_methods": ["GET", "POST"],
					"allow_credentials": true
				}],
				"landing_page_design": [{
					"title": "Welcome",
					"message": "Please sign in"
				}],
				"saas_app": [{
					"consumer_service_url": "https://example.com/callback",
					"sp_entity_id": "example.com"
				}],
				"scim_config": [{
					"enabled": true,
					"remote_uri": "https://example.com/scim"
				}],
				"policies": ["policy-123"],
				"allowed_idps": ["idp-1", "idp-2"],
				"custom_pages": ["page-1"]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"allowed_idps": [
							"idp-1",
							"idp-2"
						],
						"cors_headers": {
							"allow_credentials": true,
							"allowed_methods": [
								"GET",
								"POST"
							]
						},
						"custom_pages": [
							"page-1"
						],
						"id": "app-id-123",
						"landing_page_design": {
							"message": "Please sign in",
							"title": "Welcome"
						},
						"name": "Test App",
						"policies": [
							{
								"id": "policy-123",
								"precedence": 1
							}
						],
						"saas_app": {
							"auth_type": "saml",
							"consumer_service_url": "https://example.com/callback",
							"sp_entity_id": "example.com"
						},
						"scim_config": {
							"enabled": true,
							"remote_uri": "https://example.com/scim"
						},
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			// New Tests - State versions of config tests
			{
				Name: "Minimal resource with basic, unchanged fields",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_access_application",
		"name": "basic",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"allow_authenticate_via_warp": true,
				"app_launcher_logo_url": true,
				"app_launcher_visible": true,
				"auto_redirect_to_identity": true,
				"bg_color": "#000000",
				"custom_deny_message": "message",
				"custom_deny_url": "www.example.com",
				"custom_non_identity_deny_url": "www.example.com",
				"domain": "test.example.com/admin",
				"enable_binding_cookie": true,
				"header_bg_color": "#000000",
				"http_only_cookie_attribute": true,
				"logo_url": "www.example.com",
				"name": "test",
				"options_preflight_bypass": true,
				"same_site_cookie_attribute": "strict",
				"service_auth_401_redirect": true,
				"session_duration": "24h",
				"skip_app_launcher_login_page": true,
				"skip_interstitial": true,
				"type": "self_hosted",
				"zone_id": "1234"
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"allow_authenticate_via_warp": true,
						"app_launcher_logo_url": true,
						"app_launcher_visible": true,
						"auto_redirect_to_identity": true,
						"bg_color": "#000000",
						"custom_deny_message": "message",
						"custom_deny_url": "www.example.com",
						"custom_non_identity_deny_url": "www.example.com",
						"domain": "test.example.com/admin",
						"enable_binding_cookie": true,
						"header_bg_color": "#000000",
						"http_only_cookie_attribute": true,
						"id": "app-id-123",
						"logo_url": "www.example.com",
						"name": "test",
						"options_preflight_bypass": true,
						"same_site_cookie_attribute": "strict",
						"service_auth_401_redirect": true,
						"session_duration": "24h",
						"skip_app_launcher_login_page": true,
						"skip_interstitial": true,
						"type": "self_hosted",
						"zone_id": "1234"
					},
					"schema_version": 0
				}
			],
			"name": "basic",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Minimal resource with string set, unchanged fields",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "basic_string_set",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"allowed_idps": ["1234", "5678"],
				"custom_pages": ["1234", "5678"],
				"self_hosted_domains": ["1234", "5678"],
				"tags": ["1234", "5678"]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"allowed_idps": [
							"1234",
							"5678"
						],
						"custom_pages": [
							"1234",
							"5678"
						],
						"id": "app-id-123",
						"self_hosted_domains": [
							"1234",
							"5678"
						],
						"tags": [
							"1234",
							"5678"
						],
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "basic_string_set",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Remove domain_type attribute",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "remove_domain_type",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"domain_type": "public"
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "remove_domain_type",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with cors_headers",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "cors_headers",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"cors_headers": [{
					"allow_all_headers": true,
					"allow_all_methods": true,
					"allow_all_origins": true,
					"allow_credentials": true,
					"allowed_headers": ["string"],
					"allowed_methods": ["GET"],
					"allowed_origins": ["https://example.com"],
					"max_age": 1
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"cors_headers": {
							"allow_all_headers": true,
							"allow_all_methods": true,
							"allow_all_origins": true,
							"allow_credentials": true,
							"allowed_headers": [
								"string"
							],
							"allowed_methods": [
								"GET"
							],
							"allowed_origins": [
								"https://example.com"
							],
							"max_age": 1
						},
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "cors_headers",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with single destinations attribute",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "single_destinations",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"destinations": [{
					"cidr": "10.5.0.0/24",
					"hostname": "hostname",
					"l4_protocol": "tcp",
					"port_range": "80-90",
					"type": "private",
					"vnet_id": "vnet_id"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"destinations": [
							{
								"cidr": "10.5.0.0/24",
								"hostname": "hostname",
								"l4_protocol": "tcp",
								"port_range": "80-90",
								"type": "private",
								"vnet_id": "vnet_id"
							}
						],
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "single_destinations",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with multiple destinations attributes",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "multiple_destinations",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"destinations": [
					{
						"type": "public",
						"uri": "test.example.com/admin"
					},
					{
						"cidr": "10.5.0.0/24",
						"hostname": "hostname",
						"l4_protocol": "tcp",
						"port_range": "80-90",
						"type": "private",
						"vnet_id": "vnet_id"
					}
				]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"destinations": [
							{
								"type": "public",
								"uri": "test.example.com/admin"
							},
							{
								"cidr": "10.5.0.0/24",
								"hostname": "hostname",
								"l4_protocol": "tcp",
								"port_range": "80-90",
								"type": "private",
								"vnet_id": "vnet_id"
							}
						],
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "multiple_destinations",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with footer_links",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "footer_links",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"footer_links": [
					{
						"name": "Privacy Policy",
						"url": "https://example.com/privacy"
					},
					{
						"name": "Terms",
						"url": "https://example.com/terms"
					}
				]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"footer_links": [
							{
								"name": "Privacy Policy",
								"url": "https://example.com/privacy"
							},
							{
								"name": "Terms",
								"url": "https://example.com/terms"
							}
						],
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "footer_links",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with landing_page_design",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "landing_page_design",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"landing_page_design": [{
					"button_color": "#000000",
					"button_text_color": "#000000",
					"image_url": "example.com",
					"message": "message",
					"title": "title"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"landing_page_design": {
							"button_color": "#000000",
							"button_text_color": "#000000",
							"image_url": "example.com",
							"message": "message",
							"title": "title"
						},
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "landing_page_design",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with saas_app",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "saas_app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"saas_app": [{
					"access_token_lifetime": "24h",
					"allow_pkce_without_client_secret": true,
					"app_launcher_url": "www.example.com",
					"auth_type": "saml",
					"consumer_service_url": "www.example.com",
					"custom_attribute": [{
						"source": [{
							"name": "name",
							"name_by_idp": {
								"idp1": "1234",
								"idp2": "5678"
							}
						}],
						"friendly_name": "friendly_name",
						"name": "name",
						"name_format": "name_format",
						"required": true
					}],
					"custom_claim": [{
						"source": [{
							"name": "name",
							"name_by_idp": {
								"idp1": "1234",
								"idp2": "5678"
							}
						}],
						"name": "name",
						"required": true,
						"scope": "scope"
					}],
					"default_relay_state": "default_relay_state",
					"grant_types": ["grant_1", "grant_2"],
					"group_filter_regex": "group_filter_regex",
					"hybrid_and_implicit_options": [{
						"return_access_token_from_authorization_endpoint": true,
						"return_id_token_from_authorization_endpoint": true
					}],
					"idp_entity_id": "idp_entity_id",
					"name_id_transform_jsonata": "name_id_transform_jsonata",
					"redirect_uris": ["uri_1", "uri_2"],
					"refresh_token_options": [{
						"lifetime": "10m"
					}],
					"saml_attribute_transform_jsonata": "saml_attribute_transform_jsonata",
					"scopes": ["scope_1", "scope_2"],
					"sp_entity_id": "sp_entity_id"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"saas_app": {
							"access_token_lifetime": "24h",
							"allow_pkce_without_client_secret": true,
							"app_launcher_url": "www.example.com",
							"auth_type": "saml",
							"consumer_service_url": "www.example.com",
							"custom_attributes": [
								{
									"friendly_name": "friendly_name",
									"name": "name",
									"name_format": "name_format",
									"required": true,
									"source": {
										"name": "name",
										"name_by_idp": [
											{
												"idp_id": "idp1",
												"source_name": "1234"
											},
											{
												"idp_id": "idp2",
												"source_name": "5678"
											}
										]
									}
								}
							],
							"custom_claims": [
								{
									"name": "name",
									"required": true,
									"scope": "scope",
									"source": {
										"name": "name",
										"name_by_idp": {
											"idp1": "1234",
											"idp2": "5678"
										}
									}
								}
							],
							"default_relay_state": "default_relay_state",
							"grant_types": [
								"grant_1",
								"grant_2"
							],
							"group_filter_regex": "group_filter_regex",
							"hybrid_and_implicit_options": {
								"return_access_token_from_authorization_endpoint": true,
								"return_id_token_from_authorization_endpoint": true
							},
							"idp_entity_id": "idp_entity_id",
							"name_id_transform_jsonata": "name_id_transform_jsonata",
							"redirect_uris": [
								"uri_1",
								"uri_2"
							],
							"refresh_token_options": {
								"lifetime": "10m"
							},
							"saml_attribute_transform_jsonata": "saml_attribute_transform_jsonata",
							"scopes": [
								"scope_1",
								"scope_2"
							],
							"sp_entity_id": "sp_entity_id"
						},
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "saas_app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with policies",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "policies",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"policies": ["policy-1", "policy-2"]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"policies": [
							{
								"id": "policy-1",
								"precedence": 1
							},
							{
								"id": "policy-2",
								"precedence": 2
							}
						],
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "policies",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "Resource with target_criteria",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "target_criteria",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"name": "SSH App",
				"type": "ssh",
				"target_criteria": [
					{
						"port": 22,
						"protocol": "SSH",
						"target_attributes": [
							{
								"name": "hostname",
								"values": ["server1.example.com", "server2.example.com"]
							},
							{
								"name": "username",
								"values": ["admin", "root"]
							}
						]
					},
					{
						"port": 3389,
						"protocol": "RDP",
						"target_attributes": [
							{
								"name": "hostname",
								"values": ["windows-server.example.com"]
							}
						]
					}
				]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"name": "SSH App",
						"target_criteria": [
							{
								"port": 22,
								"protocol": "SSH",
								"target_attributes": {
									"hostname": [
										"server1.example.com",
										"server2.example.com"
									],
									"username": [
										"admin",
										"root"
									]
								}
							},
							{
								"port": 3389,
								"protocol": "RDP",
								"target_attributes": {
									"hostname": [
										"windows-server.example.com"
									]
								}
							}
						],
						"type": "ssh"
					},
					"schema_version": 0
				}
			],
			"name": "target_criteria",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			// P0/P1 Gap Tests
			{
				Name: "destinations without type field - adds default",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "test",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"destinations": [{
					"uri": "https://example.com"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"destinations": [
							{
								"type": "public",
								"uri": "https://example.com"
							}
						],
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "test",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "landing_page_design without title field - adds default",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "test",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"landing_page_design": [{
					"message": "Welcome to our app",
					"button_color": "#000000"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"landing_page_design": {
							"button_color": "#000000",
							"message": "Welcome to our app",
							"title": "Welcome!"
						},
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "test",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "saas_app without auth_type field - adds default",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "test",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"saas_app": [{
					"consumer_service_url": "https://example.com/saml/consume",
					"sp_entity_id": "example-entity"
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"saas_app": {
							"auth_type": "saml",
							"consumer_service_url": "https://example.com/saml/consume",
							"sp_entity_id": "example-entity"
						},
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "test",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "cors_headers with max_age - converts int to float64",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "test",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"cors_headers": [{
					"allowed_methods": ["GET", "POST"],
					"allowed_origins": ["https://example.com"],
					"max_age": 3600
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"cors_headers": {
							"allowed_methods": [
								"GET",
								"POST"
							],
							"allowed_origins": [
								"https://example.com"
							],
							"max_age": 3600
						},
						"id": "app-id-123",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "test",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "adds default type when not present",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "test",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"name": "My Application"
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"name": "My Application",
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "test",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "comprehensive empty values transformation to null test",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "empty_values_test",
		"instances": [{
			"attributes": {
				"id": "test-id",
				"name": "Test Empty Values",
				"type": "self_hosted",
				"domain": "test.example.com",
				"auto_redirect_to_identity": false,
				"enable_binding_cookie": false,
				"http_only_cookie_attribute": false,
				"service_auth_401_redirect": false,
				"skip_interstitial": false,
				"cors_headers": [{
					"allow_all_headers": false,
					"allow_all_methods": false,
					"allow_all_origins": false,
					"allow_credentials": false,
					"max_age": 0,
					"allowed_methods": [],
					"allowed_origins": [],
					"allowed_headers": []
				}],
				"landing_page_design": [{
					"button_color": "",
					"button_text_color": "",
					"image_url": "",
					"message": ""
				}],
				"saas_app": [],
				"scim_config": [{
					"enabled": false,
					"deactivate_on_delete": false,
					"idp_uid": "test-idp",
					"remote_uri": "https://example.com/scim",
					"mappings": [{
						"schema": "urn:ietf:params:scim:schemas:core:2.0:User",
						"enabled": false,
						"filter": "",
						"operations": [{
							"create": false,
							"update": false,
							"delete": false
						}]
					}]
				}],
				"policies": [],
				"allowed_idps": [],
				"custom_pages": []
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"allowed_idps": null,
						"auto_redirect_to_identity": null,
						"cors_headers": null,
						"custom_pages": null,
						"domain": "test.example.com",
						"enable_binding_cookie": null,
						"http_only_cookie_attribute": null,
						"id": "test-id",
						"landing_page_design": {
							"button_color": null,
							"button_text_color": null,
							"image_url": null,
							"message": null,
							"title": "Welcome!"
						},
						"name": "Test Empty Values",
						"policies": null,
						"saas_app": null,
						"scim_config": {
							"deactivate_on_delete": null,
							"enabled": null,
							"idp_uid": "test-idp",
							"mappings": [
								{
									"enabled": null,
									"filter": null,
									"operations": {
										"create": null,
										"delete": null,
										"update": null
									},
									"schema": "urn:ietf:params:scim:schemas:core:2.0:User"
								}
							],
							"remote_uri": "https://example.com/scim"
						},
						"service_auth_401_redirect": null,
						"skip_interstitial": null,
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "empty_values_test",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "large policies array with 10+ items",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"policies": ["p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8", "p9", "p10", "p11", "p12"]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"policies": [
							{
								"id": "p1",
								"precedence": 1
							},
							{
								"id": "p2",
								"precedence": 2
							},
							{
								"id": "p3",
								"precedence": 3
							},
							{
								"id": "p4",
								"precedence": 4
							},
							{
								"id": "p5",
								"precedence": 5
							},
							{
								"id": "p6",
								"precedence": 6
							},
							{
								"id": "p7",
								"precedence": 7
							},
							{
								"id": "p8",
								"precedence": 8
							},
							{
								"id": "p9",
								"precedence": 9
							},
							{
								"id": "p10",
								"precedence": 10
							},
							{
								"id": "p11",
								"precedence": 11
							},
							{
								"id": "p12",
								"precedence": 12
							}
						],
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "target_criteria with empty target_attributes - keeps empty array",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"target_criteria": [{
					"port": 22,
					"protocol": "SSH",
					"target_attributes": []
				}]
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"id": "app-id-123",
						"target_criteria": [
							{
								"port": 22,
								"protocol": "SSH",
								"target_attributes": []
							}
						],
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
			{
				Name: "null values in optional fields preserved",
				Input: `{
	"version": 4,
	"resources": [{
		"type": "cloudflare_zero_trust_access_application",
		"name": "app",
		"instances": [{
			"attributes": {
				"id": "app-id-123",
				"account_id": "1234",
				"type": "self_hosted",
				"name": "Test",
				"session_duration": null,
				"custom_deny_url": null,
				"cors_headers": null
			}
		}]
	}]
}`,
				Expected: `{
	"resources": [
		{
			"instances": [
				{
					"attributes": {
						"account_id": "1234",
						"cors_headers": null,
						"custom_deny_url": null,
						"id": "app-id-123",
						"name": "Test",
						"session_duration": null,
						"type": "self_hosted"
					},
					"schema_version": 0
				}
			],
			"name": "app",
			"type": "cloudflare_zero_trust_access_application"
		}
	],
	"version": 4
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	t.Run("ConfigEdgeCases", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "resource with count meta-argument",
				Input: `resource "cloudflare_access_application" "apps" {
  count      = 3
  account_id = "abc123"
  name       = "App ${count.index}"
  domain     = "app-${count.index}.example.com"
  type       = "self_hosted"

  policies = ["policy-${count.index}"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "apps" {
  count      = 3
  account_id = "abc123"
  name       = "App ${count.index}"
  domain     = "app-${count.index}.example.com"
  type       = "self_hosted"

  policies = [
    {
      id         = "policy-${count.index}"
      precedence = 1
    }
  ]
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "resource with for_each meta-argument - preserves meta-args",
				Input: `resource "cloudflare_access_application" "apps" {
  for_each   = var.applications
  account_id = "abc123"
  name       = each.value.name
  domain     = each.value.domain
  type       = "self_hosted"

  policies = ["policy-1", "policy-2"]
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "apps" {
  for_each   = var.applications
  account_id = "abc123"
  name       = each.value.name
  domain     = each.value.domain
  type       = "self_hosted"

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
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "dynamic block preserved",
				Input: `resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  dynamic "destinations" {
    for_each = var.destinations
    content {
      uri = destinations.value.uri
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  dynamic "destinations" {
    for_each = var.destinations
    content {
      uri = destinations.value.uri
    }
  }
}`,
			},
			{
				Name: "conditional resource creation with count",
				Input: `resource "cloudflare_access_application" "app" {
  count      = var.create_app ? 1 : 0
  account_id = "abc123"
  name       = "Test App"
  domain     = "test.example.com"
  type       = "self_hosted"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "app" {
  count                      = var.create_app ? 1 : 0
  account_id                 = "abc123"
  name                       = "Test App"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "strings with special characters",
				Input: `resource "cloudflare_zero_trust_access_application" "app" {
  account_id          = "abc123"
  name                = "App with \"quotes\" and 'apostrophes'"
  custom_deny_message = "Access denied: contact admin@example.com\nFor help: https://help.example.com"
  domain              = "test.example.com"
  type                = "self_hosted"
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "app" {
  account_id                 = "abc123"
  name                       = "App with \"quotes\" and 'apostrophes'"
  custom_deny_message        = "Access denied: contact admin@example.com\nFor help: https://help.example.com"
  domain                     = "test.example.com"
  type                       = "self_hosted"
  http_only_cookie_attribute = "false"
}`,
			},
			{
				Name: "large destinations array with 10+ items",
				Input: `resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations {
    uri = "https://app1.example.com"
  }
  destinations {
    uri = "https://app2.example.com"
  }
  destinations {
    uri = "https://app3.example.com"
  }
  destinations {
    uri = "https://app4.example.com"
  }
  destinations {
    uri = "https://app5.example.com"
  }
  destinations {
    uri = "https://app6.example.com"
  }
  destinations {
    uri = "https://app7.example.com"
  }
  destinations {
    uri = "https://app8.example.com"
  }
  destinations {
    uri = "https://app9.example.com"
  }
  destinations {
    uri = "https://app10.example.com"
  }
  destinations {
    uri = "https://app11.example.com"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "abc123"
  name       = "Test App"
  type       = "warp"

  destinations = [
    {
      uri = "https://app1.example.com"
    },
    {
      uri = "https://app2.example.com"
    },
    {
      uri = "https://app3.example.com"
    },
    {
      uri = "https://app4.example.com"
    },
    {
      uri = "https://app5.example.com"
    },
    {
      uri = "https://app6.example.com"
    },
    {
      uri = "https://app7.example.com"
    },
    {
      uri = "https://app8.example.com"
    },
    {
      uri = "https://app9.example.com"
    },
    {
      uri = "https://app10.example.com"
    },
    {
      uri = "https://app11.example.com"
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
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
