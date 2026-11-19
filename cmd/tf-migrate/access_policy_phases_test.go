package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
)

const phasedFixture = `resource "cloudflare_access_application" "app" {
  account_id = var.account_id
  name       = "App"
  domain     = "app.example.com"
  type       = "self_hosted"
}

resource "cloudflare_access_policy" "deny" {
  application_id = cloudflare_access_application.app.id
  account_id     = var.account_id
  name           = "Deny"
  decision       = "deny"
  precedence     = 2

  include {
    email_domain = ["contractor.example.com"]
  }
}

resource "cloudflare_access_policy" "allow" {
  application_id = cloudflare_access_application.app.id
  account_id     = var.account_id
  name           = "Allow"
  decision       = "allow"
  precedence     = 1

  include {
    email = ["a@example.com"]
  }
}

resource "cloudflare_access_policy" "by_zone" {
  application_id = cloudflare_access_application.app.id
  zone_id        = "zone"
  name           = "Zone"
  decision       = "allow"
  precedence     = 3

  include {
    everyone = true
  }
}
`

// runWithAnswers runs one tf-migrate migrate with stdin answering every prompt "y".
func runWithAnswers(t *testing.T, cfg config) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, _ = w.WriteString(strings.Repeat("y\n", 10))
	require.NoError(t, w.Close())
	stdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = stdin }()
	accesspolicyphase.ResetCache()
	return capturePolicyOutput(t, func() { require.NoError(t, runMigration(newTestLogger(), cfg)) })
}

func TestAccessPolicyPhasedSteps(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tf")
	record := filepath.Join(dir, accesspolicyphase.RecordFile)
	require.NoError(t, os.WriteFile(main, []byte(phasedFixture), 0600))
	cfg := config{configDir: dir, sourceVersion: "v4", targetVersion: "v5", targetProviderVersion: "5.29.0", skipVersionCheck: true}
	read := func(p string) string { b, err := os.ReadFile(p); require.NoError(t, err); return string(b) }

	// Step 1: only the record is written; main.tf is untouched.
	out := runWithAnswers(t, cfg)
	assert.Contains(t, out, "step 1 of 3")
	assert.Contains(t, out, "cloudflare_access_policy.by_zone: has no account_id")
	assert.Equal(t, phasedFixture, read(main))
	assert.Equal(t, accesspolicyphase.Recorded, accesspolicyphase.StateOf(dir))
	rec := read(record)
	assert.Contains(t, rec, "cloudflare_access_policy.allow.id")
	assert.Contains(t, rec, "cloudflare_access_policy.deny.id")
	assert.NotContains(t, rec, "by_zone")

	// Re-running step 1 before apply asks and, on "y", moves to step 2: the
	// two supported policies are removed without destroy and the record frozen.
	out = runWithAnswers(t, cfg)
	assert.Contains(t, out, "make-access-policies-reusable")
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(dir))
	m := read(main)
	assert.Contains(t, m, "# tf-migrate: resource \"cloudflare_access_policy\" \"allow\"")
	assert.Contains(t, m, "# tf-migrate: resource \"cloudflare_access_policy\" \"deny\"")
	assert.Equal(t, 2, strings.Count(m, "destroy = false"))
	assert.Contains(t, m, "resource \"cloudflare_access_policy\" \"by_zone\"", "unsupported policy is left for the manual path")

	// Step 3: v5 policies imported by recorded ID, listed on the app in
	// precedence order, no moved block for them.
	runWithAnswers(t, cfg)
	m = read(main)
	assert.NotContains(t, m, "# tf-migrate:")
	assert.Contains(t, m, `id = "${var.account_id}/${terraform_data.tf_migrate_access_policies.output.allow.id}"`)
	assert.Contains(t, m, `id = "${var.account_id}/${terraform_data.tf_migrate_access_policies.output.deny.id}"`)
	assert.NotContains(t, m, "from = cloudflare_access_policy.allow")
	assert.NotContains(t, m, "\n  application_id", "application_id is dropped from imported policies")
	allow := strings.Index(m, "id         = cloudflare_zero_trust_access_policy.allow.id")
	deny := strings.Index(m, "id         = cloudflare_zero_trust_access_policy.deny.id")
	require.True(t, allow > 0 && deny > allow, "policies on the app, in precedence order:\n%s", m)
	assert.NotContains(t, m, "cloudflare_zero_trust_access_policy.by_zone.id", "unsupported policy is not attached")
	assert.Contains(t, m, "from = cloudflare_access_policy.by_zone", "unsupported policy keeps the manual removed {}")
}
