package authenticated_origin_pulls

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "basic zone-wide AOP",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
		{
			Name: "zone-wide AOP disabled",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = false
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = false
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
		{
			Name: "AOP with hostname (removed in v5)",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname = "app.example.com"
  enabled = true
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
		{
			Name: "AOP with certificate (removed in v5)",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  authenticated_origin_pulls_certificate = "cert-id-123"
  enabled = true
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
		{
			Name: "AOP with both hostname and certificate (removed in v5)",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname = "app.example.com"
  authenticated_origin_pulls_certificate = "cert-id-123"
  enabled = true
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
		{
			Name: "multiple AOP resources",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example1" {
  zone_id = "zone-id-1"
  enabled = true
}

resource "cloudflare_authenticated_origin_pulls" "example2" {
  zone_id = "zone-id-2"
  enabled = false
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example1" {
  zone_id = "zone-id-1"
  enabled = true
}
moved {
  from = cloudflare_authenticated_origin_pulls.example1
  to   = cloudflare_authenticated_origin_pulls_settings.example1
}

resource "cloudflare_authenticated_origin_pulls_settings" "example2" {
  zone_id = "zone-id-2"
  enabled = false
}
moved {
  from = cloudflare_authenticated_origin_pulls.example2
  to   = cloudflare_authenticated_origin_pulls_settings.example2
}`,
		},
		{
			Name: "AOP with depends_on",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true

  depends_on = [cloudflare_zone.example]
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true

  depends_on = [cloudflare_zone.example]
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
		{
			Name: "AOP with lifecycle",
			Input: `
resource "cloudflare_authenticated_origin_pulls" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true

  lifecycle {
    prevent_destroy = true
  }
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  enabled = true

  lifecycle {
    prevent_destroy = true
  }
}
moved {
  from = cloudflare_authenticated_origin_pulls.example
  to   = cloudflare_authenticated_origin_pulls_settings.example
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestResourceRename(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Type assert to ResourceRenamer interface
	renamer, ok := migrator.(interface {
		GetResourceRename() (string, string)
	})

	if !ok {
		t.Fatal("Migrator does not implement ResourceRenamer interface")
	}

	oldName, newName := renamer.GetResourceRename()

	if oldName != "cloudflare_authenticated_origin_pulls" {
		t.Errorf("Expected old name 'cloudflare_authenticated_origin_pulls', got '%s'", oldName)
	}

	if newName != "cloudflare_authenticated_origin_pulls_settings" {
		t.Errorf("Expected new name 'cloudflare_authenticated_origin_pulls_settings', got '%s'", newName)
	}
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	if !migrator.CanHandle("cloudflare_authenticated_origin_pulls") {
		t.Error("Expected migrator to handle cloudflare_authenticated_origin_pulls")
	}

	if migrator.CanHandle("cloudflare_some_other_resource") {
		t.Error("Expected migrator to not handle cloudflare_some_other_resource")
	}
}

func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	resourceType := migrator.GetResourceType()

	if resourceType != "cloudflare_authenticated_origin_pulls_settings" {
		t.Errorf("Expected resource type 'cloudflare_authenticated_origin_pulls_settings', got '%s'", resourceType)
	}
}
