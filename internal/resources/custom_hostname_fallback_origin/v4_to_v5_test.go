package custom_hostname_fallback_origin

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic resource - no changes needed",
				Input: `
resource "cloudflare_custom_hostname_fallback_origin" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  origin  = "fallback.example.com"
}`,
				Expected: `
resource "cloudflare_custom_hostname_fallback_origin" "example" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  origin  = "fallback.example.com"
}`,
			},
			{
				Name: "multiple resources",
				Input: `
resource "cloudflare_custom_hostname_fallback_origin" "example1" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  origin  = "fallback1.example.com"
}

resource "cloudflare_custom_hostname_fallback_origin" "example2" {
  zone_id = "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
  origin  = "fallback2.example.com"
}`,
				Expected: `
resource "cloudflare_custom_hostname_fallback_origin" "example1" {
  zone_id = "d41d8cd98f00b204e9800998ecf8427e"
  origin  = "fallback1.example.com"
}

resource "cloudflare_custom_hostname_fallback_origin" "example2" {
  zone_id = "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
  origin  = "fallback2.example.com"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		// NOTE: TransformState is a NO-OP since the provider handles all state migration
		// via StateUpgraders. These tests verify that state passes through unchanged.
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state - passes through unchanged",
				Input: `{
  "schema_version": 500,
  "attributes": {
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "origin": "fallback.example.com",
    "status": "active"
  }
}`,
				Expected: `{
  "schema_version": 500,
  "attributes": {
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "origin": "fallback.example.com",
    "status": "active"
  }
}`,
			},
			{
				Name: "state with different status - passes through unchanged",
				Input: `{
  "schema_version": 500,
  "attributes": {
    "zone_id": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "origin": "backup.example.com",
    "status": "pending_deployment"
  }
}`,
				Expected: `{
  "schema_version": 500,
  "attributes": {
    "zone_id": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "origin": "backup.example.com",
    "status": "pending_deployment"
  }
}`,
			},
			{
				Name: "minimal state - passes through unchanged",
				Input: `{
  "schema_version": 500,
  "attributes": {
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "origin": "fallback.example.com"
  }
}`,
				Expected: `{
  "schema_version": 500,
  "attributes": {
    "zone_id": "d41d8cd98f00b204e9800998ecf8427e",
    "origin": "fallback.example.com"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

func TestMigratorInterface(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("GetResourceType", func(t *testing.T) {
		expected := "cloudflare_custom_hostname_fallback_origin"
		if got := migrator.GetResourceType(); got != expected {
			t.Errorf("GetResourceType() = %v, want %v", got, expected)
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		if !migrator.CanHandle("cloudflare_custom_hostname_fallback_origin") {
			t.Error("CanHandle() should return true for cloudflare_custom_hostname_fallback_origin")
		}

		if migrator.CanHandle("cloudflare_other_resource") {
			t.Error("CanHandle() should return false for other resources")
		}
	})

}
