package cloudflare_dns_test

import (
	"testing"

	rtesting "github.com/cloudflare/tf-migrate/internal/resources/testing"
)

func TestV4ToV5(t *testing.T) {
	suite := rtesting.TestSuite{
		ResourceType: "cloudflare_record",
		ConfigTests: []rtesting.ConfigTestCase{
			{
				Name: "basic_record_rename",
				Input: `resource "cloudflare_record" "example" {
  name    = "example"
  zone_id = "12345"
  type    = "A"
  value   = "192.0.2.1"
}`,
				Expected: `resource "cloudflare_dns_record" "example" {
  name    = "example"
  zone_id = "12345"
  type    = "A"
  content = "192.0.2.1"
}`,
			},
			{
				Name: "multiple_records",
				Input: `resource "cloudflare_record" "www" {
  name  = "www"
  value = "192.0.2.1"
}

resource "cloudflare_record" "api" {
  name  = "api"
  value = "192.0.2.2"
}`,
				Expected: `resource "cloudflare_dns_record" "www" {
  name    = "www"
  content = "192.0.2.1"
}

resource "cloudflare_dns_record" "api" {
  name    = "api"
  content = "192.0.2.2"
}`,
			},
			{
				Name: "preserves_other_resources",
				Input: `resource "cloudflare_record" "test" {
  value = "test"
}

resource "other_resource" "test" {
  name = "unchanged"
}`,
				Expected: `resource "cloudflare_dns_record" "test" {
  content = "test"
}

resource "other_resource" "test" {
  name = "unchanged"
}`,
			},
		},
		StateTests: []rtesting.StateTestCase{
			{
				Name: "renames_type_in_state",
				Input: `{
  "version": 4,
  "resources": [
    {
      "type": "cloudflare_record",
      "name": "test",
      "instances": [
        {
          "attributes": {
            "id": "123",
            "value": "192.0.2.1"
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
      "type": "cloudflare_dns_record",
      "name": "test",
      "instances": [
        {
          "attributes": {
            "content": "192.0.2.1",
            "id": "123"
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
	// Example of using QuickTest for simple cases
	rtesting.QuickTest(t, "cloudflare_record",
		`resource "cloudflare_record" "test" { value = "1.2.3.4" }`,
		`resource "cloudflare_dns_record" "test" {
  content = "1.2.3.4"
}`)
}

func BenchmarkV4ToV5Transformation(b *testing.B) {
	input := `resource "cloudflare_record" "test" {
  name    = "test"
  zone_id = "12345"
  type    = "A"
  value   = "192.0.2.1"
  ttl     = 120
  proxied = true
}`

	rtesting.BenchmarkTransformation(b, "cloudflare_record", input)
}