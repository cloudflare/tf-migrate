package ruleset

import (
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

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

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal ruleset state",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "description": "Example ruleset"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "description": "Example ruleset"
  }
}`,
			},
			{
				Name: "action_parameters array to object conversion",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "Test rule",
        "action_parameters": [
          {
            "id": "block_id"
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "Test rule",
        "action_parameters": {
          "id": "block_id"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "ratelimit with field rename and numeric conversions",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "Rate limit rule",
        "ratelimit": [
          {
            "characteristics": ["ip.src"],
            "period": 60,
            "requests_per_period": 100,
            "mitigation_expression": "(cf.zone.name eq \"example.com\")",
            "mitigation_timeout": 600
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "Rate limit rule",
        "ratelimit": {
          "characteristics": "ip.src",
          "period": 60,
          "requests_per_period": 100,
          "mitigation_expression": "(cf.zone.name eq \"example.com\")",
          "mitigation_timeout": 600
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Deeply nested uri with path and query (4 levels)",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewrite_transform_uri",
    "rules": [
      {
        "id": "rule1",
        "action": "rewrite",
        "expression": "true",
        "description": "URI rewrite",
        "action_parameters": [
          {
            "uri": [
              {
                "path": [
                  {
                    "value": "/new-path"
                  }
                ],
                "query": [
                  {
                    "value": "newparam=value"
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewrite_transform_uri",
    "rules": [
      {
        "id": "rule1",
        "action": "rewrite",
        "expression": "true",
        "description": "URI rewrite",
        "action_parameters": {
          "uri": {
            "path": {
              "value": "/new-path"
            },
            "query": {
              "value": "newparam=value"
            }
          }
        }
      }
    ]
  }
}`,
			},
			{
				Name: "exposed_credential_check array to object",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "log",
        "expression": "true",
        "description": "Check credentials",
        "exposed_credential_check": [
          {
            "username_expression": "http.request.body.form[\"username\"][0]",
            "password_expression": "http.request.body.form[\"password\"][0]"
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "log",
        "expression": "true",
        "description": "Check credentials",
        "exposed_credential_check": {
          "username_expression": "http.request.body.form[\"username\"][0]",
          "password_expression": "http.request.body.form[\"password\"][0]"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Empty arrays should be removed",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "Test rule",
        "action_parameters": [],
        "ratelimit": []
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "Test rule",
        "action_parameters": null
      }
    ]
  }
}`,
			},
			{
				Name: "Multiple rules with various transformations",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "First rule",
        "action_parameters": [
          {
            "id": "block_id_1"
          }
        ]
      },
      {
        "id": "rule2",
        "action": "challenge",
        "expression": "true",
        "description": "Second rule",
        "ratelimit": [
          {
            "characteristics": ["cf.colo.id"],
            "period": 10,
            "requests_per_period": 5
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_firewall_custom",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "description": "First rule",
        "action_parameters": {
          "id": "block_id_1"
        }
      },
      {
        "id": "rule2",
        "action": "challenge",
        "expression": "true",
        "description": "Second rule",
        "ratelimit": {
          "characteristics": "cf.colo.id",
          "period": 10,
          "requests_per_period": 5
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Remove disable_railgun from action_parameters",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_late_transform",
    "rules": [
      {
        "id": "rule1",
        "action": "route",
        "expression": "true",
        "description": "Route with origin",
        "action_parameters": [
          {
            "origin": [
              {
                "host": "example.com",
                "port": 443
              }
            ],
            "disable_railgun": true
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_late_transform",
    "rules": [
      {
        "id": "rule1",
        "action": "route",
        "expression": "true",
        "description": "Route with origin",
        "action_parameters": {
          "origin": {
            "host": "example.com",
            "port": 443
          }
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Remove disable_railgun when false",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_late_transform",
    "rules": [
      {
        "id": "rule1",
        "action": "route",
        "expression": "true",
        "description": "Route action",
        "action_parameters": [
          {
            "host_header": "example.com",
            "disable_railgun": false
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_late_transform",
    "rules": [
      {
        "id": "rule1",
        "action": "route",
        "expression": "true",
        "description": "Route action",
        "action_parameters": {
          "host_header": "example.com"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Multiple rules with disable_railgun in some",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_late_transform",
    "rules": [
      {
        "id": "rule1",
        "action": "route",
        "expression": "true",
        "description": "First route",
        "action_parameters": [
          {
            "origin": [
              {
                "host": "example.com"
              }
            ],
            "disable_railgun": true
          }
        ]
      },
      {
        "id": "rule2",
        "action": "rewrite",
        "expression": "true",
        "description": "Rewrite rule",
        "action_parameters": [
          {
            "uri": [
              {
                "path": [
                  {
                    "value": "/new"
                  }
                ]
              }
            ]
          }
        ]
      },
      {
        "id": "rule3",
        "action": "route",
        "expression": "true",
        "description": "Second route",
        "action_parameters": [
          {
            "host_header": "example2.com",
            "disable_railgun": false
          }
        ]
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test123",
    "zone_id": "test123",
    "name": "test-ruleset",
    "kind": "zone",
    "phase": "http_request_late_transform",
    "rules": [
      {
        "id": "rule1",
        "action": "route",
        "expression": "true",
        "description": "First route",
        "action_parameters": {
          "origin": {
            "host": "example.com"
          }
        }
      },
      {
        "id": "rule2",
        "action": "rewrite",
        "expression": "true",
        "description": "Rewrite rule",
        "action_parameters": {
          "uri": {
            "path": {
              "value": "/new"
            }
          }
        }
      },
      {
        "id": "rule3",
        "action": "route",
        "expression": "true",
        "description": "Second route",
        "action_parameters": {
          "host_header": "example2.com"
        }
      }
    ]
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

func TestTransformQueryStringInState(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Transform include array to include.list",
			input: `{
				"attributes": {
					"action_parameters": {
						"cache_key": {
							"custom_key": {
								"query_string": {
									"include": ["param1", "param2"]
								}
							}
						}
					}
				}
			}`,
			expected: `"include": {"list":["param1","param2"]}`,
		},
		{
			name: "Transform wildcard array to include.all",
			input: `{
				"attributes": {
					"action_parameters": {
						"cache_key": {
							"custom_key": {
								"query_string": {
									"include": ["*"]
								}
							}
						}
					}
				}
			}`,
			expected: `"include": {"all":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformQueryStringInState(tt.input, "attributes.action_parameters")
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected output to contain %q\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestTransformCustomLogFields(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		fieldPath     string
		transformFunc func(string, string) string
		expected      string
	}{
		{
			name: "Transform cookie_fields from string array to object array",
			input: `{
				"attributes": {
					"action_parameters": {
						"cookie_fields": ["session_id", "user_token"]
					}
				}
			}`,
			fieldPath:     "attributes.action_parameters",
			transformFunc: transformCookieFields,
			expected:      `{"name":"session_id"}`,
		},
		{
			name: "Transform request_fields from string array to object array",
			input: `{
				"attributes": {
					"action_parameters": {
						"request_fields": ["cf.bot_score", "http.user_agent"]
					}
				}
			}`,
			fieldPath:     "attributes.action_parameters",
			transformFunc: transformRequestFields,
			expected:      `{"name":"cf.bot_score"}`,
		},
		{
			name: "Transform response_fields from string array to object array",
			input: `{
				"attributes": {
					"action_parameters": {
						"response_fields": ["status_code", "content_type"]
					}
				}
			}`,
			fieldPath:     "attributes.action_parameters",
			transformFunc: transformResponseFields,
			expected:      `{"name":"status_code"}`,
		},
		{
			name: "Empty cookie_fields array",
			input: `{
				"attributes": {
					"action_parameters": {
						"cookie_fields": []
					}
				}
			}`,
			fieldPath:     "attributes.action_parameters",
			transformFunc: transformCookieFields,
			expected:      `"cookie_fields": []`,
		},
		{
			name: "Single field in request_fields",
			input: `{
				"attributes": {
					"action_parameters": {
						"request_fields": ["cf.bot_score"]
					}
				}
			}`,
			fieldPath:     "attributes.action_parameters",
			transformFunc: transformRequestFields,
			expected:      `{"name":"cf.bot_score"}`,
		},
		{
			name: "Missing cookie_fields - no transformation",
			input: `{
				"attributes": {
					"action_parameters": {
						"other_field": "value"
					}
				}
			}`,
			fieldPath:     "attributes.action_parameters",
			transformFunc: transformCookieFields,
			expected:      `"other_field": "value"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.transformFunc(tt.input, tt.fieldPath)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected output to contain %q\nGot: %s", tt.expected, result)
			}
		})
	}
}
func TestTransformStatusCodeTTLNumericFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Transform status_code and value to numeric",
			input: `{
				"attributes": {
					"action_parameters": {
						"edge_ttl": {
							"status_code_ttl": [
								{
									"status_code": "200",
									"value": "3600"
								}
							]
						}
					}
				}
			}`,
			expected: `"status_code": 200`,
		},
		{
			name: "Transform status_code_range with from and to",
			input: `{
				"attributes": {
					"action_parameters": {
						"edge_ttl": {
							"status_code_ttl": [
								{
									"status_code_range": {
										"from": "200",
										"to": "299"
									},
									"value": "7200"
								}
							]
						}
					}
				}
			}`,
			expected: `"from": 200`,
		},
		{
			name: "Transform status_code_range from array to object (v4 block format)",
			input: `{
				"attributes": {
					"action_parameters": {
						"edge_ttl": {
							"status_code_ttl": [
								{
									"status_code_range": [
										{
											"from": 200,
											"to": 299
										}
									],
									"value": 3600
								}
							]
						}
					}
				}
			}`,
			expected: `"from":200,"to":299`,
		},
		{
			name: "Transform multiple status_code_ttl entries",
			input: `{
				"attributes": {
					"action_parameters": {
						"edge_ttl": {
							"status_code_ttl": [
								{
									"status_code": "200",
									"value": "3600"
								},
								{
									"status_code": "404",
									"value": "60"
								}
							]
						}
					}
				}
			}`,
			expected: `"status_code": 404`,
		},
		{
			name: "Empty status_code_ttl array - no transformation",
			input: `{
				"attributes": {
					"action_parameters": {
						"edge_ttl": {
							"status_code_ttl": []
						}
					}
				}
			}`,
			expected: `"status_code_ttl": []`,
		},
		{
			name: "Missing edge_ttl - no transformation",
			input: `{
				"attributes": {
					"action_parameters": {
						"cache": true
					}
				}
			}`,
			expected: `"cache": true`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformStatusCodeTTLNumericFields(tt.input, "attributes.action_parameters")
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected output to contain %q\nGot: %s", tt.expected, result)
			}
		})
	}
}
