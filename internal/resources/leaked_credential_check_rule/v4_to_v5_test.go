package leaked_credential_check_rule

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic resource with all fields",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}`,
				Expected: `resource "cloudflare_leaked_credential_check_rule" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}`,
			},
			{
				Name: "Resource with only zone_id (both optional)",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
				Expected: `resource "cloudflare_leaked_credential_check_rule" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			},
			{
				Name: "Resource with only username",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "username_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
}`,
				Expected: `resource "cloudflare_leaked_credential_check_rule" "username_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
}`,
			},
			{
				Name: "Resource with only password",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "password_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  password = "http.request.body.form.password"
}`,
				Expected: `resource "cloudflare_leaked_credential_check_rule" "password_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  password = "http.request.body.form.password"
}`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "first" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}

resource "cloudflare_leaked_credential_check_rule" "second" {
  zone_id  = "28fea702d1075b10ba9c8620b86218ec"
  username = "http.request.body.user"
}`,
				Expected: `resource "cloudflare_leaked_credential_check_rule" "first" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}

resource "cloudflare_leaked_credential_check_rule" "second" {
  zone_id  = "28fea702d1075b10ba9c8620b86218ec"
  username = "http.request.body.user"
}`,
			},
			{
				Name: "Resource with complex ruleset expressions",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "complex" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "lookup_json_string(http.request.body.raw, \"credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"credentials.password\")"
}`,
				Expected: `resource "cloudflare_leaked_credential_check_rule" "complex" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "lookup_json_string(http.request.body.raw, \"credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"credentials.password\")"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Full state with all fields",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": "http.request.body.form.username",
        "password": "http.request.body.form.password"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": "http.request.body.form.username",
        "password": "http.request.body.form.password"
      }
    }]
  }]
}`,
			},
			{
				Name: "Minimal state with only required fields",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "minimal",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "minimal",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with username only",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "username_only",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": "http.request.body.form.username"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "username_only",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": "http.request.body.form.username"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with password only",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "password_only",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "password": "http.request.body.form.password"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "password_only",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "password": "http.request.body.form.password"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with null optional fields",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "with_nulls",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": null,
        "password": null
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "with_nulls",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": null,
        "password": null
      }
    }]
  }]
}`,
			},
			{
				Name: "Multiple instances of the same resource",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "multi",
    "instances": [
      {
        "attributes": {
          "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
          "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
          "username": "http.request.body.form.username",
          "password": "http.request.body.form.password"
        }
      },
      {
        "attributes": {
          "id": "a284f8a0-bcde-1234-abcd-5b1ed5fc9876",
          "zone_id": "28fea702d1075b10ba9c8620b86218ec",
          "username": "http.request.body.user"
        }
      }
    ]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "multi",
    "instances": [
      {
        "attributes": {
          "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
          "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
          "username": "http.request.body.form.username",
          "password": "http.request.body.form.password"
        }
      },
      {
        "attributes": {
          "id": "a284f8a0-bcde-1234-abcd-5b1ed5fc9876",
          "zone_id": "28fea702d1075b10ba9c8620b86218ec",
          "username": "http.request.body.user"
        }
      }
    ]
  }]
}`,
			},
			{
				Name: "State with empty resources array",
				Input: `{
  "resources": []
}`,
				Expected: `{
  "resources": []
}`,
			},
			{
				Name: "State without instances",
				Input: `{
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "empty",
    "instances": []
  }]
}`,
				Expected: `{
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "empty",
    "instances": []
  }]
}`,
			},
			{
				Name: "Single instance state (for internal testing)",
				Input: `{
  "attributes": {
    "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "username": "http.request.body.form.username",
    "password": "http.request.body.form.password"
  }
}`,
				Expected: `{
  "attributes": {
    "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "username": "http.request.body.form.username",
    "password": "http.request.body.form.password"
  }
}`,
			},
			{
				Name: "State with complex ruleset expressions",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "complex",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": "lookup_json_string(http.request.body.raw, \"credentials.username\")",
        "password": "lookup_json_string(http.request.body.raw, \"credentials.password\")"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_leaked_credential_check_rule",
    "name": "complex",
    "instances": [{
      "attributes": {
        "id": "f174e90a-fafe-4643-bbbc-4a0ed4fc8415",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "username": "lookup_json_string(http.request.body.raw, \"credentials.username\")",
        "password": "lookup_json_string(http.request.body.raw, \"credentials.password\")"
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
