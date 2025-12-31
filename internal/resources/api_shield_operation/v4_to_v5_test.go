package api_shield_operation

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic GET operation",
				Input: `
resource "cloudflare_api_shield_operation" "example" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "api.example.com"
  endpoint = "/api/users"
}`,
				Expected: `
resource "cloudflare_api_shield_operation" "example" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "api.example.com"
  endpoint = "/api/users"
}`,
			},
			{
				Name: "POST operation with parameterized endpoint",
				Input: `
resource "cloudflare_api_shield_operation" "create_user" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "POST"
  host     = "api.example.com"
  endpoint = "/api/v1/users/{var1}"
}`,
				Expected: `
resource "cloudflare_api_shield_operation" "create_user" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "POST"
  host     = "api.example.com"
  endpoint = "/api/v1/users/{var1}"
}`,
			},
			{
				Name: "Multiple operations in same config",
				Input: `
resource "cloudflare_api_shield_operation" "get_users" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "api.example.com"
  endpoint = "/api/users"
}

resource "cloudflare_api_shield_operation" "get_user_by_id" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "api.example.com"
  endpoint = "/api/users/{var1}"
}

resource "cloudflare_api_shield_operation" "delete_user" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "DELETE"
  host     = "api.example.com"
  endpoint = "/api/users/{var1}"
}`,
				Expected: `
resource "cloudflare_api_shield_operation" "get_users" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "api.example.com"
  endpoint = "/api/users"
}

resource "cloudflare_api_shield_operation" "get_user_by_id" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "api.example.com"
  endpoint = "/api/users/{var1}"
}

resource "cloudflare_api_shield_operation" "delete_user" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "DELETE"
  host     = "api.example.com"
  endpoint = "/api/users/{var1}"
}`,
			},
			{
				Name: "PATCH operation with multi-param endpoint",
				Input: `
resource "cloudflare_api_shield_operation" "update_post" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "PATCH"
  host     = "api.example.com"
  endpoint = "/api/users/{var1}/posts/{var2}"
}`,
				Expected: `
resource "cloudflare_api_shield_operation" "update_post" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "PATCH"
  host     = "api.example.com"
  endpoint = "/api/users/{var1}/posts/{var2}"
}`,
			},
			{
				Name: "Operation with subdomain host",
				Input: `
resource "cloudflare_api_shield_operation" "v2_endpoint" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "v2.api.example.com"
  endpoint = "/health"
}`,
				Expected: `
resource "cloudflare_api_shield_operation" "v2_endpoint" {
  zone_id  = "023e105f4ecef8ad9ca31a8372d0c353"
  method   = "GET"
  host     = "v2.api.example.com"
  endpoint = "/health"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic operation state",
				Input: `{
  "resources": [{
    "type": "cloudflare_api_shield_operation",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "abc123def456",
        "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
        "method": "GET",
        "host": "api.example.com",
        "endpoint": "/api/users"
      }
    }]
  }]
}`,
				Expected: `{
  "resources": [{
    "type": "cloudflare_api_shield_operation",
    "name": "example",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "abc123def456",
        "operation_id": "abc123def456",
        "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
        "method": "GET",
        "host": "api.example.com",
        "endpoint": "/api/users"
      }
    }]
  }]
}`,
			},
			{
				Name: "Operation with parameterized endpoint",
				Input: `{
  "resources": [{
    "type": "cloudflare_api_shield_operation",
    "name": "parameterized",
    "instances": [{
      "attributes": {
        "id": "xyz789",
        "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
        "method": "POST",
        "host": "api.example.com",
        "endpoint": "/api/users/{var1}/posts/{var2}"
      }
    }]
  }]
}`,
				Expected: `{
  "resources": [{
    "type": "cloudflare_api_shield_operation",
    "name": "parameterized",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "xyz789",
        "operation_id": "xyz789",
        "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
        "method": "POST",
        "host": "api.example.com",
        "endpoint": "/api/users/{var1}/posts/{var2}"
      }
    }]
  }]
}`,
			},
			{
				Name: "Multiple instances",
				Input: `{
  "resources": [{
    "type": "cloudflare_api_shield_operation",
    "name": "multiple",
    "instances": [{
      "attributes": {
        "id": "op1",
        "zone_id": "zone1",
        "method": "GET",
        "host": "api.example.com",
        "endpoint": "/api/v1"
      }
    }, {
      "attributes": {
        "id": "op2",
        "zone_id": "zone1",
        "method": "POST",
        "host": "api.example.com",
        "endpoint": "/api/v1"
      }
    }]
  }]
}`,
				Expected: `{
  "resources": [{
    "type": "cloudflare_api_shield_operation",
    "name": "multiple",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "id": "op1",
        "operation_id": "op1",
        "zone_id": "zone1",
        "method": "GET",
        "host": "api.example.com",
        "endpoint": "/api/v1"
      }
    }, {
      "schema_version": 0,
      "attributes": {
        "id": "op2",
        "operation_id": "op2",
        "zone_id": "zone1",
        "method": "POST",
        "host": "api.example.com",
        "endpoint": "/api/v1"
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
