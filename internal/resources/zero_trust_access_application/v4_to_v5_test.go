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
}
moved {
  from = cloudflare_access_application.rename
  to   = cloudflare_zero_trust_access_application.rename
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
}
moved {
  from = cloudflare_access_application.basic
  to   = cloudflare_zero_trust_access_application.basic
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
}
moved {
  from = cloudflare_access_application.apps
  to   = cloudflare_zero_trust_access_application.apps
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
}
moved {
  from = cloudflare_access_application.apps
  to   = cloudflare_zero_trust_access_application.apps
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
}
moved {
  from = cloudflare_access_application.app
  to   = cloudflare_zero_trust_access_application.app
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
