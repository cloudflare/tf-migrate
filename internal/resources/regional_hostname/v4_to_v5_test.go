package regional_hostname

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic regional hostname with required fields",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "eu"
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "eu"
}`,
			},
			{
				Name: "regional hostname with routing specified",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "us"
  routing    = "dns"
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "us"
  routing    = "dns"
}`,
			},
			{
				Name: "wildcard hostname",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "*.regional.example.com"
  region_key = "ca"
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "*.regional.example.com"
  region_key = "ca"
}`,
			},
			{
				Name: "multiple regional hostnames",
				Input: `
resource "cloudflare_regional_hostname" "test1" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional1.example.com"
  region_key = "eu"
}

resource "cloudflare_regional_hostname" "test2" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional2.example.com"
  region_key = "us"
  routing    = "dns"
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test1" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional1.example.com"
  region_key = "eu"
}

resource "cloudflare_regional_hostname" "test2" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional2.example.com"
  region_key = "us"
  routing    = "dns"
}`,
			},
			{
				Name: "regional hostname with variable references",
				Input: `
variable "zone_id" {
  type = string
}

resource "cloudflare_regional_hostname" "test" {
  zone_id    = var.zone_id
  hostname   = "regional.example.com"
  region_key = "au"
}`,
				Expected: `
variable "zone_id" {
  type = string
}

resource "cloudflare_regional_hostname" "test" {
  zone_id    = var.zone_id
  hostname   = "regional.example.com"
  region_key = "au"
}`,
			},
			{
				Name: "regional hostname with all optional fields",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "api-regional.example.com"
  region_key = "in"
  routing    = "dns"
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "api-regional.example.com"
  region_key = "in"
  routing    = "dns"
}`,
			},
			{
				Name: "regional hostname with timeouts block - should be removed",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "eu"

  timeouts {
    create = "30m"
    update = "30m"
    delete = "30m"
  }
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "eu"

}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state with required fields",
				Input: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "eu"
					}
				}`,
				Expected: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "eu",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with id and created_on",
				Input: `{
					"attributes": {
						"id": "regional.example.com",
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "eu",
						"created_on": "2023-01-15T10:30:00Z"
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "regional.example.com",
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "eu",
						"created_on": "2023-01-15T10:30:00Z",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with routing already present",
				Input: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "us",
						"routing": "dns"
					}
				}`,
				Expected: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "us",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "wildcard hostname state",
				Input: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "*.regional.example.com",
						"region_key": "ca"
					}
				}`,
				Expected: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "*.regional.example.com",
						"region_key": "ca",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with all v4 fields",
				Input: `{
					"attributes": {
						"id": "api-regional.example.com",
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "api-regional.example.com",
						"region_key": "in",
						"created_on": "2024-03-20T15:45:30Z"
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "api-regional.example.com",
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "api-regional.example.com",
						"region_key": "in",
						"created_on": "2024-03-20T15:45:30Z",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "incomplete state still gets schema_version",
				Input: `{
					"attributes": {
						"hostname": "incomplete.example.com"
					}
				}`,
				Expected: `{
					"attributes": {
						"hostname": "incomplete.example.com",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "empty attributes still gets schema_version",
				Input: `{
					"attributes": {}
				}`,
				Expected: `{
					"attributes": {
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with timeouts - should be removed",
				Input: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "eu",
						"timeouts": {
							"create": "30m",
							"update": "30m",
							"delete": "30m"
						}
					}
				}`,
				Expected: `{
					"attributes": {
						"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
						"hostname": "regional.example.com",
						"region_key": "eu",
						"routing": "dns"
					},
					"schema_version": 0
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
