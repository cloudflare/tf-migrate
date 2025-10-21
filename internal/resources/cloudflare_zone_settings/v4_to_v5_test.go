package cloudflare_zone_settings_test

import (
	"testing"

	rtesting "github.com/cloudflare/tf-migrate/internal/resources/testing"
)

func TestV4ToV5(t *testing.T) {
	suite := rtesting.TestSuite{
		ResourceType: "cloudflare_zone_settings",
		ConfigTests: []rtesting.ConfigTestCase{
			{
				Name: "rename_mobile_redirect",
				Input: `resource "cloudflare_zone_settings" "example" {
  zone_id         = "12345"
  mobile_redirect = "enabled"
}`,
				Expected: `resource "cloudflare_zone_settings" "example" {
  zone_id             = "12345"
  mobile_optimization = "enabled"
}`,
			},
			{
				Name: "rename_waf",
				Input: `resource "cloudflare_zone_settings" "example" {
  zone_id = "12345"
  waf     = "on"
}`,
				Expected: `resource "cloudflare_zone_settings" "example" {
  zone_id                  = "12345"
  web_application_firewall = "on"
}`,
			},
			{
				Name: "rename_ssl",
				Input: `resource "cloudflare_zone_settings" "example" {
  zone_id = "12345"
  ssl     = "flexible"
}`,
				Expected: `resource "cloudflare_zone_settings" "example" {
  zone_id  = "12345"
  ssl_mode = "flexible"
}`,
			},
			{
				Name: "rename_multiple_attributes",
				Input: `resource "cloudflare_zone_settings" "example" {
  zone_id         = "12345"
  mobile_redirect = "enabled"
  waf             = "on"
  ssl             = "flexible"
  other_field     = "unchanged"
}`,
				Expected: `resource "cloudflare_zone_settings" "example" {
  zone_id                  = "12345"
  mobile_optimization      = "enabled"
  web_application_firewall = "on"
  ssl_mode                 = "flexible"
  other_field              = "unchanged"
}`,
			},
		},
		StateTests: []rtesting.StateTestCase{
			{
				Name: "renames_attributes_in_state",
				Input: `{
  "version": 4,
  "resources": [
    {
      "type": "cloudflare_zone_settings",
      "name": "test",
      "instances": [
        {
          "attributes": {
            "id": "123",
            "zone_id": "12345",
            "mobile_redirect": "enabled",
            "waf": "on",
            "ssl": "flexible"
          }
        }
      ]
    }
  ]
}`,
				Expected: `{
  "version": 4,
  "resources": [
    {
      "type": "cloudflare_zone_settings",
      "name": "test",
      "instances": [
        {
          "attributes": {
            "id": "123",
            "zone_id": "12345",
            "mobile_optimization": "enabled",
            "web_application_firewall": "on",
            "ssl_mode": "flexible"
          }
        }
      ]
    }
  ]
}`,
			},
		},
	}

	rtesting.RunTestSuite(t, suite)
}

func TestV4ToV5QuickTest(t *testing.T) {
	// Test quick rename of waf attribute
	rtesting.QuickTest(t, "cloudflare_zone_settings",
		`resource "cloudflare_zone_settings" "test" { waf = "on" }`,
		`resource "cloudflare_zone_settings" "test" { web_application_firewall = "on"
}`)
}

func BenchmarkV4ToV5Transformation(b *testing.B) {
	input := `resource "cloudflare_zone_settings" "test" {
  zone_id         = "12345"
  mobile_redirect = "enabled"
  waf             = "on"
  ssl             = "flexible"
}`

	rtesting.BenchmarkTransformation(b, "cloudflare_zone_settings", input)
}