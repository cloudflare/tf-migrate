package argo

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Smart routing only",
				Input: `
resource "cloudflare_argo" "test" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.test
  to   = cloudflare_argo_smart_routing.test
}
`,
			},
			{
				Name: "Tiered caching only",
				Input: `
resource "cloudflare_argo" "test" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  tiered_caching = "on"
}
`,
				Expected: `
resource "cloudflare_argo_tiered_caching" "test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.test
  to   = cloudflare_argo_tiered_caching.test
}
`,
			},
			{
				Name: "Both attributes - creates both resources with naming conflict resolution",
				Input: `
resource "cloudflare_argo" "both" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing  = "on"
  tiered_caching = "on"
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "both" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.both
  to   = cloudflare_argo_smart_routing.both
}
resource "cloudflare_argo_tiered_caching" "both_tiered" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
`,
			},
			{
				Name: "Neither attribute - defaults to smart_routing with value off",
				Input: `
resource "cloudflare_argo" "default" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "default" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}
moved {
  from = cloudflare_argo.default
  to   = cloudflare_argo_smart_routing.default
}
`,
			},
			{
				Name: "With lifecycle block",
				Input: `
resource "cloudflare_argo" "lifecycle_test" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"

  lifecycle {
    ignore_changes = [smart_routing]
  }
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "lifecycle_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    ignore_changes = [value]
  }
}
moved {
  from = cloudflare_argo.lifecycle_test
  to   = cloudflare_argo_smart_routing.lifecycle_test
}
`,
			},
			{
				Name: "With variable references",
				Input: `
variable "zone_id" {
  type = string
}

variable "enable_smart_routing" {
  type = string
}

resource "cloudflare_argo" "vars" {
  zone_id       = var.zone_id
  smart_routing = var.enable_smart_routing
}
`,
				Expected: `
variable "zone_id" {
  type = string
}

variable "enable_smart_routing" {
  type = string
}

resource "cloudflare_argo_smart_routing" "vars" {
  zone_id = var.zone_id
  value   = var.enable_smart_routing
}
moved {
  from = cloudflare_argo.vars
  to   = cloudflare_argo_smart_routing.vars
}
`,
			},
			{
				Name: "Multiple resources",
				Input: `
resource "cloudflare_argo" "zone1" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "on"
}

resource "cloudflare_argo" "zone2" {
  zone_id        = "28fea702d1075b10ba9c8620b86218ec"
  tiered_caching = "off"
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "zone1" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"
}
moved {
  from = cloudflare_argo.zone1
  to   = cloudflare_argo_smart_routing.zone1
}
resource "cloudflare_argo_tiered_caching" "zone2" {
  zone_id = "28fea702d1075b10ba9c8620b86218ec"
  value   = "off"
}
moved {
  from = cloudflare_argo.zone2
  to   = cloudflare_argo_tiered_caching.zone2
}
`,
			},
			{
				Name: "Smart routing explicitly off",
				Input: `
resource "cloudflare_argo" "explicit_off" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing = "off"
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "explicit_off" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "off"
}
moved {
  from = cloudflare_argo.explicit_off
  to   = cloudflare_argo_smart_routing.explicit_off
}
`,
			},
			{
				Name: "Resource reference for zone_id",
				Input: `
resource "cloudflare_argo" "referenced" {
  zone_id       = cloudflare_zone.main.id
  smart_routing = "on"
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "referenced" {
  zone_id = cloudflare_zone.main.id
  value   = "on"
}
moved {
  from = cloudflare_argo.referenced
  to   = cloudflare_argo_smart_routing.referenced
}
`,
			},
			{
				Name: "Tiered caching with lifecycle",
				Input: `
resource "cloudflare_argo" "tiered_lifecycle" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  tiered_caching = "on"

  lifecycle {
    prevent_destroy = true
  }
}
`,
				Expected: `
resource "cloudflare_argo_tiered_caching" "tiered_lifecycle" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    prevent_destroy = true
  }
}
moved {
  from = cloudflare_argo.tiered_lifecycle
  to   = cloudflare_argo_tiered_caching.tiered_lifecycle
}
`,
			},
			{
				Name: "Both attributes with lifecycle",
				Input: `
resource "cloudflare_argo" "both_lifecycle" {
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  smart_routing  = "on"
  tiered_caching = "on"

  lifecycle {
    ignore_changes = [smart_routing, tiered_caching]
  }
}
`,
				Expected: `
resource "cloudflare_argo_smart_routing" "both_lifecycle" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    ignore_changes = [value]
  }
}
moved {
  from = cloudflare_argo.both_lifecycle
  to   = cloudflare_argo_smart_routing.both_lifecycle
}
resource "cloudflare_argo_tiered_caching" "both_lifecycle_tiered" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  value   = "on"

  lifecycle {
    ignore_changes = [value]
  }
}
`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// State transformation tests removed - state migration is now handled by provider's StateUpgraders
	// The provider's MoveState and UpgradeState interfaces handle all state transformations:
	// - Resource type change (cloudflare_argo → cloudflare_argo_smart_routing/tiered_caching)
	// - Field transformations (smart_routing/tiered_caching → value)
	// - ID transformation (checksum → zone_id)
	// - Computed field initialization (editable, modified_on)
	t.Run("StateTransformation_Removed", func(t *testing.T) {
		t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
	})
}
