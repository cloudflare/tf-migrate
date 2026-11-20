package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// checkAll writes the result of a passing make-access-policies-reusable
// --check for every policy Scan finds in dir.
func checkAll(t *testing.T, dir string) {
	t.Helper()
	ps, _, err := accesspolicyphase.Scan(dir)
	require.NoError(t, err)
	r := accesspolicyphase.CheckResult{Checked: map[string]string{}}
	for _, p := range ps {
		r.Checked[p.Name] = "id-" + p.Name
	}
	require.NoError(t, accesspolicyphase.WriteCheck(dir, r))
}

// convertAll marks every checked policy converted, as the convert run does.
func convertAll(t *testing.T, dir string) {
	t.Helper()
	r, ok, err := accesspolicyphase.ReadCheck(dir)
	require.NoError(t, err)
	require.True(t, ok)
	r.Converted = map[string]string{}
	for k, v := range r.Checked {
		r.Converted[k] = v
	}
	require.NoError(t, accesspolicyphase.WriteCheck(dir, r))
}

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
	assert.Contains(t, out, "step 1 of 4")
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

	assert.Contains(t, out, "make-access-policies-reusable --check --ids")

	// Step 2 waits for the API check.
	out = runWithAnswers(t, cfg)
	assert.Contains(t, out, "step 2 of 4 needs the check first")
	assert.Equal(t, accesspolicyphase.Recorded, accesspolicyphase.StateOf(dir))
	assert.Equal(t, phasedFixture, read(main))

	// With the check: the supported policies are removed without destroy
	// and the record frozen.
	checkAll(t, dir)
	out = runWithAnswers(t, cfg)
	assert.Contains(t, out, "make-access-policies-reusable --ids")
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(dir))
	m := read(main)
	assert.Contains(t, m, "# tf-migrate: resource \"cloudflare_access_policy\" \"allow\"")
	assert.Contains(t, m, "# tf-migrate: resource \"cloudflare_access_policy\" \"deny\"")
	assert.Equal(t, 4, strings.Count(m, "destroy = false"))
	assert.Contains(t, m, "resource \"cloudflare_access_policy\" \"counted\"", "unsupported policy is left for the manual path")

	// Step 4 waits for the convert run's marks.
	out = runWithAnswers(t, cfg)
	assert.Contains(t, out, "not converted to reusable yet: allow, by_zone, deny, looped")
	assert.Contains(t, read(main), "# tf-migrate:")

	// Step 4: v5 policies imported by recorded ID, listed on the app in
	// precedence order, no moved block for them.
	convertAll(t, dir)
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

	checkAll(t, mod)
	out := runWithAnswers(t, cfg)
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(mod))
	assert.Contains(t, read(filepath.Join(mod, "main.tf")), "# tf-migrate: resource \"cloudflare_access_policy\" \"allow\"")
	assert.Contains(t, out, "output -json tf_migrate_access_policies_access")
	assert.Contains(t, out, "--config-dir "+mod)

	convertAll(t, mod)
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

func phasedCfg(dir string) config {
	return config{configDir: dir, sourceVersion: "v4", targetVersion: "v5", targetProviderVersion: "5.29.0", skipVersionCheck: true}
}

const twoPolicies = `provider "cloudflare" {}

resource "cloudflare_access_application" "app" {
  account_id = var.account_id
  name       = "App"
  domain     = "app.example.com"
  type       = "self_hosted"
}

resource "cloudflare_access_policy" "first" {
  application_id = cloudflare_access_application.app.id
  account_id     = var.account_id
  name           = "First"
  decision       = "allow"
  precedence     = var.prec_first

  include {
    everyone = true
  }
}

resource "cloudflare_access_policy" "second" {
  application_id = cloudflare_access_application.app.id
  account_id     = var.account_id
  name           = "Second"
  decision       = "deny"
  precedence     = 1

  include {
    everyone = true
  }
}
`

func TestAccessPolicyPhased_NonLiteralPrecedenceSorts(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(main, []byte(twoPolicies), 0600))
	cfg := phasedCfg(dir)
	runWithAnswers(t, cfg)
	checkAll(t, dir)
	runWithAnswers(t, cfg)
	convertAll(t, dir)
	runWithAnswers(t, cfg)
	b, err := os.ReadFile(main)
	require.NoError(t, err)
	m := string(b)
	assert.Contains(t, m, `= [for s in sort([for k, v in local.tf_migrate_app_policies : format("%09d|%s", v.precedence, k)]) : local.tf_migrate_app_policies[substr(s, 10, -1)]]`)
	assert.Contains(t, m, "first  = { id = cloudflare_zero_trust_access_policy.first.id, precedence = terraform_data.tf_migrate_access_policies.output.first.precedence }")

	// A re-run over the migrated output leaves the sorted list alone and
	// doesn't warn about it.
	out := runWithAnswers(t, cfg)
	b2, _ := os.ReadFile(main)
	assert.Equal(t, m, string(b2))
	assert.NotContains(t, out, "policies not converted")
}

func TestAccessPolicyPhased_ReferencesBlockStepTwo(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tf")
	src := strings.Replace(twoPolicies, "var.prec_first", "2", 1) + `
output "first_id" {
  value = cloudflare_access_policy.first.id
}
`
	require.NoError(t, os.WriteFile(main, []byte(src), 0600))
	cfg := phasedCfg(dir)
	out := runWithAnswers(t, cfg)
	assert.Contains(t, out, "main.tf:35 (cloudflare_access_policy.first)")
	assert.Equal(t, accesspolicyphase.Recorded, accesspolicyphase.StateOf(dir))

	checkAll(t, dir)
	out = runWithAnswers(t, cfg)
	assert.Contains(t, out, "Remove these references, then re-run tf-migrate for step 2.")
	assert.Equal(t, accesspolicyphase.Recorded, accesspolicyphase.StateOf(dir), "step 2 waits")
	b, _ := os.ReadFile(main)
	assert.Equal(t, src, string(b))

	require.NoError(t, os.WriteFile(main, []byte(strings.Replace(twoPolicies, "var.prec_first", "2", 1)), 0600))
	runWithAnswers(t, cfg)
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(dir))
}

func TestAccessPolicyPhased_ModuleAloneTakesManualPath(t *testing.T) {
	dir := t.TempDir()
	src := strings.Replace(twoPolicies, "provider \"cloudflare\" {}\n", "", 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(src), 0600))
	cfg := phasedCfg(dir)
	cfg.skipPhaseCheck = false
	out := runWithAnswers(t, cfg)
	assert.Contains(t, out, "looks like a module called from outside --config-dir")
	assert.NoFileExists(t, filepath.Join(dir, accesspolicyphase.RecordFile))
	assert.Contains(t, out, "ACTION REQUIRED")

	// --root-module overrides the heuristic for --config-dir.
	dirR := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dirR, "main.tf"), []byte(src), 0600))
	cfgR := phasedCfg(dirR)
	cfgR.rootModule = true
	assert.Contains(t, runWithAnswers(t, cfgR), "step 1 of 4")

	// A lock file is enough to count as a root.
	dir2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "main.tf"), []byte(src), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, ".terraform.lock.hcl"), nil, 0600))
	out = runWithAnswers(t, phasedCfg(dir2))
	assert.Contains(t, out, "step 1 of 4")
}

func TestAccessPolicyPhased_DryRunAtStepFour(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tf")
	require.NoError(t, os.WriteFile(main, []byte(strings.Replace(twoPolicies, "var.prec_first", "2", 1)), 0600))
	cfg := phasedCfg(dir)
	runWithAnswers(t, cfg)
	checkAll(t, dir)
	runWithAnswers(t, cfg)
	convertAll(t, dir)
	before, _ := os.ReadFile(main)
	cfg.dryRun = true
	out := runWithAnswers(t, cfg)
	assert.Contains(t, out, "(dry run) would restore")
	after, _ := os.ReadFile(main)
	assert.Equal(t, string(before), string(after))
	assert.Equal(t, accesspolicyphase.Frozen, accesspolicyphase.StateOf(dir))
}

// fakeAccessAPI serves the Access endpoints make-access-policies-reusable
// calls. apps maps application ID -> attached policy IDs.
type fakeAccessAPI struct {
	apps     map[string][]string
	reusable map[string]bool
	fail     map[string]int // application ID -> status for its list call
	puts     int
}

func (f *fakeAccessAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/") // accounts A access ...
	switch {
	case r.Header.Get("Authorization") != "Bearer tok":
		w.WriteHeader(http.StatusForbidden)
	case r.Method == http.MethodGet && len(parts) == 6 && parts[3] == "apps" && parts[5] == "policies" && f.fail[parts[4]] != 0:
		w.WriteHeader(f.fail[parts[4]])
	case r.Method == http.MethodGet && len(parts) == 6 && parts[3] == "apps" && parts[5] == "policies":
		var res []map[string]any
		for i, id := range f.apps[parts[4]] {
			res = append(res, map[string]any{"id": id, "name": "Policy " + id, "decision": "allow", "precedence": i + 1})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": res, "result_info": map[string]any{"page": 1, "total_pages": 1, "total_count": len(res)}})
	case r.Method == http.MethodGet && len(parts) == 5 && parts[3] == "apps":
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"name": "app " + parts[4]}})
	case r.Method == http.MethodGet && len(parts) == 5 && parts[3] == "policies":
		if !f.reusable[parts[4]] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": parts[4], "reusable": true}})
	case r.Method == http.MethodPut && parts[len(parts)-1] == "make_reusable":
		f.puts++
		f.reusable[parts[len(parts)-2]] = true
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": parts[len(parts)-2], "reusable": true}})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestMakeAccessPoliciesReusableCommand(t *testing.T) {
	api := &fakeAccessAPI{apps: map[string][]string{"app1": {"p1", "p2"}, "app2": {"p3", "p4"}}, reusable: map[string]bool{}, fail: map[string]int{}}
	srv := httptest.NewServer(api)
	defer srv.Close()
	dir := t.TempDir()
	record := filepath.Join(dir, accesspolicyphase.RecordFile)
	recorded := "resource \"terraform_data\" \"" + accesspolicyphase.Name + "\" {\n  input = {}\n}\n"
	require.NoError(t, os.WriteFile(record, []byte(recorded), 0600))
	const ids = `{"a": {"id": "p1", "application_id": "app1", "account_id": "A"},
 "b": {"id": "p2", "application_id": "app1", "account_id": "A"},
 "c": {"id": "p3", "application_id": "app2", "account_id": "A"}}`
	const idsWithoutApp2 = `{"a": {"id": "p1", "application_id": "app1", "account_id": "A"},
 "b": {"id": "p2", "application_id": "app1", "account_id": "A"}}`
	run := func(ids string, args ...string) (string, error) {
		t.Helper()
		f := filepath.Join(t.TempDir(), "ids.json")
		require.NoError(t, os.WriteFile(f, []byte(ids), 0600))
		cmd := newMakeAccessPoliciesReusableCommand(&config{configDir: dir})
		var buf strings.Builder
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs(append([]string{"--ids", f, "--api-url", srv.URL}, args...))
		err := cmd.Execute()
		return buf.String(), err
	}
	check := func() accesspolicyphase.CheckResult {
		t.Helper()
		r, ok, err := accesspolicyphase.ReadCheck(dir)
		require.NoError(t, err)
		require.True(t, ok)
		return r
	}
	freeze := func() { require.NoError(t, os.WriteFile(record, accesspolicyphase.RenderFrozen(), 0600)) }

	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_EMAIL", "")
	t.Setenv("CLOUDFLARE_API_KEY", "")
	_, err := run(ids, "--check")
	assert.ErrorContains(t, err, "set CLOUDFLARE_API_TOKEN")

	t.Setenv("CLOUDFLARE_API_TOKEN", "tok")
	_, err = run(`{}`, "--check")
	assert.ErrorContains(t, err, "has no policies")

	_, err = run(ids)
	assert.ErrorContains(t, err, "Run with --check after the step 1 apply")

	// An API error is "could not check", not "incomplete", and leaves no file.
	api.fail["app2"] = http.StatusForbidden
	out, err := run(ids, "--check")
	assert.ErrorContains(t, err, "1 application(s) could not be checked")
	assert.NotContains(t, err.Error(), "incomplete")
	assert.Contains(t, out, "app2: could not check: HTTP 403")
	assert.NotContains(t, out, "--skip-app <id>")
	assert.NotContains(t, out, "Usage:", "no cobra usage dump")
	assert.NoFileExists(t, filepath.Join(dir, accesspolicyphase.CheckFile))
	delete(api.fail, "app2")

	// app2 has p4 in Cloudflare, not in the record: incomplete, named.
	out, err = run(ids, "--check")
	assert.ErrorContains(t, err, "1 application(s) incomplete")
	assert.Contains(t, out, `app app2 (app2): 2 policies in Cloudflare, 1 in your Terraform record`)
	assert.Contains(t, out, `not in Terraform: "Policy p4" (allow, precedence 2, id p4)`)
	assert.Contains(t, out, "--skip-app <id>")
	assert.Equal(t, map[string]string{"a": "p1", "b": "p2"}, check().Checked)
	assert.Equal(t, 0, api.puts)

	// Deliberately skipped: the check passes, app2 stays manual.
	out, err = run(ids, "--check", "--skip-app", "app2")
	require.NoError(t, err, out)
	r := check()
	assert.Equal(t, []string{"app2"}, r.SkippedApps)
	assert.Equal(t, map[string]string{"c": "p3"}, r.Skipped)

	// Step 2 rewrites the record without app2; the next check (with or
	// without --skip-app again) keeps the skip.
	out, err = run(idsWithoutApp2, "--check")
	require.NoError(t, err, out)
	r = check()
	assert.Equal(t, []string{"app2"}, r.SkippedApps, "skip carried forward")
	assert.Equal(t, map[string]string{"c": "p3"}, r.Skipped)
	assert.Equal(t, map[string]string{"a": "p1", "b": "p2"}, r.Checked)

	// After step 2 (record frozen), no new skips, with or without --check.
	freeze()
	_, err = run(idsWithoutApp2, "--skip-app", "app1")
	assert.ErrorContains(t, err, "can only be skipped with --check, before step 2")
	_, err = run(idsWithoutApp2, "--check", "--skip-app", "app1")
	assert.ErrorContains(t, err, "can only be skipped with --check, before step 2")

	// Convert: the complete application, and the marks for step 4.
	out, err = run(idsWithoutApp2)
	require.NoError(t, err, out)
	assert.Equal(t, 2, api.puts)
	assert.False(t, api.reusable["p3"], "skipped application is not converted")
	assert.Equal(t, map[string]string{"a": "p1", "b": "p2"}, check().Converted)

	// A transient error on a later --check keeps Checked and the marks.
	api.fail["app1"] = http.StatusInternalServerError
	_, err = run(idsWithoutApp2, "--check")
	assert.ErrorContains(t, err, "could not be checked")
	r = check()
	assert.Equal(t, map[string]string{"a": "p1", "b": "p2"}, r.Checked)
	assert.Equal(t, map[string]string{"a": "p1", "b": "p2"}, r.Converted)
	delete(api.fail, "app1")

	// A policy attached after step 2: --check and convert keep Checked and
	// drop the marks, so step 4 waits.
	api.apps["app1"] = append(api.apps["app1"], "p9")
	out, err = run(idsWithoutApp2, "--check")
	assert.ErrorContains(t, err, "1 application(s) incomplete")
	assert.Contains(t, out, "id p9")
	assert.Contains(t, out, "can't be skipped any more")
	assert.Contains(t, out, "0 of 2 checked policies on complete applications")
	r = check()
	assert.Equal(t, map[string]string{"a": "p1", "b": "p2"}, r.Checked, "step 2's decision is kept")
	assert.Empty(t, r.Converted)
	_, err = run(idsWithoutApp2)
	assert.ErrorContains(t, err, "1 application(s) incomplete")
	assert.Empty(t, check().Converted)

	// A check file lost after step 2 can't be recreated.
	saved, err := os.ReadFile(filepath.Join(dir, accesspolicyphase.CheckFile))
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(dir, accesspolicyphase.CheckFile)))
	for _, args := range [][]string{{"--check"}, nil} {
		_, err = run(idsWithoutApp2, args...)
		assert.ErrorContains(t, err, "restore it from git")
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, accesspolicyphase.CheckFile), saved, 0600))

	// Partial failure exits non-zero.
	api.apps["app1"] = []string{"p1", "p2"}
	api.reusable = map[string]bool{}
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/p2/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		api.ServeHTTP(w, r)
	})
	out, err = run(idsWithoutApp2)
	assert.ErrorContains(t, err, "1 of 2 policies failed")
	assert.Contains(t, out, "HTTP 403")
	assert.Equal(t, map[string]string{"a": "p1"}, check().Converted)
}

func TestPreflightClassifiesPhasedPolicies(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(phasedFixture), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "providers.tf"), []byte("provider \"cloudflare\" {}\n"), 0600))
	accesspolicyphase.ResetCache()
	report, err := runPreMigrationScan(newTestLogger(), phasedCfg(dir))
	require.NoError(t, err)
	class := map[string]resourceClassification{}
	for _, r := range report.Resources {
		class[r.ResourceName] = r.Class
	}
	assert.Equal(t, classAutoMigrated, class["allow"], "phased")
	assert.Equal(t, classAutoMigrated, class["looped"], "phased")
	assert.Equal(t, classManualIntervention, class["counted"], "manual: count")
}
