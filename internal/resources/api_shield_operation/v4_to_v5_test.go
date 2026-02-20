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
}
