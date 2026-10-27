package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestPreserveZeroTrustSettingsAfterMigration(t *testing.T) {
	registry.RegisterAllMigrations()
	input := []byte(`
resource "cloudflare_zero_trust_access_organization" "example" {
  account_id = var.account_id
  name = "Example"
  auth_domain = "example.cloudflareaccess.com"
}
resource "cloudflare_teams_account" "example" {
  account_id = var.account_id
  tls_decrypt_enabled = true
  proxy {
    tcp = true
    udp = true
    root_ca = true
    virtual_ip = false
  }
}
`)
	p := pipeline.BuildConfigPipeline(hclog.NewNullLogger(), getProviders())
	migrated, err := p.Transform(&transform.Context{Content: input, Filename: "main.tf", SourceVersion: "v4", TargetVersion: "v5", Metadata: map[string]interface{}{}})
	require.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(path, migrated, 0600))
	snapshot := filepath.Join(dir, "snapshot.json")
	require.NoError(t, os.WriteFile(snapshot, []byte(`{
"cloudflare_zero_trust_organization.example":{"result":{"mfa_required_for_all_apps":true,"mfa_config":{"allowed_authenticators":["totp"],"session_duration":"24h"}}},
"cloudflare_zero_trust_gateway_settings.example":{"result":{"settings":{"host_selector":{"enabled":true},"sandbox":{"enabled":true,"fallback_action":"block"}}}},
"cloudflare_zero_trust_device_settings.example_device_settings":{"result":{"external_emergency_signal_enabled":true,"external_emergency_signal_interval":"7m"}}
}`), 0600))
	cfg := &config{configDir: dir, dryRun: true}
	cmd := newPreserveZeroTrustSettingsCommand(cfg)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--snapshot", snapshot})
	require.NoError(t, cmd.Execute())
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, migrated, unchanged, "dry run must not modify files")
	require.Contains(t, stdout.String(), "mfa_required_for_all_apps")
	cfg.dryRun = false
	stdout.Reset()
	require.NoError(t, cmd.Execute())
	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, field := range []string{"mfa_required_for_all_apps", "mfa_config", "host_selector", "sandbox", "external_emergency_signal_enabled", "external_emergency_signal_interval"} {
		require.Contains(t, string(updated), field)
	}
	require.Contains(t, string(updated), "gateway_proxy_enabled")
	require.Contains(t, string(updated), "moved {")
	require.NotContains(t, string(updated), "ignore_changes")
	stdout.Reset()
	require.NoError(t, cmd.Execute())
	require.Empty(t, stdout.String(), "restoration must be idempotent")
}

func TestPreserveZeroTrustSettingsValidatesBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	input := []byte(`resource "cloudflare_zero_trust_organization" "example" {}`)
	require.NoError(t, os.WriteFile(path, input, 0600))
	snapshot := filepath.Join(dir, "snapshot.json")
	require.NoError(t, os.WriteFile(snapshot, []byte(`{
"cloudflare_zero_trust_organization.example":{"mfa_required_for_all_apps":true},
"cloudflare_zero_trust_organization.missing":{}
}`), 0600))
	cmd := newPreserveZeroTrustSettingsCommand(&config{configDir: dir})
	cmd.SetArgs([]string{"--snapshot", snapshot})
	require.Error(t, cmd.Execute())
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, input, actual)
}
