package url_normalization_settings

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic url_normalization_settings with variable",
				Input: `resource "cloudflare_url_normalization_settings" "example" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "example" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"
}`,
			},
			{
				Name: "with literal zone_id",
				Input: `resource "cloudflare_url_normalization_settings" "example" {
  zone_id = "abc123def456"
  type    = "cloudflare"
  scope   = "incoming"
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "example" {
  zone_id = "abc123def456"
  type    = "cloudflare"
  scope   = "incoming"
}`,
			},
			{
				Name: "all type values - cloudflare and rfc3986",
				Input: `resource "cloudflare_url_normalization_settings" "test1" {
  zone_id = "abc123"
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "test2" {
  zone_id = "def456"
  type    = "rfc3986"
  scope   = "incoming"
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "test1" {
  zone_id = "abc123"
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "test2" {
  zone_id = "def456"
  type    = "rfc3986"
  scope   = "incoming"
}`,
			},
			{
				Name: "all scope values - incoming, both, none",
				Input: `resource "cloudflare_url_normalization_settings" "test1" {
  zone_id = "abc123"
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "test2" {
  zone_id = "def456"
  type    = "cloudflare"
  scope   = "both"
}

resource "cloudflare_url_normalization_settings" "test3" {
  zone_id = "ghi789"
  type    = "cloudflare"
  scope   = "none"
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "test1" {
  zone_id = "abc123"
  type    = "cloudflare"
  scope   = "incoming"
}

resource "cloudflare_url_normalization_settings" "test2" {
  zone_id = "def456"
  type    = "cloudflare"
  scope   = "both"
}

resource "cloudflare_url_normalization_settings" "test3" {
  zone_id = "ghi789"
  type    = "cloudflare"
  scope   = "none"
}`,
			},
			{
				Name: "with for_each reference",
				Input: `resource "cloudflare_url_normalization_settings" "example" {
  for_each = toset(var.zone_ids)
  zone_id  = each.value
  type     = "cloudflare"
  scope    = "incoming"
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "example" {
  for_each = toset(var.zone_ids)
  zone_id  = each.value
  type     = "cloudflare"
  scope    = "incoming"
}`,
			},
			{
				Name: "with count",
				Input: `resource "cloudflare_url_normalization_settings" "example" {
  count   = length(var.zone_ids)
  zone_id = var.zone_ids[count.index]
  type    = "cloudflare"
  scope   = "incoming"
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "example" {
  count   = length(var.zone_ids)
  zone_id = var.zone_ids[count.index]
  type    = "cloudflare"
  scope   = "incoming"
}`,
			},
			{
				Name: "with lifecycle block",
				Input: `resource "cloudflare_url_normalization_settings" "example" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"

  lifecycle {
    prevent_destroy = true
  }
}`,
				Expected: `resource "cloudflare_url_normalization_settings" "example" {
  zone_id = var.cloudflare_zone_id
  type    = "cloudflare"
  scope   = "incoming"

  lifecycle {
    prevent_destroy = true
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state with schema_version - cloudflare type, incoming scope",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "abc123",
    "zone_id": "abc123",
    "type": "cloudflare",
    "scope": "incoming"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "abc123",
    "zone_id": "abc123",
    "type": "cloudflare",
    "scope": "incoming"
  }
}`,
			},
			{
				Name: "rfc3986 type with both scope",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "def456",
    "zone_id": "def456",
    "type": "rfc3986",
    "scope": "both"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "def456",
    "zone_id": "def456",
    "type": "rfc3986",
    "scope": "both"
  }
}`,
			},
			{
				Name: "cloudflare type with none scope",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "ghi789",
    "zone_id": "ghi789",
    "type": "cloudflare",
    "scope": "none"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "ghi789",
    "zone_id": "ghi789",
    "type": "cloudflare",
    "scope": "none"
  }
}`,
			},
			{
				Name: "rfc3986 type with incoming scope",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "jkl012",
    "zone_id": "jkl012",
    "type": "rfc3986",
    "scope": "incoming"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "jkl012",
    "zone_id": "jkl012",
    "type": "rfc3986",
    "scope": "incoming"
  }
}`,
			},
			{
				Name: "state without schema_version field",
				Input: `{
  "attributes": {
    "id": "mno345",
    "zone_id": "mno345",
    "type": "cloudflare",
    "scope": "both"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "mno345",
    "zone_id": "mno345",
    "type": "cloudflare",
    "scope": "both"
  }
}`,
			},
			{
				Name: "invalid instance without attributes",
				Input: `{
  "schema_version": 1
}`,
				Expected: `{
  "schema_version": 0
}`,
			},
			{
				Name: "empty instance",
				Input: `{}`,
				Expected: `{
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

func TestMigratorInterface(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("GetResourceType", func(t *testing.T) {
		if got := migrator.GetResourceType(); got != "cloudflare_url_normalization_settings" {
			t.Errorf("GetResourceType() = %v, want cloudflare_url_normalization_settings", got)
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		tests := []struct {
			name         string
			resourceType string
			want         bool
		}{
			{
				name:         "handles correct resource type",
				resourceType: "cloudflare_url_normalization_settings",
				want:         true,
			},
			{
				name:         "rejects other resource types",
				resourceType: "cloudflare_zone",
				want:         false,
			},
			{
				name:         "rejects empty string",
				resourceType: "",
				want:         false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := migrator.CanHandle(tt.resourceType); got != tt.want {
					t.Errorf("CanHandle(%v) = %v, want %v", tt.resourceType, got, tt.want)
				}
			})
		}
	})

	t.Run("Preprocess", func(t *testing.T) {
		input := `resource "cloudflare_url_normalization_settings" "test" {
  zone_id = "abc123"
  type    = "cloudflare"
  scope   = "incoming"
}`
		// Preprocess should return content unchanged
		if got := migrator.Preprocess(input); got != input {
			t.Errorf("Preprocess() modified content, want unchanged")
		}
	})
}
