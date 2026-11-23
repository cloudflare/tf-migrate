package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const settingsModule = `variable "zone_id" {}

resource "cloudflare_zone_settings_override" "s" {
  zone_id = var.zone_id
  settings {
    always_use_https = "on"
  }
}
`

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	return dir
}

func importCount(s string) int {
	return strings.Count(s, "\nimport {") + boolInt(strings.HasPrefix(s, "import {"))
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestHoistModuleImports(t *testing.T) {
	root := `variable "zones" {
  default = { a = "z-a" }
}

module "m" {
  source  = "./mod"
  zone_id = "z-1"
}

module "each" {
  source   = "./mod"
  for_each = var.zones
  zone_id  = each.value
}
`
	t.Run("in place, and again over the output", func(t *testing.T) {
		dir := writeTree(t, map[string]string{"main.tf": root, "mod/main.tf": settingsModule})
		for i := 0; i < 2; i++ {
			runSymlinkMigration(t, config{configDir: dir, outputDir: dir})
		}
		got := readFile(t, filepath.Join(dir, "main.tf"))
		assert.Equal(t, 2, importCount(got), got)
		assert.Contains(t, got, `to = module.m.cloudflare_zone_setting.s_always_use_https`)
		assert.Contains(t, got, `id = "z-1/always_use_https"`)
		assert.Contains(t, got, "for_each = var.zones\n  to       = module.each[each.key].cloudflare_zone_setting.s_always_use_https")
		assert.Contains(t, got, `id       = "${each.value}/always_use_https"`)
		mod := readFile(t, filepath.Join(dir, "mod", "main.tf"))
		assert.Equal(t, 0, importCount(mod), mod)
		assert.Contains(t, mod, `resource "cloudflare_zone_setting" "s_always_use_https"`)
	})

	t.Run("output dir", func(t *testing.T) {
		in := writeTree(t, map[string]string{"main.tf": root, "mod/main.tf": settingsModule})
		out := t.TempDir()
		runSymlinkMigration(t, config{configDir: in, outputDir: out})
		assert.Equal(t, 2, importCount(readFile(t, filepath.Join(out, "main.tf"))))
		assert.Equal(t, 0, importCount(readFile(t, filepath.Join(out, "mod", "main.tf"))))
		assert.Equal(t, root, readFile(t, filepath.Join(in, "main.tf")))
	})

	t.Run("variable default", func(t *testing.T) {
		mod := strings.Replace(settingsModule, `variable "zone_id" {}`, `variable "zone_id" {
  default = "z-default"
}`, 1)
		dir := writeTree(t, map[string]string{
			"main.tf":     "module \"m\" {\n  source = \"./mod\"\n}\n",
			"mod/main.tf": mod,
		})
		runSymlinkMigration(t, config{configDir: dir, outputDir: dir})
		assert.Contains(t, readFile(t, filepath.Join(dir, "main.tf")), `id = "z-default/always_use_https"`)
	})

	// Blocks that can't be written in the caller stay in the module.
	for _, tc := range []struct {
		name   string
		files  map[string]string
		reason string
	}{
		{"local in id", map[string]string{
			"main.tf":     "module \"m\" {\n  source = \"./mod\"\n}\n",
			"mod/main.tf": strings.Replace(settingsModule, "var.zone_id", "local.zone", 1) + "locals {\n  zone = \"z\"\n}\n",
		}, "uses local.*"},
		{"count on the call", map[string]string{
			"main.tf":     "module \"m\" {\n  source  = \"./mod\"\n  count   = 1\n  zone_id = \"z\"\n}\n",
			"mod/main.tf": settingsModule,
		}, "uses count"},
		{"argument not set, no default", map[string]string{
			"main.tf":     "module \"m\" {\n  source = \"./mod\"\n}\n",
			"mod/main.tf": settingsModule,
		}, "doesn't set zone_id"},
		{"caller is itself a module", map[string]string{
			"main.tf":     "module \"a\" {\n  source = \"./a\"\n}\n",
			"a/main.tf":   "module \"b\" {\n  source  = \"../mod\"\n  zone_id = \"z\"\n}\n",
			"mod/main.tf": settingsModule,
		}, "itself a module"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTree(t, tc.files)
			runSymlinkMigration(t, config{configDir: dir, outputDir: dir})
			assert.Equal(t, 1, importCount(readFile(t, filepath.Join(dir, "mod", "main.tf"))))
			assert.Equal(t, 0, importCount(readFile(t, filepath.Join(dir, "main.tf"))))

			// One warning with the reason replaces the per-resource one.
			paths := []string{filepath.Join(dir, "main.tf"), filepath.Join(dir, "mod", "main.tf")}
			if _, err := os.Stat(filepath.Join(dir, "a", "main.tf")); err == nil {
				paths = append(paths, filepath.Join(dir, "a", "main.tf"))
			}
			perResource := &hcl.Diagnostic{Severity: hcl.DiagWarning,
				Subject: &hcl.Range{Filename: filepath.Join(dir, "mod", "main.tf")},
				Summary: "Action required: move 1 import block(s) for cloudflare_zone_settings_override.s to the root module"}
			diags := hoistModuleImports(newTestLogger(), config{configDir: dir, outputDir: dir, recursive: true}, paths, hcl.Diagnostics{perResource})
			require.Len(t, diags, 1)
			assert.Contains(t, diags[0].Summary, "move 1 import block(s) in mod to the root module")
			assert.Contains(t, diags[0].Detail, tc.reason)
			assert.Contains(t, diags[0].Detail, "cloudflare_zone_setting.s_always_use_https")
		})
	}

	t.Run("moved blocks replace the per-resource warning", func(t *testing.T) {
		dir2 := writeTree(t, map[string]string{
			"main.tf":     "module \"m\" {\n  source  = \"./mod\"\n  zone_id = \"z\"\n}\n",
			"mod/main.tf": "variable \"zone_id\" {}\n\nimport {\n  to = cloudflare_zone_setting.x\n  id = \"${var.zone_id}/ssl\"\n}\n",
		})
		other := &hcl.Diagnostic{Severity: hcl.DiagWarning, Summary: "unrelated"}
		perResource := &hcl.Diagnostic{Severity: hcl.DiagWarning,
			Subject: &hcl.Range{Filename: filepath.Join(dir2, "mod", "main.tf")},
			Summary: "Action required: move 1 import block(s) for cloudflare_zone_settings_override.s to the root module"}
		diags := hoistModuleImports(newTestLogger(), config{configDir: dir2, outputDir: dir2, recursive: true},
			[]string{filepath.Join(dir2, "main.tf"), filepath.Join(dir2, "mod", "main.tf")}, hcl.Diagnostics{other, perResource})
		require.Len(t, diags, 2)
		assert.Equal(t, "unrelated", diags[0].Summary)
		assert.Contains(t, diags[1].Summary, "Moved 1 import block(s) from module mod")
		assert.Contains(t, readFile(t, filepath.Join(dir2, "main.tf")), "to = module.m.cloudflare_zone_setting.x\n  id = \"z/ssl\"")
	})
}
