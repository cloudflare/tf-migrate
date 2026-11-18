package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const versionTestProviders = `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4"
    }
  }
}
`

func writeVersionTestFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(path, []byte(versionTestProviders), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Running the version update twice must not add a second version line.
func TestUpdateProviderVersionConstraintIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := writeVersionTestFile(t, dir)
	cfg := config{configDir: dir, targetVersion: "v5", targetProviderVersion: "5.29.0"}
	for i := 0; i < 2; i++ {
		if err := updateProviderVersionConstraint(newTestLogger(), cfg, nil); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(got), "version") != 1 || !strings.Contains(string(got), `version = "5.29.0"`) {
		t.Errorf("want one version = \"5.29.0\" after two runs, got:\n%s", got)
	}
}

// With --output-dir, the version is updated in the migrated copy, and the
// input file is left alone.
func TestUpdateProviderVersionConstraintOutputDir(t *testing.T) {
	in, out := t.TempDir(), t.TempDir()
	inPath := writeVersionTestFile(t, in)
	outPath := writeVersionTestFile(t, out)

	cfg := config{configDir: in, outputDir: out, targetVersion: "v5", targetProviderVersion: "5.29.0"}
	if err := updateProviderVersionConstraint(newTestLogger(), cfg, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(outPath); !strings.Contains(string(got), `version = "5.29.0"`) {
		t.Errorf("output not updated:\n%s", got)
	}
	if got, _ := os.ReadFile(inPath); string(got) != versionTestProviders {
		t.Errorf("input was modified:\n%s", got)
	}
}
