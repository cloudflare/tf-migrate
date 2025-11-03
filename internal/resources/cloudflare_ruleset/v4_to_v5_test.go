package cloudflare_ruleset

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic rules transformation from indexed to array",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "zone_id": "zone1",
    "name": "My ruleset",
    "phase": "http_request_firewall_custom",
    "kind": "zone",
    "rules.#": "2",
    "rules.0.id": "rule1",
    "rules.0.expression": "ip.src eq 1.1.1.1",
    "rules.0.action": "block",
    "rules.0.enabled": true,
    "rules.1.id": "rule2",
    "rules.1.expression": "ip.src eq 2.2.2.2",
    "rules.1.action": "skip",
    "rules.1.action_parameters.#": "1",
    "rules.1.action_parameters.0.ruleset": "current"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "zone_id": "zone1",
    "name": "My ruleset",
    "phase": "http_request_firewall_custom",
    "kind": "zone",
    "rules": [
      {
        "id": "rule1",
        "expression": "ip.src eq 1.1.1.1",
        "action": "block",
        "enabled": true
      },
      {
        "id": "rule2",
        "expression": "ip.src eq 2.2.2.2",
        "action": "skip",
        "action_parameters": {
          "ruleset": "current"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "headers transformation from list to map",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "rules.#": "1",
    "rules.0.action": "rewrite",
    "rules.0.expression": "true",
    "rules.0.action_parameters.#": "1",
    "rules.0.action_parameters.0.headers.#": "2",
    "rules.0.action_parameters.0.headers.0.name": "X-Custom-Header",
    "rules.0.action_parameters.0.headers.0.operation": "set",
    "rules.0.action_parameters.0.headers.0.value": "test-value",
    "rules.0.action_parameters.0.headers.1.name": "X-Another-Header",
    "rules.0.action_parameters.0.headers.1.operation": "remove"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "action": "rewrite",
        "expression": "true",
        "action_parameters": {
          "headers": {
            "X-Custom-Header": {
              "operation": "set",
              "value": "test-value"
            },
            "X-Another-Header": {
              "operation": "remove"
            }
          }
        }
      }
    ]
  }
}`,
			},
			{
				Name: "log custom fields transformation",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "rules.#": "1",
    "rules.0.action": "log_custom_field",
    "rules.0.expression": "true",
    "rules.0.action_parameters.#": "1",
    "rules.0.action_parameters.0.cookie_fields.#": "2",
    "rules.0.action_parameters.0.cookie_fields.0": "session_id",
    "rules.0.action_parameters.0.cookie_fields.1": "user_token"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "action": "log_custom_field",
        "expression": "true",
        "action_parameters": {
          "cookie_fields": [
            {"name": "session_id"},
            {"name": "user_token"}
          ]
        }
      }
    ]
  }
}`,
			},
			{
				Name: "ratelimit transformation",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "rules.#": "1",
    "rules.0.action": "block",
    "rules.0.expression": "true",
    "rules.0.ratelimit.#": "1",
    "rules.0.ratelimit.0.characteristics.#": "2",
    "rules.0.ratelimit.0.characteristics.0": "cf.colo.id",
    "rules.0.ratelimit.0.characteristics.1": "ip.src",
    "rules.0.ratelimit.0.period": 60,
    "rules.0.ratelimit.0.requests_per_period": 100
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "action": "block",
        "expression": "true",
        "ratelimit": {
          "characteristics": ["cf.colo.id", "ip.src"],
          "period": 60,
          "requests_per_period": 100
        }
      }
    ]
  }
}`,
			},
			{
				Name: "exposed credential check transformation",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "rules.#": "1",
    "rules.0.action": "log",
    "rules.0.expression": "true",
    "rules.0.exposed_credential_check.#": "1",
    "rules.0.exposed_credential_check.0.username_expression": "http.request.body.form[\"username\"]",
    "rules.0.exposed_credential_check.0.password_expression": "http.request.body.form[\"password\"]"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "action": "log",
        "expression": "true",
        "exposed_credential_check": {
          "username_expression": "http.request.body.form[\"username\"]",
          "password_expression": "http.request.body.form[\"password\"]"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "already in v5 format remains unchanged",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "enabled": true
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "id": "rule1",
        "action": "block",
        "expression": "true",
        "enabled": true
      }
    ]
  }
}`,
			},
			{
				Name: "removes disable_railgun from action_parameters",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "rules.#": "1",
    "rules.0.action": "execute",
    "rules.0.expression": "true",
    "rules.0.action_parameters.#": "1",
    "rules.0.action_parameters.0.id": "test-id",
    "rules.0.action_parameters.0.disable_railgun": true
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "rules": [
      {
        "action": "execute",
        "expression": "true",
        "action_parameters": {
          "id": "test-id"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "handles empty rules",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "zone_id": "zone1"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "zone_id": "zone1"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
