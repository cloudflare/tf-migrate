package zone_setting

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic_single_setting",
				Input: `resource "cloudflare_zone_settings_override" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    always_online = "on"
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "example_always_online" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "always_online"
  value      = "on"
}`,
			},
			{
				Name: "multiple_simple_settings",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    always_online = "on"
    brotli        = "on"
    ipv6          = "on"
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_always_online" {
  zone_id    = var.zone_id
  setting_id = "always_online"
  value      = "on"
}
resource "cloudflare_zone_setting" "test_brotli" {
  zone_id    = var.zone_id
  setting_id = "brotli"
  value      = "on"
}
resource "cloudflare_zone_setting" "test_ipv6" {
  zone_id    = var.zone_id
  setting_id = "ipv6"
  value      = "on"
}`,
			},
			{
				Name: "integer_settings",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    browser_cache_ttl = 14400
    challenge_ttl     = 1800
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_browser_cache_ttl" {
  zone_id    = var.zone_id
  setting_id = "browser_cache_ttl"
  value      = 14400
}
resource "cloudflare_zone_setting" "test_challenge_ttl" {
  zone_id    = var.zone_id
  setting_id = "challenge_ttl"
  value      = 1800
}`,
			},
			{
				Name: "minify_nested_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    minify {
      css  = "on"
      html = "on"
      js   = "off"
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_minify" {
  zone_id    = var.zone_id
  setting_id = "minify"
  value = {
    css  = "on"
    html = "on"
    js   = "off"
  }
}`,
			},
			{
				Name: "mobile_redirect_nested_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    mobile_redirect {
      mobile_subdomain = "m"
      status           = "on"
      strip_uri        = false
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_mobile_redirect" {
  zone_id    = var.zone_id
  setting_id = "mobile_redirect"
  value = {
    mobile_subdomain = "m"
    status           = "on"
    strip_uri        = false
  }
}`,
			},
			{
				Name: "security_header_with_wrapping",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    security_header {
      enabled            = true
      max_age            = 86400
      include_subdomains = true
      preload            = true
      nosniff            = true
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_security_header" {
  zone_id    = var.zone_id
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = true
      include_subdomains = true
      max_age            = 86400
      nosniff            = true
      preload            = true
    }
  }
}`,
			},
			{
				Name: "nel_nested_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    nel {
      enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_nel" {
  zone_id    = var.zone_id
  setting_id = "nel"
  value = {
    enabled = true
  }
}`,
			},
			{
				Name: "deprecated_setting_filtered",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    always_online = "on"
    universal_ssl = "on"
    brotli        = "on"
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_always_online" {
  zone_id    = var.zone_id
  setting_id = "always_online"
  value      = "on"
}
resource "cloudflare_zone_setting" "test_brotli" {
  zone_id    = var.zone_id
  setting_id = "brotli"
  value      = "on"
}`,
			},
			{
				Name: "zero_rtt_name_mapping",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    zero_rtt = "on"
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_zero_rtt" {
  zone_id    = var.zone_id
  setting_id = "0rtt"
  value      = "on"
}`,
			},
			{
				Name: "comprehensive_config",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    always_online     = "on"
    brotli            = "on"
    browser_cache_ttl = 14400
    cache_level       = "aggressive"

    minify {
      css  = "on"
      html = "on"
      js   = "off"
    }

    security_header {
      enabled            = true
      max_age            = 86400
      include_subdomains = true
      preload            = true
      nosniff            = true
    }
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_always_online" {
  zone_id    = var.zone_id
  setting_id = "always_online"
  value      = "on"
}
resource "cloudflare_zone_setting" "test_brotli" {
  zone_id    = var.zone_id
  setting_id = "brotli"
  value      = "on"
}
resource "cloudflare_zone_setting" "test_browser_cache_ttl" {
  zone_id    = var.zone_id
  setting_id = "browser_cache_ttl"
  value      = 14400
}
resource "cloudflare_zone_setting" "test_cache_level" {
  zone_id    = var.zone_id
  setting_id = "cache_level"
  value      = "aggressive"
}
resource "cloudflare_zone_setting" "test_minify" {
  zone_id    = var.zone_id
  setting_id = "minify"
  value = {
    css  = "on"
    html = "on"
    js   = "off"
  }
}
resource "cloudflare_zone_setting" "test_security_header" {
  zone_id    = var.zone_id
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = true
      include_subdomains = true
      max_age            = 86400
      nosniff            = true
      preload            = true
    }
  }
}`,
			},
			{
				Name: "variable_references_preserved",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
    browser_cache_ttl = var.cache_ttl
    always_online     = var.always_on
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "test_always_online" {
  zone_id    = var.zone_id
  setting_id = "always_online"
  value      = var.always_on
}
resource "cloudflare_zone_setting" "test_browser_cache_ttl" {
  zone_id    = var.zone_id
  setting_id = "browser_cache_ttl"
  value      = var.cache_ttl
}`,
			},
			{
				Name: "empty_settings_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
  }
}`,
				Expected: ``, // No output, all removed
			},
			{
				Name: "no_settings_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id
}`,
				Expected: ``, // No output, all removed
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Note: State transformation deletes the v4 resources (returns empty string)
	// State is recreated via import blocks, which is validated in E2E tests
	// No unit test needed for state deletion
}

func TestMigratorInterface(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("GetResourceType", func(t *testing.T) {
		expected := "cloudflare_zone_setting"
		if got := migrator.GetResourceType(); got != expected {
			t.Errorf("GetResourceType() = %v, want %v", got, expected)
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		tests := []struct {
			name         string
			resourceType string
			want         bool
		}{
			{
				name:         "handles zone_settings_override",
				resourceType: "cloudflare_zone_settings_override",
				want:         true,
			},
			{
				name:         "rejects zone_setting",
				resourceType: "cloudflare_zone_setting",
				want:         false,
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
		input := `resource "cloudflare_zone_settings_override" "test" {
  zone_id = "abc123"
}`
		// Preprocess should return content unchanged
		if got := migrator.Preprocess(input); got != input {
			t.Errorf("Preprocess() modified content, want unchanged")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isDeprecatedSetting", func(t *testing.T) {
		tests := []struct {
			name     string
			setting  string
			expected bool
		}{
			{"universal_ssl is deprecated", "universal_ssl", true},
			{"always_online is not deprecated", "always_online", false},
			{"brotli is not deprecated", "brotli", false},
			{"unknown setting not deprecated", "some_random_setting", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isDeprecatedSetting(tt.setting)
				if result != tt.expected {
					t.Errorf("isDeprecatedSetting(%q) = %v, want %v", tt.setting, result, tt.expected)
				}
			})
		}
	})

	t.Run("mapSettingName", func(t *testing.T) {
		tests := []struct {
			name     string
			v4Name   string
			expected string
		}{
			{"zero_rtt maps to 0rtt", "zero_rtt", "0rtt"},
			{"always_online unchanged", "always_online", "always_online"},
			{"brotli unchanged", "brotli", "brotli"},
			{"unknown setting unchanged", "some_setting", "some_setting"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := mapSettingName(tt.v4Name)
				if result != tt.expected {
					t.Errorf("mapSettingName(%q) = %q, want %q", tt.v4Name, result, tt.expected)
				}
			})
		}
	})
}
