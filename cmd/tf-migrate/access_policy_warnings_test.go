package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
)

func capturePolicyOutput(t *testing.T, fn func()) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	defer file.Close()
	stdout := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = stdout }()
	fn()
	data, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	return string(data)
}

const appScopedPolicyFixture = `resource "cloudflare_access_application" "app" {
  account_id = "account-123"
  name = "App"
  type = "self_hosted"
}
resource "cloudflare_access_policy" "legacy" {
  account_id = "account-123"
  application_id = cloudflare_access_application.app.id
  name = "Employees"
  decision = "allow"
  precedence = 4
  include { email = ["employee@example.com"] }
}
resource "cloudflare_zero_trust_access_policy" "alias" {
  account_id = "account-123"
  application_id = cloudflare_access_application.app.id
  name = "Service access"
  decision = "non_identity"
  precedence = 8
  include { any_valid_service_token = true }
}
resource "cloudflare_access_policy" "reusable" {
  account_id = "account-123"
  name = "Reusable"
  decision = "allow"
  include { everyone = true }
}
output "legacy_ref" { value = cloudflare_access_policy.legacy.id }
output "alias_ref" { value = cloudflare_zero_trust_access_policy.alias.id }
output "reusable_ref" { value = cloudflare_access_policy.reusable.id }
`

func TestAccessPolicyManualWarnings(t *testing.T) {
	for _, mode := range []string{"normal", "verbose", "quiet", "dry-run"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "access.tf")
			require.NoError(t, os.WriteFile(path, []byte(appScopedPolicyFixture), 0600))
			cfg := config{configDir: dir, sourceVersion: "v4", targetVersion: "v5", targetProviderVersion: "5.27.0",
				verbose: mode == "verbose", quiet: mode == "quiet", dryRun: mode == "dry-run"}
			report, err := runPreMigrationScan(newTestLogger(), cfg)
			require.NoError(t, err)
			manual := 0
			for _, r := range report.Resources {
				if r.Class == classManualIntervention {
					manual++
				}
			}
			require.Equal(t, 2, manual)
			preflight := capturePolicyOutput(t, func() { printPreflightReport(report, cfg) })
			output := capturePolicyOutput(t, func() { require.NoError(t, runFullMigration(newTestLogger(), cfg)) })
			for _, text := range []string{preflight, output} {
				if cfg.quiet {
					require.NotContains(t, text, "ACTION REQUIRED")
					continue
				}
				require.Contains(t, text, "ACTION REQUIRED: Access policy migration is incomplete (2 policies)")
				require.Contains(t, text, "cloudflare_access_policy.legacy  (access.tf)")
				require.Contains(t, text, "cloudflare_zero_trust_access_policy.alias  (access.tf)")
				require.NotContains(t, text, "  ("+path+")")
				require.Contains(t, text, "can detach and delete")
			}
			if !cfg.quiet && !cfg.dryRun {
				require.Greater(t, strings.LastIndex(output, "ACTION REQUIRED"), strings.LastIndex(output, "Next steps:"))
			}
			migrated, err := os.ReadFile(path)
			require.NoError(t, err)
			if cfg.dryRun {
				require.Equal(t, appScopedPolicyFixture, string(migrated))
				return
			}
			parsed, diags := hclwrite.ParseConfig(migrated, path, hcl.InitialPos)
			require.False(t, diags.HasErrors(), diags.Error())
			removed := 0
			for _, block := range parsed.Body().Blocks() {
				if block.Type() == "removed" {
					removed++
					require.Contains(t, string(block.BuildTokens(nil).Bytes()), "#   application_id = cloudflare_access_application.app.id")
				}
			}
			require.Equal(t, 2, removed)
			require.Contains(t, string(migrated), "value = cloudflare_access_policy.legacy.id")
			require.Contains(t, string(migrated), "value = cloudflare_zero_trust_access_policy.alias.id")
			require.Contains(t, string(migrated), "value = cloudflare_zero_trust_access_policy.reusable.id")
			require.Contains(t, string(migrated), `resource "cloudflare_zero_trust_access_policy" "reusable"`)
			capturePolicyOutput(t, func() { require.NoError(t, runFullMigration(newTestLogger(), cfg)) })
			rerun, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(migrated), string(rerun), "rerunning must preserve the source comments without duplication")
		})
	}
}

type failedVersionTransport struct{}

func (failedVersionTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("version discovery unavailable")
}

func TestAccessPolicyReminderAfterVersionFetchFailure(t *testing.T) {
	transport := http.DefaultTransport
	http.DefaultTransport = failedVersionTransport{}
	defer func() { http.DefaultTransport = transport }()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "access.tf"), []byte(appScopedPolicyFixture), 0600))
	output := capturePolicyOutput(t, func() {
		require.NoError(t, runFullMigration(newTestLogger(), config{configDir: dir, sourceVersion: "v4", targetVersion: "v5"}))
	})
	require.Contains(t, output, "Could not fetch the latest provider version")
	require.Contains(t, output, "ACTION REQUIRED: Access policy migration is incomplete (2 policies)")
	require.Greater(t, strings.LastIndex(output, "ACTION REQUIRED"), strings.LastIndex(output, "Next steps:"), output)
}
