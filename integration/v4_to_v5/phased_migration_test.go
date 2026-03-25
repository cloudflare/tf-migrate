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
// Phase 1: tf-migrate writes removed {} blocks to a separate _phase1_cleanup.tf
//
//	file, leaving the original .tf files completely untouched. The user commits
//	_phase1_cleanup.tf; Atlantis applies it with the v4 provider, dropping the
//	old state entries. The user then deletes _phase1_cleanup.tf.
//
// Phase 2: the user re-runs tf-migrate (original files intact). The tool detects
//
//	that _phase1_cleanup.tf no longer exists and that the state is clean,
//	confirms with the user (or --yes), and runs the full v4→v5 migration.
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

	// Build the binary once — reused across sub-tests
	binaryPath := filepath.Join(runner.TfMigrateDir, "tf-migrate-phased-test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/tf-migrate")
	buildCmd.Dir = runner.TfMigrateDir
	out, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build tf-migrate binary: %s", out)
	defer os.Remove(binaryPath)

	cleanupFilename := "_phase1_cleanup.tf"

	// -------------------------------------------------------------------------
	// Phase 1: _phase1_cleanup.tf written, original file untouched
	// -------------------------------------------------------------------------
	t.Run("Phase1_WritesCleanupFileOriginalUntouched", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
		)
		migrateCmd.Dir = dir
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate phase 1 failed: %s", string(cmdOut))

		// Original file must be completely unchanged
		original, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		assert.Equal(t, inputTF, string(original),
			"original .tf file must be completely unchanged after phase 1")

		// _phase1_cleanup.tf must have been written
		cleanupPath := filepath.Join(dir, cleanupFilename)
		cleanupContent, err := os.ReadFile(cleanupPath)
		require.NoError(t, err, "_phase1_cleanup.tf should exist after phase 1")
		cleanupStr := string(cleanupContent)

		// Cleanup file must contain the removed {} block
		assert.Contains(t, cleanupStr, `removed {`,
			"cleanup file should contain a removed block")
		assert.Contains(t, cleanupStr, `cloudflare_zone_settings_override.example`,
			"cleanup file removed block should reference the correct resource")
		assert.Contains(t, cleanupStr, `destroy = false`,
			"cleanup file removed block must set destroy = false")

		// Cleanup file must NOT contain v5 resources
		assert.NotContains(t, cleanupStr, `resource "cloudflare_zone_setting"`,
			"v5 resources should not appear in the cleanup file")
	})

	// -------------------------------------------------------------------------
	// Phase 2: full migration runs with --yes (original files intact)
	// -------------------------------------------------------------------------
	t.Run("Phase2_RunsFullMigrationWithYesFlag", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		// Original v4 file intact — as it would be after the user deletes
		// _phase1_cleanup.tf following a successful Atlantis apply.
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
			"--yes", // skip phase detection — go straight to full migration
		)
		migrateCmd.Dir = dir
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate phase 2 failed: %s", string(cmdOut))

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)

		// v4 resource must be gone
		assert.NotContains(t, out, `resource "cloudflare_zone_settings_override"`,
			"original v4 resource block should be removed after phase 2")

		// v5 resources must be present
		assert.Contains(t, out, `resource "cloudflare_zone_setting" "example_always_online"`,
			"v5 zone_setting for always_online should be present")
		assert.Contains(t, out, `resource "cloudflare_zone_setting" "example_ssl"`,
			"v5 zone_setting for ssl should be present")

		// import blocks must be present
		assert.Contains(t, out, `import {`,
			"import blocks should be present after phase 2")

		// removed block should be in the final output
		assert.True(t, strings.Contains(out, "removed {"),
			"removed block should be in the migrated config")

		// No cleanup file should be left (--yes bypasses phase detection entirely)
		cleanupPath := filepath.Join(dir, cleanupFilename)
		_, statErr := os.Stat(cleanupPath)
		assert.True(t, os.IsNotExist(statErr), "no cleanup file should exist after phase 2 with --yes")
	})

	// -------------------------------------------------------------------------
	// Phase 2 prompt: cleanup file present, user confirms → full migration
	// -------------------------------------------------------------------------
	t.Run("Phase2_PromptConfirmed_RunsFullMigration", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		// Simulate: phase 1 already ran (cleanup file exists), user re-runs
		// tf-migrate and is asked whether the apply succeeded.
		cleanupPath := filepath.Join(dir, cleanupFilename)
		require.NoError(t, os.WriteFile(cleanupPath, []byte(`
removed {
  from = cloudflare_zone_settings_override.example
  lifecycle { destroy = false }
}
`), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
		)
		migrateCmd.Dir = dir
		migrateCmd.Stdin = strings.NewReader("y\n") // confirm the apply succeeded
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate phase 2 (prompt) failed: %s", string(cmdOut))

		// Cleanup file must be deleted
		_, statErr := os.Stat(cleanupPath)
		assert.True(t, os.IsNotExist(statErr), "cleanup file should be deleted after confirmed phase 2")

		// Full migration must have run
		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)
		assert.NotContains(t, out, `resource "cloudflare_zone_settings_override"`)
		assert.Contains(t, out, `resource "cloudflare_zone_setting"`)
	})

	// -------------------------------------------------------------------------
	// Phase 2 prompt: cleanup file present, user declines → no migration
	// -------------------------------------------------------------------------
	t.Run("Phase2_PromptDeclined_NoMigration", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		cleanupPath := filepath.Join(dir, cleanupFilename)
		require.NoError(t, os.WriteFile(cleanupPath, []byte(`
removed {
  from = cloudflare_zone_settings_override.example
  lifecycle { destroy = false }
}
`), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
		)
		migrateCmd.Dir = dir
		migrateCmd.Stdin = strings.NewReader("n\n") // decline
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate should exit cleanly on 'n': %s", string(cmdOut))

		// Original file must be unchanged
		original, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		assert.Equal(t, inputTF, string(original), "original file must be unchanged when user declines")

		// Cleanup file must still exist
		_, statErr := os.Stat(cleanupPath)
		assert.NoError(t, statErr, "cleanup file should still exist when user declines")

		// No v5 resources
		assert.NotContains(t, string(original), `resource "cloudflare_zone_setting"`)
	})

	// -------------------------------------------------------------------------
	// No phase-1 resources → runs normal migration without any prompt
	// -------------------------------------------------------------------------
	t.Run("NoPhaseOneResources_RunsNormalMigration", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "dns.tf")
		dnsInput := `resource "cloudflare_record" "example" {
  zone_id = "abc123"
  name    = "test"
  type    = "A"
  content = "1.2.3.4"
}
`
		require.NoError(t, os.WriteFile(inputFile, []byte(dnsInput), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
		)
		migrateCmd.Dir = dir
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate failed: %s", string(cmdOut))

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), `resource "cloudflare_dns_record"`,
			"cloudflare_record should be migrated to cloudflare_dns_record")

		// No cleanup file should exist
		cleanupPath := filepath.Join(dir, cleanupFilename)
		_, statErr := os.Stat(cleanupPath)
		assert.True(t, os.IsNotExist(statErr), "no cleanup file for non-phased resources")
	})
}
