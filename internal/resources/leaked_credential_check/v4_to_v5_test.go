package leaked_credential_check

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("BasicTransformations", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic resource with enabled=true",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}`,
			},
			{
				Name: "basic resource with enabled=false",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = false
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = false
}`,
			},
		}
		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("VariableReferences", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "zone_id with variable reference",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = var.zone_id
  enabled = true
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = var.zone_id
  enabled = true
}`,
			},
			{
				Name: "enabled with variable reference",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = var.enable_check
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = var.enable_check
}`,
			},
			{
				Name: "zone_id with resource reference",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = cloudflare_zone.example.id
  enabled = true
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = cloudflare_zone.example.id
  enabled = true
}`,
			},
		}
		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("MultipleResources", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "multiple resources in one file",
				Input: `
resource "cloudflare_leaked_credential_check" "zone1" {
  zone_id = "zone-id-1"
  enabled = true
}

resource "cloudflare_leaked_credential_check" "zone2" {
  zone_id = "zone-id-2"
  enabled = false
}

resource "cloudflare_leaked_credential_check" "zone3" {
  zone_id = "zone-id-3"
  enabled = true
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "zone1" {
  zone_id = "zone-id-1"
  enabled = true
}

resource "cloudflare_leaked_credential_check" "zone2" {
  zone_id = "zone-id-2"
  enabled = false
}

resource "cloudflare_leaked_credential_check" "zone3" {
  zone_id = "zone-id-3"
  enabled = true
}`,
			},
		}
		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("MetaArguments", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "resource with lifecycle block",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true

  lifecycle {
    create_before_destroy = true
  }
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
  lifecycle {
    create_before_destroy = true
  }
}`,
			},
			{
				Name: "resource with depends_on",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = cloudflare_zone.example.id
  enabled = true

  depends_on = [cloudflare_zone.example]
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = cloudflare_zone.example.id
  enabled = true
  depends_on = [cloudflare_zone.example]
}`,
			},
			{
				Name: "resource with count",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  count   = 3
  zone_id = var.zone_ids[count.index]
  enabled = true
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  count   = 3
  zone_id = var.zone_ids[count.index]
  enabled = true
}`,
			},
			{
				Name: "resource with for_each",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  for_each = var.zones
  zone_id  = each.value.id
  enabled  = each.value.enabled
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  for_each = var.zones
  zone_id  = each.value.id
  enabled  = each.value.enabled
}`,
			},
		}
		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "resource with comments",
				Input: `
# Main leaked credential check
resource "cloudflare_leaked_credential_check" "test" {
  # Zone identifier
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true # Enable the check
}`,
				Expected: `
# Main leaked credential check
resource "cloudflare_leaked_credential_check" "test" {
  # Zone identifier
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true # Enable the check
}`,
			},
			{
				Name: "resource with conditional expression",
				Input: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = var.environment == "production" ? true : false
}`,
				Expected: `
resource "cloudflare_leaked_credential_check" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = var.environment == "production" ? true : false
}`,
			},
		}
		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
