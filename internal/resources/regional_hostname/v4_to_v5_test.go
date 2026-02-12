package regional_hostname

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestNewV4ToV5Migrator(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	assert.NotNil(t, migrator)
}

func TestGetResourceType(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	assert.Equal(t, "cloudflare_regional_hostname", migrator.GetResourceType())
}

func TestCanHandle(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "handles regional_hostname",
			resourceType: "cloudflare_regional_hostname",
			expected:     true,
		},
		{
			name:         "does not handle other resources",
			resourceType: "cloudflare_zone",
			expected:     false,
		},
		{
			name:         "does not handle empty string",
			resourceType: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreprocess(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "returns content unchanged",
			input:    "resource \"cloudflare_regional_hostname\" \"test\" {}",
			expected: "resource \"cloudflare_regional_hostname\" \"test\" {}",
		},
		{
			name:     "returns empty string unchanged",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.Preprocess(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetResourceRename(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	oldName, newName := migrator.GetResourceRename()
	assert.Equal(t, "cloudflare_regional_hostname", oldName)
	assert.Equal(t, "cloudflare_regional_hostname", newName)
}

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "removes timeouts block",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

  timeouts {
    create = "30s"
    update = "30s"
  }
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

}`,
			},
			{
				Name: "removes timeouts with only create",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

  timeouts {
    create = "30s"
  }
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

}`,
			},
			{
				Name: "no change when no timeouts",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"
}`,
			},
			{
				Name: "preserves all fields when removing timeouts",
				Input: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "foo.example.com"
  region_key = "ca"

  timeouts {
    create = "30s"
  }
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "foo.example.com"
  region_key = "ca"

}`,
			},
			{
				Name: "multiple regional hostname resources",
				Input: `
resource "cloudflare_regional_hostname" "test1" {
  zone_id    = "abc123"
  hostname   = "example1.com"
  region_key = "us"

  timeouts {
    create = "30s"
  }
}

resource "cloudflare_regional_hostname" "test2" {
  zone_id    = "abc456"
  hostname   = "example2.com"
  region_key = "eu"

  timeouts {
    create = "30s"
    update = "30s"
  }
}`,
				Expected: `
resource "cloudflare_regional_hostname" "test1" {
  zone_id    = "abc123"
  hostname   = "example1.com"
  region_key = "us"

}

resource "cloudflare_regional_hostname" "test2" {
  zone_id    = "abc456"
  hostname   = "example2.com"
  region_key = "eu"

}`,
			},
			{
				Name: "preserves variable references",
				Input: `
variable "zone_id" {
  type = string
}

resource "cloudflare_regional_hostname" "test" {
  zone_id    = var.zone_id
  hostname   = "example.com"
  region_key = "us"

  timeouts {
    create = "30s"
  }
}`,
				Expected: `
variable "zone_id" {
  type = string
}

resource "cloudflare_regional_hostname" "test" {
  zone_id    = var.zone_id
  hostname   = "example.com"
  region_key = "us"

}`,
			},
			{
				Name: "mixed resources - only removes timeouts from regional_hostname",
				Input: `
resource "cloudflare_zone" "test" {
  zone = "example.com"

  timeouts {
    create = "30s"
  }
}

resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

  timeouts {
    create = "30s"
    update = "30s"
  }
}`,
				Expected: `
resource "cloudflare_zone" "test" {
  zone = "example.com"

  timeouts {
    create = "30s"
  }
}

resource "cloudflare_regional_hostname" "test" {
  zone_id    = "abc123"
  hostname   = "example.com"
  region_key = "us"

}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}

func TestV4ToV5TransformationState_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

func TestUsesProviderStateUpgrader(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	if got := migrator.(*V4ToV5Migrator).UsesProviderStateUpgrader(); !got {
		t.Errorf("UsesProviderStateUpgrader() = %v, want true", got)
	}
}
