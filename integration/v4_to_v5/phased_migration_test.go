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
// Phase 1: tf-migrate appends removed {} blocks while leaving the v4 config intact.
//
//	The user commits and pushes; Atlantis applies with the v4 provider, dropping
//	the old state entries via Terraform's lifecycle mechanism.
//
// Phase 2: the user re-runs tf-migrate and confirms the apply succeeded (--yes
//
//	in non-interactive/CI mode), and the full v4→v5 migration runs.
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

	// -------------------------------------------------------------------------
	// Phase 1: resource block replaced by removed {} block
	// -------------------------------------------------------------------------
	t.Run("Phase1_ReplacesResourceWithRemovedBlock", func(t *testing.T) {
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

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)

		// Original v4 resource block must be GONE — Terraform errors if removed {}
		// and its resource {} block coexist in the same config.
		assert.NotContains(t, out, `resource "cloudflare_zone_settings_override"`,
			"original v4 resource block must be removed in phase 1")

		// A removed block must be present
		assert.Contains(t, out, `removed {`,
			"removed block should be present")
		assert.Contains(t, out, `cloudflare_zone_settings_override.example`,
			"removed block should reference the correct resource address")
		assert.Contains(t, out, `destroy = false`,
			"removed block must set destroy = false")

		// v5 resources must NOT yet be present
		assert.NotContains(t, out, `resource "cloudflare_zone_setting"`,
			"v5 cloudflare_zone_setting resources should not appear until phase 2")
		assert.NotContains(t, out, `import {`,
			"import blocks should not appear until phase 2")

		// No marker file — detection is based on the removed {} blocks
		markerPath := filepath.Join(dir, ".tf-migrate-phase1-complete")
		_, statErr := os.Stat(markerPath)
		assert.True(t, os.IsNotExist(statErr), "no marker file should be written")
	})

	// -------------------------------------------------------------------------
	// Phase 2: full migration runs when --yes confirms the apply succeeded
	// -------------------------------------------------------------------------
	t.Run("Phase2_RunsFullMigrationWithYesFlag", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")

		// Simulate the e2e runner's second call: fresh v4 files (resource blocks
		// present), state already cleaned by phase-1 apply. --yes skips straight
		// to full migration without any phase detection.
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
			"--yes", // auto-confirm "did you apply the v4 config?" prompt
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

		// removed block should still be present in the final output
		assert.True(t, strings.Contains(out, "removed {"),
			"removed block should remain in the migrated config")
	})

	// -------------------------------------------------------------------------
	// Phase 1 only: user runs tf-migrate, phase 1 output produced, no phase 2
	// -------------------------------------------------------------------------
	t.Run("Phase1Only_NoYesFlag_StopsAfterPhase1", func(t *testing.T) {
		dir := t.TempDir()
		inputFile := filepath.Join(dir, "zone_setting.tf")

		// The interactive prompt fires when the local files have BOTH the original
		// resource blocks AND removed {} blocks (user ran phase 1 locally, which
		// removes the resource block but they somehow still have it — or the user
		// manually added a removed {} block). Simulate this with just the resource
		// block present; phase 1 will run and produce the removed {} output, then
		// we pipe "n" to decline proceeding.
		require.NoError(t, os.WriteFile(inputFile, []byte(inputTF), 0644))

		migrateCmd := exec.Command(binaryPath,
			"migrate",
			"--config-dir", dir,
			"--source-version", "v4",
			"--target-version", "v5",
			"--backup=false",
			// no --yes, pipe "n" as stdin
		)
		migrateCmd.Dir = dir
		migrateCmd.Stdin = strings.NewReader("n\n")
		cmdOut, err := migrateCmd.CombinedOutput()
		require.NoError(t, err, "tf-migrate should exit cleanly on 'n': %s", string(cmdOut))

		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		out := string(content)

		// Phase 1 ran (resource block removed, removed {} added), user declined
		// proceeding to phase 2 — no v5 resources should have been generated.
		assert.Contains(t, out, `removed {`,
			"removed block should be present after phase 1")
		assert.NotContains(t, out, `resource "cloudflare_zone_settings_override"`,
			"original v4 resource should be gone after phase 1")
		assert.NotContains(t, out, `resource "cloudflare_zone_setting"`,
			"v5 resources should not appear when user declines phase 2")
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
		out := string(content)

		// DNS record should be renamed to cloudflare_dns_record
		assert.Contains(t, out, `resource "cloudflare_dns_record"`,
			"cloudflare_record should be migrated to cloudflare_dns_record")
	})
}
