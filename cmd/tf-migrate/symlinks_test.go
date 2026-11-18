package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const symlinkTestProviders = `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4"
    }
  }
}
`

const symlinkTestRecord = `resource "cloudflare_record" "www" {
  zone_id = var.zone_id
  name    = "www"
  type    = "A"
  value   = "192.0.2.1"
}

variable "zone_id" {
  type = string
}
`

// newSymlinkTree builds a tree where two modules symlink a shared providers
// file and a shared resource file, as module repos commonly do:
//
//	providers.tf
//	modules/common.tf
//	modules/{a,b}/providers.tf -> ../../providers.tf
//	modules/{a,b}/common.tf    -> ../common.tf
func newSymlinkTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("providers.tf", symlinkTestProviders)
	write("modules/common.tf", symlinkTestRecord)
	for _, m := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(dir, "modules", m), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../../providers.tf", filepath.Join(dir, "modules", m, "providers.tf")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../common.tf", filepath.Join(dir, "modules", m, "common.tf")); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestFindTerraformFilesWithAliases(t *testing.T) {
	dir := newSymlinkTree(t)

	files, aliases, err := findTerraformFilesWithAliases(dir, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	providers := filepath.Join(dir, "providers.tf")
	common := filepath.Join(dir, "modules", "common.tf")
	if len(files) != 2 || files[0] != common || files[1] != providers {
		t.Fatalf("want each real file once (%s, %s), got %v", common, providers, files)
	}
	if got := aliases[providers]; len(got) != 2 {
		t.Errorf("want 2 symlinks to providers.tf, got %v", got)
	}
	if got := aliases[common]; len(got) != 2 {
		t.Errorf("want 2 symlinks to common.tf, got %v", got)
	}

	// The non-recursive wrapper used by every other caller dedupes too.
	listed, err := findTerraformFilesWithRecursion(dir, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Errorf("findTerraformFilesWithRecursion: want 2 files, got %v", listed)
	}
}

// A symlink is listed when its target is outside the config dir, since that
// path is the only way the file is reached.
func TestFindTerraformFilesWithAliases_TargetOutsideDir(t *testing.T) {
	tree := newSymlinkTree(t)
	dir := filepath.Join(tree, "modules", "a")

	files, aliases, err := findTerraformFilesWithAliases(dir, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || len(aliases) != 0 {
		t.Fatalf("want both symlinks listed and no aliases, got files=%v aliases=%v", files, aliases)
	}
}

func runSymlinkMigration(t *testing.T, cfg config) {
	t.Helper()
	cfg.recursive = true
	cfg.sourceVersion = "v4"
	cfg.targetVersion = "v5"
	cfg.targetProviderVersion = "5.29.0"
	cfg.skipVersionCheck = true
	if err := runFullMigration(newTestLogger(), cfg); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Each shared file is migrated once, not once per symlink. Before the fix,
// the version constraint was added again for every path, producing three
// version lines in providers.tf.
func TestSymlinkedFilesMigratedOnceInPlace(t *testing.T) {
	dir := newSymlinkTree(t)
	runSymlinkMigration(t, config{configDir: dir, outputDir: dir, backup: true})

	providers := readFile(t, filepath.Join(dir, "providers.tf"))
	if n := strings.Count(providers, "version"); n != 1 {
		t.Errorf("want 1 version line, got %d:\n%s", n, providers)
	}
	if !strings.Contains(providers, `version = "5.29.0"`) {
		t.Errorf("version not updated:\n%s", providers)
	}

	common := readFile(t, filepath.Join(dir, "modules", "common.tf"))
	if n := strings.Count(common, `resource "cloudflare_dns_record" "www"`); n != 1 {
		t.Errorf("want the record migrated once, got %d:\n%s", n, common)
	}

	for _, m := range []string{"a", "b"} {
		for _, f := range []string{"providers.tf", "common.tf"} {
			path := filepath.Join(dir, "modules", m, f)
			if !isSymlink(path) {
				t.Errorf("%s is no longer a symlink", path)
			}
			if _, err := os.Lstat(path + ".backup"); err == nil {
				t.Errorf("unexpected backup for symlink %s", path)
			}
		}
	}
}

// With a separate output directory, every path gets a migrated copy, the
// version bump lands in the output and the input is left alone.
func TestSymlinkedFilesWithOutputDir(t *testing.T) {
	dir := newSymlinkTree(t)
	out := t.TempDir()
	runSymlinkMigration(t, config{configDir: dir, outputDir: out})

	for _, rel := range []string{"providers.tf", "modules/a/providers.tf", "modules/b/providers.tf"} {
		got := readFile(t, filepath.Join(out, rel))
		if strings.Count(got, "version") != 1 || !strings.Contains(got, `version = "5.29.0"`) {
			t.Errorf("%s: want one version = \"5.29.0\", got:\n%s", rel, got)
		}
	}
	for _, rel := range []string{"modules/common.tf", "modules/a/common.tf", "modules/b/common.tf"} {
		if got := readFile(t, filepath.Join(out, rel)); !strings.Contains(got, `resource "cloudflare_dns_record" "www"`) {
			t.Errorf("%s not migrated:\n%s", rel, got)
		}
	}
	if got := readFile(t, filepath.Join(dir, "providers.tf")); got != symlinkTestProviders {
		t.Errorf("input providers.tf was modified:\n%s", got)
	}
}

func TestSymlinkOutsideDirWarning(t *testing.T) {
	tree := newSymlinkTree(t)
	module := filepath.Join(tree, "modules", "a")
	link := filepath.Join(module, "common.tf")

	inPlace := config{configDir: module, outputDir: module}
	d := symlinkOutsideDirWarning(inPlace, link)
	if d == nil {
		t.Fatal("expected a warning for a symlink to a file outside the config dir")
	}
	if !strings.Contains(d.Summary, filepath.Join("modules", "common.tf")) {
		t.Errorf("summary should name the target: %s", d.Summary)
	}

	if d := symlinkOutsideDirWarning(config{configDir: tree, outputDir: tree}, link); d != nil {
		t.Errorf("target inside the config dir must not warn: %s", d.Summary)
	}
	if d := symlinkOutsideDirWarning(config{configDir: module, outputDir: t.TempDir()}, link); d != nil {
		t.Errorf("--output-dir does not edit the target and must not warn: %s", d.Summary)
	}
	if d := symlinkOutsideDirWarning(inPlace, filepath.Join(tree, "providers.tf")); d != nil {
		t.Errorf("a regular file must not warn: %s", d.Summary)
	}
}

// A per-hostname certificate declared in a file shared by symlink is declared
// in every module that links it, so references in each of those modules must
// be renamed, not only in the module whose path was migrated.
func TestSymlinkedPerHostnameCertificateReferences(t *testing.T) {
	for _, mode := range []string{"in place", "output dir"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "shared"), 0o755); err != nil {
				t.Fatal(err)
			}
			cert := `resource "cloudflare_authenticated_origin_pulls_certificate" "host" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  certificate = "cert"
  private_key = "key"
  type        = "per-hostname"
}
`
			if err := os.WriteFile(filepath.Join(dir, "shared", "certs.tf"), []byte(cert), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, m := range []string{"a", "b"} {
				mod := filepath.Join(dir, "mods", m)
				if err := os.MkdirAll(mod, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../../shared/certs.tf", filepath.Join(mod, "certs.tf")); err != nil {
					t.Fatal(err)
				}
				out := "output \"cert_id\" {\n  value = cloudflare_authenticated_origin_pulls_certificate.host.id\n}\n"
				if err := os.WriteFile(filepath.Join(mod, "outputs.tf"), []byte(out), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			// Run on mods/ only, so both links point outside the config dir and
			// mods/a/certs.tf is the path that gets migrated.
			cfg := config{configDir: filepath.Join(dir, "mods")}
			cfg.outputDir = cfg.configDir
			if mode == "output dir" {
				cfg.outputDir = t.TempDir()
			}
			runSymlinkMigration(t, cfg)

			for _, m := range []string{"a", "b"} {
				got := readFile(t, filepath.Join(cfg.outputDir, m, "outputs.tf"))
				if !strings.Contains(got, "cloudflare_authenticated_origin_pulls_hostname_certificate.host.id") {
					t.Errorf("mods/%s/outputs.tf not renamed:\n%s", m, got)
				}
			}
		})
	}
}
