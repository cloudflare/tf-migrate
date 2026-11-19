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

resource "cloudflare_access_application" "zone_app" {
  zone_id = var.zone_id
  name    = "Zone app"
  domain  = "zone.example.com"
  type    = "self_hosted"
}

resource "cloudflare_access_policy" "by_zone" {
  application_id = cloudflare_access_application.zone_app.id
  zone_id        = var.zone_id
  name           = "Zone"
  decision       = "allow"
  precedence     = 1

  include {
    everyone = true
  }
}

resource "cloudflare_access_policy" "looped" {
  for_each       = { x = 3, y = 4 }
  application_id = cloudflare_access_application.app.id
  account_id     = var.account_id
  name           = each.key
  decision       = "allow"
  precedence     = each.value

  include {
    everyone = true
  }
}

resource "cloudflare_access_application" "other" {
  account_id = var.account_id
  name       = "Other"
  domain     = "other.example.com"
  type       = "self_hosted"
}

resource "cloudflare_access_policy" "counted" {
  count          = 1
  application_id = cloudflare_access_application.other.id
  account_id     = var.account_id
  name           = "counted"
  decision       = "allow"
  precedence     = 9

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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "providers.tf"), []byte("provider \"cloudflare\" {}\n"), 0600))
	cfg := config{configDir: dir, sourceVersion: "v4", targetVersion: "v5", targetProviderVersion: "5.29.0", skipVersionCheck: true}
	read := func(p string) string { b, err := os.ReadFile(p); require.NoError(t, err); return string(b) }

	// Step 1: only the record is written; main.tf is untouched.
	out := runWithAnswers(t, cfg)
	assert.Contains(t, out, "step 1 of 3")
	assert.Contains(t, out, "cloudflare_access_policy.counted: uses count")
	assert.Equal(t, phasedFixture, read(main))
	assert.Equal(t, accesspolicyphase.Recorded, accesspolicyphase.StateOf(dir))
	rec := read(record)
	assert.Contains(t, rec, "cloudflare_access_policy.allow.id")
	assert.Contains(t, rec, "cloudflare_access_policy.deny.id")
	assert.Contains(t, rec, "account_id     = data.cloudflare_zone.tf_migrate_access_policies_0.account_id")
	assert.Contains(t, rec, "zone_id = var.zone_id")
	assert.Contains(t, rec, "looped = { for k, i in cloudflare_access_policy.looped : k => {")
	assert.NotContains(t, rec, "counted")

	// Re-running step 1 before apply asks and, on "y", moves to step 2: the
	// two supported policies are removed without destroy and the record frozen.
	out = runWithAnswers(t, cfg)
	assert.Contains(t, out, "make-access-policies-reusable")
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(dir))
	m := read(main)
	assert.Contains(t, m, "# tf-migrate: resource \"cloudflare_access_policy\" \"allow\"")
	assert.Contains(t, m, "# tf-migrate: resource \"cloudflare_access_policy\" \"deny\"")
	assert.Equal(t, 4, strings.Count(m, "destroy = false"))
	assert.Contains(t, m, "resource \"cloudflare_access_policy\" \"counted\"", "unsupported policy is left for the manual path")

	// Step 3: v5 policies imported by recorded ID, listed on the app in
	// precedence order, no moved block for them.
	runWithAnswers(t, cfg)
	m = read(main)
	assert.NotContains(t, m, "# tf-migrate:")
	assert.Contains(t, m, `id = "${var.account_id}/${terraform_data.tf_migrate_access_policies.output.allow.id}"`)
	assert.Contains(t, m, `id = "${var.account_id}/${terraform_data.tf_migrate_access_policies.output.deny.id}"`)
	assert.NotContains(t, m, "from = cloudflare_access_policy.allow")
	assert.NotContains(t, m, "\n  application_id", "application_id is dropped from imported policies")
	// zone policy: account-level in v5, account read from the record
	assert.Contains(t, m, "account_id = terraform_data.tf_migrate_access_policies.output.by_zone.account_id")
	assert.Contains(t, m, `id = "${terraform_data.tf_migrate_access_policies.output.by_zone.account_id}/${terraform_data.tf_migrate_access_policies.output.by_zone.id}"`)
	assert.Contains(t, m, "id         = cloudflare_zero_trust_access_policy.by_zone.id", "single policies only: static list")
	// for_each policy: one import over the record, instances listed on the app
	assert.Contains(t, m, "for_each = terraform_data.tf_migrate_access_policies.output.looped")
	assert.Contains(t, m, "to       = cloudflare_zero_trust_access_policy.looped[each.key]")
	assert.Contains(t, m, `{ for k, v in terraform_data.tf_migrate_access_policies.output.looped : "looped/${k}" => { id = cloudflare_zero_trust_access_policy.looped[k].id, precedence = v.precedence } }`)
	assert.Contains(t, m, "allow = { id = cloudflare_zero_trust_access_policy.allow.id, precedence = terraform_data.tf_migrate_access_policies.output.allow.precedence }")
	assert.Contains(t, m, `= [for s in sort([for k, v in local.tf_migrate_app_policies : format("%09d|%s", v.precedence, k)]) : local.tf_migrate_app_policies[substr(s, 10, -1)]]`)
	assert.NotContains(t, m, "cloudflare_zero_trust_access_policy.counted", "unsupported policy is not attached")
	assert.NotContains(t, m, "tf_migrate_other_policies", "its application is left alone")
	assert.Contains(t, m, "from = cloudflare_access_policy.counted", "unsupported policy keeps the manual removed {}")
}

func TestAccessPolicyPhasedSteps_LocalModule(t *testing.T) {
	root := t.TempDir()
	mod := filepath.Join(root, "modules", "access")
	require.NoError(t, os.MkdirAll(mod, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.tf"), []byte(`module "access" {
  source     = "./modules/access"
  account_id = var.account_id
}
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "main.tf"), []byte(phasedFixture), 0600))
	cfg := config{configDir: root, recursive: true, sourceVersion: "v4", targetVersion: "v5", targetProviderVersion: "5.29.0", skipVersionCheck: true}
	read := func(p string) string { b, err := os.ReadFile(p); require.NoError(t, err); return string(b) }
	caller := filepath.Join(root, "tf-migrate-access-policies-access.tf")

	runWithAnswers(t, cfg)
	assert.Equal(t, accesspolicyphase.Recorded, accesspolicyphase.StateOf(mod))
	assert.Contains(t, read(caller), "value = module.access.tf_migrate_access_policies")
	assert.NotContains(t, read(caller), "import {")

	out := runWithAnswers(t, cfg)
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(mod))
	assert.Contains(t, read(filepath.Join(mod, "main.tf")), "# tf-migrate: resource \"cloudflare_access_policy\" \"allow\"")
	assert.Contains(t, out, "output -json tf_migrate_access_policies_access")

	runWithAnswers(t, cfg)
	m := read(filepath.Join(mod, "main.tf"))
	assert.NotContains(t, m, "import {", "imports are only allowed in the root module")
	assert.Contains(t, m, `resource "cloudflare_zero_trust_access_policy" "allow"`)
	assert.NotContains(t, m, "from = cloudflare_access_policy.allow", "no moved block for an imported policy")
	c := read(caller)
	assert.Contains(t, c, "to = module.access.cloudflare_zero_trust_access_policy.allow")
	assert.Contains(t, c, `id = "${module.access.tf_migrate_access_policies.allow.account_id}/${module.access.tf_migrate_access_policies.allow.id}"`)
	assert.Contains(t, c, "for_each = module.access.tf_migrate_access_policies.looped")
	assert.Contains(t, c, "to       = module.access.cloudflare_zero_trust_access_policy.looped[each.key]")
}

func TestFindLocalModuleCalls_Unsupported(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a", "b", "c", "c/inner"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, d), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.tf"), []byte(`module "a1" { source = "./a" }
module "a2" { source = "./a" }
module "b" {
  source   = "./b"
  for_each = toset(["x"])
}
module "c" { source = "./c" }
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "c", "main.tf"), []byte(`module "inner" { source = "./inner" }`+"\n"), 0600))
	calls, unsupported, err := findLocalModuleCalls(config{configDir: root, recursive: true})
	require.NoError(t, err)
	assert.Contains(t, unsupported[filepath.Join(root, "a")], "more than once")
	assert.Contains(t, unsupported[filepath.Join(root, "b")], "count or for_each")
	assert.Contains(t, unsupported[filepath.Join(root, "c", "inner")], "nested module")
	assert.Equal(t, "c", calls[filepath.Join(root, "c")].Name)
}
