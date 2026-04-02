package v4_to_v5

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/integration"
)

// TestPhasedMigration_ZoneSettingsOverride tests the two-phase migration workflow
// for cloudflare_zone_settings_override in an Atlantis-managed workspace.
//
// Phase 1: tf-migrate comments out the resource blocks (with a marker prefix)
// and appends removed {} blocks in the same file. Terraform only sees the
// removed {} blocks — no coexistence error. Atlantis applies with the v4
// provider, dropping the state entries.
//
// Phase 2: the user re-runs tf-migrate. The tool detects the commented-out
// blocks, asks for confirmation, uncomments them, removes the removed {}
// blocks, and runs the full v4→v5 migration.
func TestPhasedMigration_ZoneSettingsOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	runner, err := integration.NewTestRunner("v4", "v5")
	require.NoError(t, err, "Failed to create test runner")

	const inputTF = `resource "cloudflare_zone_settings_override" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    always_online = "on"
    ssl           = "full"
  }
}
`
	const markerPrefix = "# tf-migrate: "

	// Build the binary once — reused across sub-tests
	binaryPath := filepath.Join(runner.TfMigrateDir, "tf-migrate-phased-test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/tf-migrate")
	buildCmd.Dir = runner.TfMigrateDir
	out, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build tf-migrate binary: %s", out)
	defer os.Remove(binaryPath)

	// -------------------------------------------------------------------------
	// Phase 1: resource blocks commented out, removed {} blocks added
	// -------------------------------------------------------------------------
	t.Run("Phase1_CommentsOutResourceAddsRemovedBlock", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
			"--skip-version-check",
		)
		migrateCmd.Dir = dir
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate phase 1 failed: %s", string(cmdOut))

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)

		// Resource block must be commented out with the marker prefix
		assert.Contains(t, out, markerPrefix+`resource "cloudflare_zone_settings_override" "example" {`,
			"resource block should be commented out with marker prefix")

		// A removed {} block must be present
		assert.Contains(t, out, `removed {`)
		assert.Contains(t, out, `cloudflare_zone_settings_override.example`)
		assert.Contains(t, out, `destroy = false`)

		// No uncommented resource block visible to Terraform
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, `resource "cloudflare_zone_settings_override"`) {
				t.Errorf("resource block must be commented out, found uncommented: %s", line)
			}
		}

		// No v5 resources yet
		assert.NotContains(t, out, `resource "cloudflare_zone_setting"`)
		assert.NotContains(t, out, `import {`)

		// No separate cleanup file
		_, statErr := os.Stat(filepath.Join(dir, "_phase1_cleanup.tf"))
		assert.True(t, os.IsNotExist(statErr), "no separate cleanup file should be created")
	})

	// -------------------------------------------------------------------------
	// Phase 2 with --yes: full migration runs directly
	// -------------------------------------------------------------------------
	t.Run("Phase2_RunsFullMigrationWithYesFlag", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
			"--skip-phase-check",
			"--skip-version-check",
		)
		migrateCmd.Dir = dir
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate --yes failed: %s", string(cmdOut))

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)

		assert.NotContains(t, out, `resource "cloudflare_zone_settings_override"`)
		assert.Contains(t, out, `resource "cloudflare_zone_setting" "example_always_online"`)
		assert.Contains(t, out, `resource "cloudflare_zone_setting" "example_ssl"`)
		assert.Contains(t, out, `import {`)
	})

	// -------------------------------------------------------------------------
	// Phase 1 then phase 2 via prompt (y)
	// -------------------------------------------------------------------------
	t.Run("Phase2_PromptConfirmed_UncommentsAndMigrates", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		// Phase 1
		p1 := exec.Command(binaryPath,
			"migrate", "--config-dir", dir,
			"--source-version", "v4", "--target-version", "v5", "--backup=false", "--skip-version-check",
		)
		p1.Dir = dir
		_, err := p1.CombinedOutput()
		require.NoError(t, err)

		// Phase 2 — confirm with "y"
		p2 := exec.Command(binaryPath,
			"migrate", "--config-dir", dir,
			"--source-version", "v4", "--target-version", "v5", "--backup=false", "--skip-version-check",
		)
		p2.Dir = dir
		p2.Stdin = strings.NewReader("y\n")
		cmdOut, err := p2.CombinedOutput()
		require.NoError(t, err, "phase 2 prompt failed: %s", string(cmdOut))

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)

		assert.NotContains(t, out, `resource "cloudflare_zone_settings_override"`)
		assert.NotContains(t, out, markerPrefix, "no phase-1 markers should remain")
		assert.Contains(t, out, `resource "cloudflare_zone_setting"`)
		assert.Contains(t, out, `import {`)
	})

	// -------------------------------------------------------------------------
	// Phase 1 then decline (n) → file unchanged
	// -------------------------------------------------------------------------
	t.Run("Phase2_PromptDeclined_FileUnchanged", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		p1 := exec.Command(binaryPath,
			"migrate", "--config-dir", dir,
			"--source-version", "v4", "--target-version", "v5", "--backup=false", "--skip-version-check",
		)
		p1.Dir = dir
		_, err := p1.CombinedOutput()
		require.NoError(t, err)

		afterPhase1, _ := os.ReadFile(inputFile)

		p2 := exec.Command(binaryPath,
			"migrate", "--config-dir", dir,
			"--source-version", "v4", "--target-version", "v5", "--backup=false", "--skip-version-check",
		)
		p2.Dir = dir
		p2.Stdin = strings.NewReader("n\n")
		cmdOut, err := p2.CombinedOutput()
		require.NoError(t, err, "should exit cleanly on 'n': %s", string(cmdOut))

		afterDecline, _ := os.ReadFile(inputFile)
		assert.Equal(t, string(afterPhase1), string(afterDecline),
			"file must be unchanged when user declines")
	})

	// -------------------------------------------------------------------------
	// No phase-1 resources → normal migration
	// -------------------------------------------------------------------------
	t.Run("NoPhaseOneResources_RunsNormalMigration", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "dns.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(`resource "cloudflare_record" "example" {
  zone_id = "abc123"
  name    = "test"
  type    = "A"
  content = "1.2.3.4"
}
`), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate", "--config-dir", dir,
			"--source-version", "v4", "--target-version", "v5", "--backup=false", "--skip-version-check",
		)
		migrateCmd.Dir = dir
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate failed: %s", string(cmdOut))

		content, _ := os.ReadFile(inputFile)
		assert.Contains(t, string(content), `resource "cloudflare_dns_record"`)
		assert.NotContains(t, string(content), markerPrefix)
	})
}
