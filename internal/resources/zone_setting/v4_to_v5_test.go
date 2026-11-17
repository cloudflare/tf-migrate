package zone_setting

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
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
}
import {
  to = cloudflare_zone_setting.example_always_online
  id = "0da42c8d2132a9ddaf714f9e7c920711/always_online"
}
removed {
  from = cloudflare_zone_settings_override.example
  lifecycle {
    destroy = false
  }
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
import {
  to = cloudflare_zone_setting.test_always_online
  id = "${var.zone_id}/always_online"
}
resource "cloudflare_zone_setting" "test_brotli" {
  zone_id    = var.zone_id
  setting_id = "brotli"
  value      = "on"
}
import {
  to = cloudflare_zone_setting.test_brotli
  id = "${var.zone_id}/brotli"
}
resource "cloudflare_zone_setting" "test_ipv6" {
  zone_id    = var.zone_id
  setting_id = "ipv6"
  value      = "on"
}
import {
  to = cloudflare_zone_setting.test_ipv6
  id = "${var.zone_id}/ipv6"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
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
import {
  to = cloudflare_zone_setting.test_browser_cache_ttl
  id = "${var.zone_id}/browser_cache_ttl"
}
resource "cloudflare_zone_setting" "test_challenge_ttl" {
  zone_id    = var.zone_id
  setting_id = "challenge_ttl"
  value      = 1800
}
import {
  to = cloudflare_zone_setting.test_challenge_ttl
  id = "${var.zone_id}/challenge_ttl"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
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
}
import {
  to = cloudflare_zone_setting.test_minify
  id = "${var.zone_id}/minify"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
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
}
import {
  to = cloudflare_zone_setting.test_mobile_redirect
  id = "${var.zone_id}/mobile_redirect"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
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
}
import {
  to = cloudflare_zone_setting.test_security_header
  id = "${var.zone_id}/security_header"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
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
}
import {
  to = cloudflare_zone_setting.test_nel
  id = "${var.zone_id}/nel"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
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
import {
  to = cloudflare_zone_setting.test_always_online
  id = "${var.zone_id}/always_online"
}
resource "cloudflare_zone_setting" "test_brotli" {
  zone_id    = var.zone_id
  setting_id = "brotli"
  value      = "on"
}
import {
  to = cloudflare_zone_setting.test_brotli
  id = "${var.zone_id}/brotli"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
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
}
import {
  to = cloudflare_zone_setting.test_zero_rtt
  id = "${var.zone_id}/0rtt"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
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
import {
  to = cloudflare_zone_setting.test_always_online
  id = "${var.zone_id}/always_online"
}
resource "cloudflare_zone_setting" "test_brotli" {
  zone_id    = var.zone_id
  setting_id = "brotli"
  value      = "on"
}
import {
  to = cloudflare_zone_setting.test_brotli
  id = "${var.zone_id}/brotli"
}
resource "cloudflare_zone_setting" "test_browser_cache_ttl" {
  zone_id    = var.zone_id
  setting_id = "browser_cache_ttl"
  value      = 14400
}
import {
  to = cloudflare_zone_setting.test_browser_cache_ttl
  id = "${var.zone_id}/browser_cache_ttl"
}
resource "cloudflare_zone_setting" "test_cache_level" {
  zone_id    = var.zone_id
  setting_id = "cache_level"
  value      = "aggressive"
}
import {
  to = cloudflare_zone_setting.test_cache_level
  id = "${var.zone_id}/cache_level"
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
import {
  to = cloudflare_zone_setting.test_minify
  id = "${var.zone_id}/minify"
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
}
import {
  to = cloudflare_zone_setting.test_security_header
  id = "${var.zone_id}/security_header"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
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
import {
  to = cloudflare_zone_setting.test_always_online
  id = "${var.zone_id}/always_online"
}
resource "cloudflare_zone_setting" "test_browser_cache_ttl" {
  zone_id    = var.zone_id
  setting_id = "browser_cache_ttl"
  value      = var.cache_ttl
}
import {
  to = cloudflare_zone_setting.test_browser_cache_ttl
  id = "${var.zone_id}/browser_cache_ttl"
}
removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
}`,
			},
			{
				Name: "empty_settings_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id

  settings {
  }
}`,
				Expected: `removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
}`,
			},
			{
				Name: "no_settings_block",
				Input: `resource "cloudflare_zone_settings_override" "test" {
  zone_id = var.zone_id
}`,
				Expected: `removed {
  from = cloudflare_zone_settings_override.test
  lifecycle {
    destroy = false
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("CountBasedResource", func(t *testing.T) {
		// Resources with count do not get import blocks: Terraform does not support
		// count on import blocks, and the resource may not exist when count = 0.
		// The removed block is still emitted so the old state entry is cleaned up.
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "count_no_import_block",
				Input: `resource "cloudflare_zone_settings_override" "conditional" {
  count   = local.enabled ? 1 : 0
  zone_id = var.zone_id

  settings {
    http3 = "on"
  }
}`,
				Expected: `resource "cloudflare_zone_setting" "conditional_http3" {
  count      = local.enabled ? 1 : 0
  zone_id    = var.zone_id
  setting_id = "http3"
  value      = "on"
}
removed {
  from = cloudflare_zone_settings_override.conditional
  lifecycle {
    destroy = false
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
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

func TestTransformPhaseOne(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	tests := []struct {
		name            string
		input           string
		wantRemovedFrom string
	}{
		{
			name: "resource block is replaced by removed block",
			input: `resource "cloudflare_zone_settings_override" "example" {
  zone_id = "abc123"

  settings {
    always_online = "on"
  }
}`,
			wantRemovedFrom: "cloudflare_zone_settings_override.example",
		},
		{
			name: "count resource uses bare address without instance key",
			input: `resource "cloudflare_zone_settings_override" "conditional" {
  count   = var.enabled ? 1 : 0
  zone_id = "abc123"

  settings {
    tls_1_3 = "on"
  }
}`,
			// removed {} does not support instance keys ([0]) — Terraform requires
			// a bare resource address; the block applies to all instances.
			wantRemovedFrom: "cloudflare_zone_settings_override.conditional",
		},
		{
			name: "resource with no settings block still gets removed block",
			input: `resource "cloudflare_zone_settings_override" "empty" {
  zone_id = "abc123"
}`,
			wantRemovedFrom: "cloudflare_zone_settings_override.empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclwrite.ParseConfig([]byte(tt.input), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "parse error: %v", diags)

			ctx := &transform.Context{
				Content:       []byte(tt.input),
				Filename:      "test.tf",
				Diagnostics:   make(hcl.Diagnostics, 0),
				Metadata:      make(map[string]interface{}),
				SourceVersion: "v4",
				TargetVersion: "v5",
			}

			var block *hclwrite.Block
			for _, b := range file.Body().Blocks() {
				if b.Type() == "resource" {
					block = b
					break
				}
			}
			require.NotNil(t, block)

			result, err := migrator.TransformPhaseOne(ctx, block)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Phase 1 must NOT remove the original block — removed {} blocks go
			// to a separate _phase1_cleanup.tf, originals stay intact for phase 2.
			assert.False(t, result.RemoveOriginal, "TransformPhaseOne must not remove the original resource block")

			// Must produce exactly one block (the removed {} block)
			require.Len(t, result.Blocks, 1)
			removedBlock := result.Blocks[0]
			assert.Equal(t, "removed", removedBlock.Type())

			// Serialize the removed block via a scratch file for accurate formatting
			scratch := hclwrite.NewEmptyFile()
			scratch.Body().AppendBlock(removedBlock)
			output := string(hclwrite.Format(scratch.Bytes()))

			// The removed block must have the correct from address
			assert.True(t, strings.Contains(output, tt.wantRemovedFrom),
				"removed block should reference %q, got:\n%s", tt.wantRemovedFrom, output)

			// The removed block must have destroy = false lifecycle
			assert.True(t, strings.Contains(output, "destroy = false"),
				"removed block should contain destroy = false lifecycle, got:\n%s", output)
		})
	}
}

// runTransformConfig transforms the single resource in input and returns the
// formatted generated blocks keyed by resource name, the full output and the
// diagnostics.
func runTransformConfig(t *testing.T, input string) (map[string]string, string, hcl.Diagnostics) {
	t.Helper()
	migrator := &V4ToV5Migrator{}

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)

	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "test.tf",
		Diagnostics:   make(hcl.Diagnostics, 0),
		Metadata:      make(map[string]interface{}),
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	var block *hclwrite.Block
	for _, b := range file.Body().Blocks() {
		if b.Type() == "resource" {
			block = b
			break
		}
	}
	require.NotNil(t, block)

	result, err := migrator.TransformConfig(ctx, block)
	require.NoError(t, err)
	require.NotNil(t, result)

	resources := map[string]string{}
	var out string
	for _, b := range result.Blocks {
		text := string(hclwrite.Format(b.BuildTokens(nil).Bytes()))
		out += text
		if b.Type() == "resource" {
			resources[b.Labels()[1]] = text
		}
	}
	return resources, out, ctx.Diagnostics
}

// The rest of the lifecycle block must survive the split into one resource
// per setting. Dropping the whole block would quietly remove prevent_destroy,
// turning a protected resource into a destroyable one with no warning.
func TestLifecyclePreservedAlongsideIgnoreChanges(t *testing.T) {
	input := `resource "cloudflare_zone_settings_override" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    always_use_https = "on"
  }

  lifecycle {
    prevent_destroy       = true
    create_before_destroy = true
    ignore_changes        = [settings[0].always_use_https]
  }
}`

	resources, out, diags := runTransformConfig(t, input)

	assert.Contains(t, out, "prevent_destroy",
		"prevent_destroy must survive the migration, got:\n%s", out)
	assert.Contains(t, out, "create_before_destroy",
		"create_before_destroy must survive the migration, got:\n%s", out)
	assert.Equal(t, `resource "cloudflare_zone_setting" "example_always_use_https" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "always_use_https"
  value      = "on"
  lifecycle {
    create_before_destroy = true
    ignore_changes        = [value]
    prevent_destroy       = true
  }
}
`, resources["example_always_use_https"])
	assert.Empty(t, diags)
}

// v4 ignore_changes entries point at settings[0].<setting> paths. In v5 each
// setting is its own resource with a single value attribute, so each entry
// maps to ignore_changes = [value] on the matching generated resource.
func TestIgnoreChangesMapping(t *testing.T) {
	const settings = `  zone_id = var.zone_id

  settings {
    brotli   = "on"
    polish   = var.polish
    zero_rtt = "on"
    universal_ssl = "on"

    security_header {
      enabled = true
      max_age = 86400
    }
  }
`
	tests := []struct {
		name string
		// lifecycle block body
		lifecycle string
		// generated resource name -> expected ignore_changes value ("" = none)
		want map[string]string
		// substrings expected in warning diagnostics (empty = no diagnostics)
		wantWarnings []string
	}{
		{
			name:      "single setting",
			lifecycle: `ignore_changes = [settings[0].polish]`,
			want: map[string]string{
				"this_polish": "[value]", "this_brotli": "", "this_zero_rtt": "", "this_security_header": "",
			},
		},
		{
			name:      "legacy dotted index form",
			lifecycle: `ignore_changes = [settings.0.polish]`,
			want:      map[string]string{"this_polish": "[value]", "this_brotli": ""},
		},
		{
			name:      "renamed setting keeps v4 resource name",
			lifecycle: `ignore_changes = [settings[0].zero_rtt]`,
			want:      map[string]string{"this_zero_rtt": "[value]", "this_polish": ""},
		},
		{
			name:      "all",
			lifecycle: `ignore_changes = all`,
			want: map[string]string{
				"this_polish": "all", "this_brotli": "all", "this_zero_rtt": "all", "this_security_header": "all",
			},
		},
		{
			name:      "whole settings block",
			lifecycle: `ignore_changes = [settings]`,
			want: map[string]string{
				"this_polish": "[value]", "this_brotli": "[value]", "this_zero_rtt": "[value]", "this_security_header": "[value]",
			},
		},
		{
			name:      "deep path is widened with a warning",
			lifecycle: `ignore_changes = [settings[0].security_header[0].max_age]`,
			want:      map[string]string{"this_security_header": "[value]", "this_polish": ""},
			wantWarnings: []string{
				"widened",
				"settings[0].security_header[0].max_age -> cloudflare_zone_setting.this_security_header: [value]",
			},
		},
		{
			name:      "unmappable entries warn and are dropped",
			lifecycle: `ignore_changes = [zone_id, settings[0].universal_ssl, settings[0].brotli]`,
			want:      map[string]string{"this_brotli": "[value]", "this_polish": ""},
			wantWarnings: []string{
				"cloudflare_zone_settings_override.this",
				"zone_id",
				"settings[0].universal_ssl",
				"by hand",
			},
		},
		{
			name: "mixed with prevent_destroy",
			lifecycle: `prevent_destroy = true
    ignore_changes  = [settings[0].brotli]`,
			want: map[string]string{"this_brotli": "[value]", "this_polish": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "resource \"cloudflare_zone_settings_override\" \"this\" {\n" + settings +
				"\n  lifecycle {\n    " + tt.lifecycle + "\n  }\n}\n"
			resources, _, diags := runTransformConfig(t, input)

			_, hasUniversalSSL := resources["this_universal_ssl"]
			assert.False(t, hasUniversalSSL)

			for name, want := range tt.want {
				res, ok := resources[name]
				require.True(t, ok, "resource %s not generated", name)
				if want == "" {
					assert.NotContains(t, res, "ignore_changes", "resource %s", name)
				} else {
					assert.Regexp(t, `ignore_changes\s+= `+regexp.QuoteMeta(want)+`\n`, res, "resource %s", name)
				}
				if strings.Contains(tt.lifecycle, "prevent_destroy") {
					assert.Contains(t, res, "prevent_destroy = true", "resource %s", name)
					assert.Equal(t, 1, strings.Count(res, "lifecycle {"), "resource %s", name)
				}
			}

			var ignoreDiags []string
			for _, d := range diags {
				if strings.Contains(d.Summary, "ignore_changes") {
					assert.Equal(t, hcl.DiagWarning, d.Severity)
					ignoreDiags = append(ignoreDiags, d.Summary+"\n"+d.Detail)
				}
			}
			if len(tt.wantWarnings) == 0 {
				assert.Empty(t, ignoreDiags)
				return
			}
			require.Len(t, ignoreDiags, 1)
			for _, w := range tt.wantWarnings {
				assert.Contains(t, ignoreDiags[0], w)
			}
		})
	}
}
