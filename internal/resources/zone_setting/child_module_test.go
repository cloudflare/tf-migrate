package zone_setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestLooksLikeChildModule(t *testing.T) {
	t.Run("empty path is never a child module", func(t *testing.T) {
		assert.False(t, looksLikeChildModule(""))
	})

	t.Run("directory with a lock file is a root", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".terraform.lock.hcl"), nil, 0o644))
		assert.False(t, looksLikeChildModule(filepath.Join(dir, "main.tf")))
	})

	t.Run("directory with .terraform is a root", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".terraform"), 0o755))
		assert.False(t, looksLikeChildModule(filepath.Join(dir, "main.tf")))
	})

	t.Run("directory with neither is a child module", func(t *testing.T) {
		dir := t.TempDir()
		assert.True(t, looksLikeChildModule(filepath.Join(dir, "main.tf")))
	})
}

// Import blocks generated into a child module are rejected by Terraform at plan
// time. Until now the only hint was a comment inside the removed block, which
// does not appear in the end-of-run warnings.
func TestChildModuleImportWarning(t *testing.T) {
	const input = `resource "cloudflare_zone_settings_override" "zone_settings" {
  zone_id = var.zone_id
  settings {
    always_use_https = "on"
    brotli           = "on"
    security_header {
      enabled = true
    }
  }
}`

	const countInput = `resource "cloudflare_zone_settings_override" "zone_settings" {
  count   = 1
  zone_id = var.zone_id
  settings {
    brotli = "on"
  }
}`

	childDir := t.TempDir()
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, ".terraform.lock.hcl"), nil, 0o644))

	cases := []struct {
		name     string
		input    string
		filePath string
		wantWarn bool
	}{
		{"child module warns", input, filepath.Join(childDir, "settings.tf"), true},
		{"root module does not warn", input, filepath.Join(rootDir, "settings.tf"), false},
		{"no file path does not warn", input, "", false},
		// count produces no import blocks, so there is nothing to move.
		{"count in child module does not warn", countInput, filepath.Join(childDir, "settings.tf"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := transformForTest(t, tc.input, tc.filePath)

			var found *hcl.Diagnostic
			for _, d := range ctx.Diagnostics {
				if strings.Contains(d.Summary, "to the root module") {
					found = d
				}
			}

			if !tc.wantWarn {
				assert.Nil(t, found, "unexpected warning: %v", found)
				return
			}
			require.NotNil(t, found, "expected a child-module import warning, got: %v", ctx.Diagnostics)
			assert.Equal(t, hcl.DiagWarning, found.Severity)
			// 2 attributes + 1 nested block -> 3 import blocks.
			assert.Contains(t, found.Summary, "move 3 import block(s)")
			assert.Contains(t, found.Summary, "cloudflare_zone_settings_override.zone_settings")
			assert.Contains(t, found.Detail, "module.<name>.cloudflare_zone_setting.zone_settings_<setting>")
			assert.Contains(t, found.Detail, childDir)
		})
	}
}

func transformForTest(t *testing.T, input, filePath string) *transform.Context {
	t.Helper()
	file, diags := hclwrite.ParseConfig([]byte(input), "settings.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)

	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "settings.tf",
		FilePath:      filePath,
		Diagnostics:   make(hcl.Diagnostics, 0),
		Metadata:      make(map[string]interface{}),
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	var block *hclwrite.Block
	for _, b := range file.Body().Blocks() {
		if b.Type() == "resource" {
			block = b
		}
	}
	require.NotNil(t, block)

	_, err := NewV4ToV5Migrator().TransformConfig(ctx, block)
	require.NoError(t, err)
	return ctx
}
