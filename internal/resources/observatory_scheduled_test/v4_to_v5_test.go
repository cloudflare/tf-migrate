package observatory_scheduled_test

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("InterfaceMethods", func(t *testing.T) {
		// Cast to concrete type to test all methods
		m := migrator.(*V4ToV5Migrator)

		// Test GetResourceType
		if got := m.GetResourceType(); got != "cloudflare_observatory_scheduled_test" {
			t.Errorf("GetResourceType() = %v, want %v", got, "cloudflare_observatory_scheduled_test")
		}

		// Test CanHandle
		if !m.CanHandle("cloudflare_observatory_scheduled_test") {
			t.Error("CanHandle('cloudflare_observatory_scheduled_test') should return true")
		}
		if m.CanHandle("cloudflare_other_resource") {
			t.Error("CanHandle('cloudflare_other_resource') should return false")
		}

		// Test Preprocess (no-op)
		input := "test content"
		if got := m.Preprocess(input); got != input {
			t.Errorf("Preprocess() should return input unchanged, got %v", got)
		}

		// Test Postprocess (no-op)
		if got := m.Postprocess(input); got != input {
			t.Errorf("Postprocess() should return input unchanged, got %v", got)
		}
	})

	t.Run("ConfigTransformation", func(t *testing.T) {
		testConfigTransformations(t, migrator)
	})

	// StateTransformation tests removed - TransformState is now a no-op
	// State migration is handled by the provider's StateUpgraders (UpgradeState)
}

func testConfigTransformations(t *testing.T, migrator transform.ResourceTransformer) {
	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Basic observatory test - no changes needed",
			Input: `resource "cloudflare_observatory_scheduled_test" "example" {
  zone_id   = "0da42c8d2132a9ddaf714f9e7c920711"
  url       = "https://example.com"
  region    = "us-central1"
  frequency = "DAILY"
}`,
			Expected: `resource "cloudflare_observatory_scheduled_test" "example" {
  zone_id   = "0da42c8d2132a9ddaf714f9e7c920711"
  url       = "https://example.com"
  region    = "us-central1"
  frequency = "DAILY"
}`,
		},
		{
			Name: "Observatory test with WEEKLY frequency",
			Input: `resource "cloudflare_observatory_scheduled_test" "weekly" {
  zone_id   = "0da42c8d2132a9ddaf714f9e7c920711"
  url       = "https://example.com/test"
  region    = "europe-west1"
  frequency = "WEEKLY"
}`,
			Expected: `resource "cloudflare_observatory_scheduled_test" "weekly" {
  zone_id   = "0da42c8d2132a9ddaf714f9e7c920711"
  url       = "https://example.com/test"
  region    = "europe-west1"
  frequency = "WEEKLY"
}`,
		},
		{
			Name: "Multiple resources",
			Input: `resource "cloudflare_observatory_scheduled_test" "test1" {
  zone_id   = "zone1"
  url       = "https://site1.com"
  region    = "us-east1"
  frequency = "DAILY"
}

resource "cloudflare_observatory_scheduled_test" "test2" {
  zone_id   = "zone2"
  url       = "https://site2.com"
  region    = "asia-east1"
  frequency = "WEEKLY"
}`,
			Expected: `resource "cloudflare_observatory_scheduled_test" "test1" {
  zone_id   = "zone1"
  url       = "https://site1.com"
  region    = "us-east1"
  frequency = "DAILY"
}

resource "cloudflare_observatory_scheduled_test" "test2" {
  zone_id   = "zone2"
  url       = "https://site2.com"
  region    = "asia-east1"
  frequency = "WEEKLY"
}`,
		},
		{
			Name: "URL with trailing slash",
			Input: `resource "cloudflare_observatory_scheduled_test" "trailing" {
  zone_id   = "0da42c8d2132a9ddaf714f9e7c920711"
  url       = "https://example.com/"
  region    = "us-central1"
  frequency = "DAILY"
}`,
			Expected: `resource "cloudflare_observatory_scheduled_test" "trailing" {
  zone_id   = "0da42c8d2132a9ddaf714f9e7c920711"
  url       = "https://example.com/"
  region    = "us-central1"
  frequency = "DAILY"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
