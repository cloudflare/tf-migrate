package ruleset

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestIsWildcardList(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{name: "wildcard", expression: `["*"]`, expected: true},
		{name: "commented wildcard", expression: "[\n  # all query parameters\n  \"*\",\n]", expected: true},
		{name: "wildcard and named parameter", expression: `["*", "session"]`, expected: false},
		{name: "dynamic tuple element", expression: `[var.query_parameter]`, expected: false},
		{name: "typed null tuple element", expression: `[true ? null : "*"]`, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file, diagnostics := hclwrite.ParseConfig([]byte("value = "+test.expression+"\n"), "test.tf", hcl.InitialPos)
			if diagnostics.HasErrors() {
				t.Fatalf("failed to parse test expression: %s", diagnostics.Error())
			}

			tokens := file.Body().GetAttribute("value").Expr().BuildTokens(nil)
			if actual := isWildcardList(tokens); actual != test.expected {
				t.Fatalf("isWildcardList() = %t, want %t", actual, test.expected)
			}
		})
	}
}

func TestV4ToV5Transformation(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal ruleset",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id     = "test123"
  name        = "test-ruleset"
  kind        = "zone"
  phase       = "http_request_firewall_custom"
  description = "Example ruleset"
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id     = "test123"
  name        = "test-ruleset"
  kind        = "zone"
  phase       = "http_request_firewall_custom"
  description = "Example ruleset"
}`,
			},
			{
				Name: "Ruleset with single rule - action_parameters block to attribute",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id     = "test123"
  name        = "test-ruleset"
  kind        = "zone"
  phase       = "http_request_firewall_custom"

  rules {
    action      = "block"
    expression  = "(http.request.uri.path matches \"^/api/\")"
    description = "Block API access"

    action_parameters {
      id = "block_id"
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [
    {
      action      = "block"
      expression  = "(http.request.uri.path matches \"^/api/\")"
      description = "Block API access"
      action_parameters = {
        id = "block_id"
      }
    }
  ]
}`,
			},
			{
				Name: "Ruleset with ratelimit - block to attribute and field rename",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules {
    action      = "block"
    expression  = "(http.request.uri.path matches \"^/api/\")"
    description = "Rate limit API"

    ratelimit {
      characteristics       = ["ip.src"]
      period                = 60
      requests_per_period   = 100
      mitigation_expression = "(cf.zone.name eq \"example.com\")"
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [
    {
      action      = "block"
      expression  = "(http.request.uri.path matches \"^/api/\")"
      description = "Rate limit API"
      ratelimit = {
        characteristics       = ["ip.src"]
        period                = 60
        requests_per_period   = 100
        mitigation_expression = "(cf.zone.name eq \"example.com\")"
      }
    }
  ]
}`,
			},
			{
				Name: "Ruleset with deeply nested uri - 4 levels deep",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewrite_transform_uri"

  rules {
    action      = "rewrite"
    expression  = "true"
    description = "URI rewrite"

    action_parameters {
      uri {
        path {
          value = "/new-path"
        }
        query {
          value = "newparam=value"
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewrite_transform_uri"

  rules = [
    {
      action      = "rewrite"
      expression  = "true"
      description = "URI rewrite"
      action_parameters = {
        uri = {
          path = {
            value = "/new-path"
          }
          query = {
            value = "newparam=value"
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Ruleset with exposed_credential_check",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules {
    action      = "log"
    expression  = "true"
    description = "Check exposed credentials"

    exposed_credential_check {
      username_expression = "http.request.body.form[\"username\"][0]"
      password_expression = "http.request.body.form[\"password\"][0]"
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [
    {
      action      = "log"
      expression  = "true"
      description = "Check exposed credentials"
      exposed_credential_check = {
        username_expression = "http.request.body.form[\"username\"][0]"
        password_expression = "http.request.body.form[\"password\"][0]"
      }
    }
  ]
}`,
			},
			{
				Name: "Ruleset with multiple rules and nested blocks",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules {
    action      = "block"
    expression  = "(http.request.uri.path matches \"^/api/\")"
    description = "First rule"

    action_parameters {
      id = "block_id_1"
    }
  }

  rules {
    action      = "challenge"
    expression  = "(http.request.uri.path matches \"^/admin/\")"
    description = "Second rule"

    ratelimit {
      characteristics     = ["cf.colo.id"]
      period              = 10
      requests_per_period = 5
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [
    {
      action      = "block"
      expression  = "(http.request.uri.path matches \"^/api/\")"
      description = "First rule"
      action_parameters = {
        id = "block_id_1"
      }
    },
    {
      action      = "challenge"
      expression  = "(http.request.uri.path matches \"^/admin/\")"
      description = "Second rule"
      ratelimit = {
        characteristics     = ["cf.colo.id"]
        period              = 10
        requests_per_period = 5
      }
    }
  ]
}`,
			},
			{
				Name: "Dynamic rules block - convert to for expression",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  dynamic "rules" {
    for_each = local.rule_configs
    content {
      action      = rules.value.action
      expression  = rules.value.expression
      description = rules.value.description
      enabled     = true
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [for rules in local.rule_configs : {
    action      = rules.action
    description = rules.description
    enabled     = true
    expression  = rules.expression
  }]
}`,
			},
			{
				Name: "Dynamic rules block with nested action_parameters",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  dynamic "rules" {
    for_each = var.rules
    content {
      action      = rules.value.action
      expression  = rules.value.expression
      description = rules.value.description

      action_parameters {
        id = rules.value.id
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [for rules in var.rules : {
    action      = rules.action
    description = rules.description
    expression  = rules.expression
    action_parameters = {
      id = rules.id
    }
  }]
}`,
			},
			{
				Name: "Dynamic rules with custom iterator name",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  dynamic "rules" {
    for_each = var.firewall_rules
    iterator = rule
    content {
      action      = rule.value.action
      expression  = rule.value.expression
      description = rule.value.desc
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [for rule in var.firewall_rules : {
    action      = rule.action
    description = rule.desc
    expression  = rule.expression
  }]
}`,
			},
			{
				Name: "Cache key query_string blocks - merge include and exclude",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action      = "set_cache_settings"
    expression  = "true"
    description = "Cache with query string"

    action_parameters {
      cache = true

      cache_key {
        cache_by_device_type = true

        custom_key {
          query_string {
            include = ["utm_source", "utm_medium", "page"]
          }

          query_string {
            exclude = ["session", "userid"]
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action      = "set_cache_settings"
      expression  = "true"
      description = "Cache with query string"
      action_parameters = {
        cache = true
        cache_key = {
          cache_by_device_type = true
          custom_key = {
            query_string = {
              include = {
                list = ["utm_source", "utm_medium", "page"]
              }
              exclude = {
                list = ["session", "userid"]
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Cache key query_string with wildcard - convert to all = true",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "true"

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            include = ["*"]
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action     = "set_cache_settings"
      expression = "true"
      action_parameters = {
        cache = true
        cache_key = {
          custom_key = {
            query_string = {
              include = {
                all = true
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Cache key query_string with exclude wildcard - convert to all = true",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "true"

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            exclude = ["*"]
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action     = "set_cache_settings"
      expression = "true"
      action_parameters = {
        cache = true
        cache_key = {
          custom_key = {
            query_string = {
              exclude = {
                all = true
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Cache key query_string with commented exclude wildcard - convert to all = true",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "true"

    action_parameters {
      cache_key {
        custom_key {
          query_string {
            exclude = [
              # Exclude all query parameters.
              "*",
            ]
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action     = "set_cache_settings"
      expression = "true"
      action_parameters = {
        cache_key = {
          custom_key = {
            query_string = {
              exclude = {
                all = true
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Cache key query_string with exclude wildcard and named parameter - preserve list",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "true"

    action_parameters {
      cache_key {
        custom_key {
          query_string {
            exclude = ["*", "session"]
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action     = "set_cache_settings"
      expression = "true"
      action_parameters = {
        cache_key = {
          custom_key = {
            query_string = {
              exclude = {
                list = ["*", "session"]
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Cache key query_string with dynamic exclude list - preserve expression",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "true"

    action_parameters {
      cache_key {
        custom_key {
          query_string {
            exclude = var.excluded_query_parameters
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action     = "set_cache_settings"
      expression = "true"
      action_parameters = {
        cache_key = {
          custom_key = {
            query_string = {
              exclude = {
                list = var.excluded_query_parameters
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "WAF managed ruleset skip with action_parameters.rules comma-separated strings to lists",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "zone"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules {
    action = "skip"
    action_parameters {
      rules = {
        efb7b8c949ac4650a09736fc376e9aee = "0c054d4e4dd5455c9ff8f01efe5abb10,3536f964ccc345308b6445e8dc29b753,e7e4b386797e417c998d872956c390a1"
      }
    }
    description = "Exempt firewall rule expressions from log4j WAF block"
    enabled     = true
    expression  = "(http.request.uri.path contains \"/filters\")"
    logging {
      enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "zone"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules = [
    {
      action      = "skip"
      description = "Exempt firewall rule expressions from log4j WAF block"
      enabled     = true
      expression  = "(http.request.uri.path contains \"/filters\")"
      action_parameters = {
        rules = {
          efb7b8c949ac4650a09736fc376e9aee = ["0c054d4e4dd5455c9ff8f01efe5abb10", "3536f964ccc345308b6445e8dc29b753", "e7e4b386797e417c998d872956c390a1"]
        }
      }
      logging = {
        enabled = true
      }
    }
  ]
}`,
			},
			{
				Name: "WAF managed ruleset skip with single rule ID (no comma) also becomes list",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "zone"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules {
    action = "skip"
    action_parameters {
      rules = {
        efb7b8c949ac4650a09736fc376e9aee = "5f6744fa026a4638bda5b3d7d5e015dd"
      }
    }
    description = "Single rule skip"
    enabled     = true
    expression  = "true"
    logging {
      enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "zone"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules = [
    {
      action      = "skip"
      description = "Single rule skip"
      enabled     = true
      expression  = "true"
      action_parameters = {
        rules = {
          efb7b8c949ac4650a09736fc376e9aee = ["5f6744fa026a4638bda5b3d7d5e015dd"]
        }
      }
      logging = {
        enabled = true
      }
    }
  ]
}`,
			},
			{
				Name: "WAF managed ruleset skip with multiple map entries and mixed comma values",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "zone"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules {
    action = "skip"
    action_parameters {
      rules = {
        "4814384a9e5d4991b9815dcfc25d2f1f" = "6179ae15870a4bb7b2d480d4843b323c"
        "efb7b8c949ac4650a09736fc376e9aee" = "0242110ae62e44028a13bf4834780914,6b1cc72dff9746469d4695a474430f12,55b100786189495c93744db0e1efdffb"
      }
    }
    description = "Multiple map entries"
    enabled     = true
    expression  = "true"
    logging {
      enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "zone"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules = [
    {
      action      = "skip"
      description = "Multiple map entries"
      enabled     = true
      expression  = "true"
      action_parameters = {
        rules = {
          "4814384a9e5d4991b9815dcfc25d2f1f" = ["6179ae15870a4bb7b2d480d4843b323c"]
          "efb7b8c949ac4650a09736fc376e9aee" = ["0242110ae62e44028a13bf4834780914", "6b1cc72dff9746469d4695a474430f12", "55b100786189495c93744db0e1efdffb"]
        }
      }
      logging = {
        enabled = true
      }
    }
  ]
}`,
			},
			{
				Name: "Cache key query_string with only exclude",
				Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules {
    action     = "set_cache_settings"
    expression = "true"

    action_parameters {
      cache = true

      cache_key {
        custom_key {
          query_string {
            exclude = ["session", "token"]
          }
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      action     = "set_cache_settings"
      expression = "true"
      action_parameters = {
        cache = true
        cache_key = {
          custom_key = {
            query_string = {
              exclude = {
                list = ["session", "token"]
              }
            }
          }
        }
      }
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
