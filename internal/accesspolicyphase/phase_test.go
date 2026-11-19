package accesspolicyphase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func block(t *testing.T, src string) *hclwrite.Block {
	t.Helper()
	f, diags := hclwrite.ParseConfig([]byte(src), "t.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	return f.Body().Blocks()[0]
}

func TestInspect(t *testing.T) {
	ok := `resource "cloudflare_access_policy" "p" {
  application_id = cloudflare_access_application.app.id
  account_id     = var.account_id
  precedence     = 2
}`
	p, supported, reason := Inspect(block(t, ok))
	require.True(t, supported, reason)
	assert.Equal(t, Policy{Type: "cloudflare_access_policy", Name: "p", App: "app", AccountID: "var.account_id", Precedence: "2"}, p)

	for name, src := range map[string]string{
		"count": `resource "cloudflare_access_policy" "p" {` + "\n" + `count = 1` + "\n" + `application_id = cloudflare_access_application.app.id` + "\n" + `account_id = "a"` + "\n" + `precedence = 1
}`,
		"no scope": `resource "cloudflare_access_policy" "p" {` + "\n" + `application_id = cloudflare_access_application.app.id` + "\n" + `precedence = 1
}`,
		"literal id": `resource "cloudflare_access_policy" "p" {` + "\n" + `application_id = "abc"` + "\n" + `account_id = "a"` + "\n" + `precedence = 1
}`,
		"module ref": `resource "cloudflare_access_policy" "p" {` + "\n" + `application_id = module.m.app_id` + "\n" + `account_id = "a"` + "\n" + `precedence = 1
}`,
	} {
		_, supported, reason := Inspect(block(t, src))
		assert.False(t, supported, name)
		assert.NotEmpty(t, reason, name)
	}

	zp, supported, reason := Inspect(block(t, `resource "cloudflare_access_policy" "z" {
  application_id = cloudflare_access_application.app.id
  zone_id        = var.zone_id
  precedence     = 1
}`))
	require.True(t, supported, reason)
	assert.Equal(t, "var.zone_id", zp.ZoneID)
	assert.Equal(t, "terraform_data.tf_migrate_access_policies.output.z.account_id", zp.AccountID)

	fp, supported, reason := Inspect(block(t, `resource "cloudflare_access_policy" "fe" {
  for_each       = var.policies
  application_id = cloudflare_access_application.app.id
  zone_id        = var.zone_id
  precedence     = each.value
}`))
	require.True(t, supported, reason)
	assert.True(t, fp.ForEach)
	assert.Equal(t, "terraform_data.tf_migrate_access_policies.output.fe[each.key].account_id", fp.AccountID)
	assert.Equal(t, `"${each.value.account_id}/${each.value.id}"`, ImportID(fp))
	rec := string(RenderRecord([]Policy{fp}))
	assert.Contains(t, rec, "fe = { for k, i in cloudflare_access_policy.fe : k => {")
	assert.Contains(t, rec, "zone_id        = i.zone_id")
	assert.Contains(t, rec, "account_id     = data.cloudflare_zone.tf_migrate_access_policies_0.account_id")

	_, supported, _ = Inspect(block(t, `resource "cloudflare_access_policy" "fe" {
  for_each       = var.policies
  application_id = cloudflare_access_application.app.id
  zone_id        = each.value.zone
  precedence     = 1
}`))
	assert.False(t, supported, "per-instance zone_id can't be looked up once")

	_, supported, reason = Inspect(block(t, `resource "cloudflare_access_policy" "p" { account_id = "a" }`))
	assert.False(t, supported)
	assert.Empty(t, reason, "account-level policies are not app-scoped")
}

func TestStateOfAndRender(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, NoRecord, StateOf(dir))

	ps := []Policy{{Type: "cloudflare_access_policy", Name: "a", App: "app", AccountID: "var.account_id", Precedence: "1"}}
	rec := RenderRecord(ps)
	require.NoError(t, os.WriteFile(filepath.Join(dir, RecordFile), rec, 0600))
	assert.Equal(t, Recorded, StateOf(dir))
	assert.Contains(t, string(rec), "id             = cloudflare_access_policy.a.id")
	assert.Contains(t, string(rec), "value = local."+Name)

	zoned := RenderRecord(append(ps,
		Policy{Type: "cloudflare_access_policy", Name: "z1", ZoneID: "var.zone_id", Precedence: "1"},
		Policy{Type: "cloudflare_access_policy", Name: "z2", ZoneID: "var.zone_id", Precedence: "2"}))
	assert.Equal(t, 1, strings.Count(string(zoned), `data "cloudflare_zone"`), "one lookup per zone:\n%s", zoned)
	assert.Contains(t, string(zoned), "account_id     = data.cloudflare_zone.tf_migrate_access_policies_0.account_id")
	assert.Contains(t, string(zoned), "zone_id        = cloudflare_access_policy.z2.zone_id")
	assert.Contains(t, string(zoned), "account_id     = cloudflare_access_policy.a.account_id")

	require.NoError(t, os.WriteFile(filepath.Join(dir, RecordFile), RenderFrozen(), 0600))
	assert.Equal(t, Frozen, StateOf(dir))
}

func TestForAppOrdersByPrecedence(t *testing.T) {
	dir := t.TempDir()
	src := ""
	for _, p := range []struct{ name, prec string }{{"b", "2"}, {"a", "1"}, {"c", "3"}} {
		src += `resource "cloudflare_access_policy" "` + p.name + `" {
  application_id = cloudflare_access_application.app.id
  account_id     = "acct"
  precedence     = ` + p.prec + "\n}\n"
	}
	src += `resource "cloudflare_access_policy" "other" {
  application_id = cloudflare_access_application.other.id
  account_id     = "acct"
  precedence     = 1
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p.tf"), []byte(src), 0600))
	ResetCache()
	defer ResetCache()
	var names []string
	for _, p := range ForApp(dir, "app") {
		names = append(names, p.Name)
	}
	assert.Equal(t, []string{"a", "b", "c"}, names)
}

func TestImportID(t *testing.T) {
	assert.Equal(t, `"${var.account_id}/${terraform_data.tf_migrate_access_policies.output.p.id}"`,
		ImportID(Policy{Name: "p", AccountID: "var.account_id"}))
	assert.Equal(t, `"acct/${terraform_data.tf_migrate_access_policies.output.p.id}"`,
		ImportID(Policy{Name: "p", AccountID: `"acct"`}))
}

func TestMakeReusable(t *testing.T) {
	reusable := map[string]bool{"p2": true}
	puts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer t", r.Header.Get("Authorization"))
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		switch {
		case r.Method == http.MethodGet && len(parts) == 5: // accounts/A/access/policies/ID
			if !reusable[parts[4]] {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": parts[4], "reusable": true}})
		case r.Method == http.MethodPut && parts[len(parts)-1] == "make_reusable":
			puts++
			id := parts[len(parts)-2]
			if parts[0] == "zones" {
				assert.Equal(t, "Z", parts[1])
			} else {
				assert.Equal(t, "A", parts[1])
			}
			assert.Equal(t, "app1", parts[4])
			reusable[id] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"id": id, "reusable": true}})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	rec, err := ParseRecord([]byte(`{
  "a": {"id": "p1", "application_id": "app1", "account_id": "A", "precedence": 1},
  "b": {"id": "p2", "application_id": "app1", "account_id": "A", "precedence": 2},
  "c": {"id": "p3", "application_id": "app1", "account_id": "A", "zone_id": "Z", "precedence": 3}
}`))
	require.NoError(t, err)
	c := &Client{BaseURL: srv.URL, Auth: func(r *http.Request) { r.Header.Set("Authorization", "Bearer t") }}

	dry := c.MakeReusable(rec, true)
	assert.Equal(t, "would convert", dry[0].Result)
	assert.Equal(t, "already reusable", dry[1].Result)
	assert.Equal(t, 0, puts)

	out := c.MakeReusable(rec, false)
	assert.Equal(t, "converted", out[0].Result)
	assert.NoError(t, out[0].Err)
	assert.Equal(t, "already reusable", out[1].Result)
	assert.Equal(t, "converted", out[2].Result, "zone app policy, through the zone endpoint")
	assert.Equal(t, 2, puts)

	again := c.MakeReusable(rec, false)
	assert.Equal(t, "already reusable", again[0].Result)
	assert.Equal(t, 2, puts, "re-run converts nothing")

	nested, err := ParseRecord([]byte(`{"fe": {"x": {"id": "p9", "application_id": "app1", "account_id": "A"}}}`))
	require.NoError(t, err)
	assert.Equal(t, "p9", nested[`fe["x"]`].ID)

	_, err = ParseRecord([]byte(`{"a": {"id": "p1"}}`))
	assert.Error(t, err)
}

func TestMakeReusableErrors(t *testing.T) {
	// per policy ID: what the PUT does, and whether GET reports it reusable after
	mode := map[string]string{}
	reusable := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if r.Method == http.MethodGet {
			id := parts[4]
			switch {
			case mode[id] == "get500":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"success":false,"errors":[{"message":"boom"},{"message":"again"}]}`))
			case reusable[id]:
				_, _ = w.Write([]byte(`{"success":true,"result":{"id":"` + id + `","reusable":true}}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}
		id := parts[len(parts)-2]
		switch mode[id] {
		case "conflict": // converted by someone else meanwhile
			reusable[id] = true
			w.WriteHeader(http.StatusConflict)
		case "noop": // 200, but the policy stays app-scoped
		case "otherid":
			_, _ = w.Write([]byte(`{"success":true,"result":{"id":"other","reusable":true}}`))
		case "forbidden":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`not json`))
		}
	}))
	defer srv.Close()
	mode = map[string]string{"p1": "conflict", "p2": "noop", "p3": "otherid", "p4": "forbidden", "p5": "get500"}
	rec := map[string]RecordedPolicy{}
	for i, id := range []string{"p1", "p2", "p3", "p4", "p5"} {
		rec[string(rune('a'+i))] = RecordedPolicy{ID: id, ApplicationID: "app", AccountID: "A"}
	}
	out := (&Client{BaseURL: srv.URL}).MakeReusable(rec, false)
	require.Len(t, out, 5)
	assert.Equal(t, "already reusable", out[0].Result, "409 then verified")
	assert.NoError(t, out[0].Err)
	assert.EqualError(t, out[1].Err, "verify: policy is still not reusable")
	assert.EqualError(t, out[2].Err, "API returned a different policy ID other")
	assert.EqualError(t, out[3].Err, "HTTP 403")
	assert.EqualError(t, out[4].Err, "HTTP 500: boom; again")
	for _, o := range out[1:] {
		assert.Equal(t, "failed", o.Result)
	}
}

func TestMakeReusableEscapesPath(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_ = (&Client{BaseURL: srv.URL}).MakeReusable(map[string]RecordedPolicy{"a": {ID: "p/../x", ApplicationID: "app", AccountID: "A"}}, true)
	assert.Equal(t, "/accounts/A/access/policies/p%2F..%2Fx", got)
}

func writeTF(t *testing.T, dir, src string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(src), 0600))
}

func policySrc(typ, name, app, extra string, prec string) string {
	return `resource "` + typ + `" "` + name + `" {
  application_id = ` + app + `
  account_id     = "a"
  precedence     = ` + prec + `
` + extra + `}
`
}

func TestScanAllOrNothingPerApp(t *testing.T) {
	const apps = `resource "cloudflare_access_application" "one" {}
resource "cloudflare_access_application" "two" {}
resource "cloudflare_access_application" "three" {
  policies = ["x"]
}
`
	names := func(ps []Policy) []string {
		var out []string
		for _, p := range ps {
			out = append(out, p.Name)
		}
		return out
	}
	t.Run("mixed app goes manual, other app phased", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, apps+
			policySrc("cloudflare_access_policy", "ok1", "cloudflare_access_application.one.id", "", "1")+
			policySrc("cloudflare_access_policy", "counted", "cloudflare_access_application.one.id", "  count = 1\n", "2")+
			policySrc("cloudflare_access_policy", "ok2", "cloudflare_access_application.two.id", "", "1"))
		ps, skipped, err := Scan(dir)
		require.NoError(t, err)
		assert.Equal(t, []string{"ok2"}, names(ps))
		assert.Len(t, skipped, 2)
		assert.Contains(t, strings.Join(skipped, "\n"), "cloudflare_access_policy.ok1: its application (one) also has cloudflare_access_policy.counted")
	})
	t.Run("app that already sets policies", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, apps+policySrc("cloudflare_access_policy", "p", "cloudflare_access_application.three.id", "", "1"))
		ps, skipped, _ := Scan(dir)
		assert.Empty(t, ps)
		assert.Equal(t, []string{"cloudflare_access_policy.p: its application (three) already sets policies"}, skipped)
	})
	t.Run("same name under both v4 types", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, apps+
			policySrc("cloudflare_access_policy", "dup", "cloudflare_access_application.one.id", "", "1")+
			policySrc("cloudflare_zero_trust_access_policy", "dup", "cloudflare_access_application.two.id", "", "1"))
		ps, skipped, _ := Scan(dir)
		assert.Empty(t, ps)
		assert.Len(t, skipped, 2)
		_, found := Find(dir, "cloudflare_access_policy", "dup")
		assert.False(t, found)
	})
	t.Run("unresolvable application_id only affects itself", func(t *testing.T) {
		// The API check (CheckApps) catches it if it is on a phased app.
		dir := t.TempDir()
		writeTF(t, dir, apps+
			policySrc("cloudflare_access_policy", "ok", "cloudflare_access_application.one.id", "", "1")+
			policySrc("cloudflare_access_policy", "lit", `"0000-literal"`, "", "1"))
		ps, skipped, _ := Scan(dir)
		assert.Equal(t, []string{"ok"}, names(ps))
		assert.Equal(t, []string{"cloudflare_access_policy.lit: application_id is not a reference to an application resource in the same module"}, skipped)
	})
	t.Run("indexed application blocks only that application", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, apps+
			policySrc("cloudflare_access_policy", "ok", "cloudflare_access_application.one.id", "", "1")+
			policySrc("cloudflare_access_policy", "idx", "cloudflare_access_application.two[each.key].id", "  for_each = toset([\"a\"])\n", "1")+
			policySrc("cloudflare_access_policy", "two_ok", "cloudflare_access_application.two.id", "", "2")+
			`resource "cloudflare_access_policy" "fe" {
  for_each       = cloudflare_access_application.three
  application_id = each.value.id
  account_id     = "a"
  precedence     = 1
}
`)
		ps, skipped, _ := Scan(dir)
		assert.Equal(t, []string{"ok"}, names(ps))
		j := strings.Join(skipped, "\n")
		assert.Contains(t, j, "cloudflare_access_policy.idx: application_id refers to an instance of an application with count or for_each")
		assert.Contains(t, j, "cloudflare_access_policy.two_ok: its application (two) also has cloudflare_access_policy.idx")
		assert.Contains(t, j, "cloudflare_access_policy.fe: ")
	})
	t.Run("skipped with --skip-app", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, apps+
			policySrc("cloudflare_access_policy", "ok", "cloudflare_access_application.one.id", "", "1")+
			policySrc("cloudflare_access_policy", "s", "cloudflare_access_application.two.id", "", "1"))
		require.NoError(t, WriteCheck(dir, CheckResult{Checked: map[string]string{"ok": "p1"}, SkippedApps: []string{"app2"}, Skipped: map[string]string{"s": "p2"}}))
		ps, skipped, _ := Scan(dir)
		assert.Equal(t, []string{"ok"}, names(ps))
		assert.Equal(t, []string{"cloudflare_access_policy.s: its application was skipped with --skip-app"}, skipped)
	})
	t.Run("Find matches the type", func(t *testing.T) {
		dir := t.TempDir()
		writeTF(t, dir, apps+policySrc("cloudflare_access_policy", "p", "cloudflare_access_application.one.id", "", "1"))
		ResetCache()
		_, ok := Find(dir, "cloudflare_access_policy", "p")
		assert.True(t, ok)
		_, ok = Find(dir, "cloudflare_zero_trust_access_policy", "p")
		assert.False(t, ok)
	})
}

func TestReferences(t *testing.T) {
	dir := t.TempDir()
	writeTF(t, dir, `resource "cloudflare_access_application" "one" {}
`+policySrc("cloudflare_access_policy", "p", "cloudflare_access_application.one.id", "", "1")+`
output "pid" {
  value = cloudflare_access_policy.p.id
}
# value = cloudflare_access_policy.p.id
output "other" {
  value = cloudflare_access_policy.p2.id
}
moved {
  from = cloudflare_access_policy.old
  to   = cloudflare_access_policy.p
}
import {
  to = cloudflare_access_policy.p
  id = "x"
}
`)
	ps, _, err := Scan(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"main.tf:9 (cloudflare_access_policy.p)", "main.tf:20 (cloudflare_access_policy.p, import block)"}, References(dir, ps), "moved blocks are ignored")
}

func TestCheckApps(t *testing.T) {
	attached := map[string][]string{"app1": {"p1", "p2", "p3", "p4", "p5"}, "zapp": {"z1", "zx"}}
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 5 { // the app, for its name
			_, _ = w.Write([]byte(`{"success":true,"result":{"name":"App ` + parts[4] + `"}}`))
			return
		}
		ids := attached[parts[4]]
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		var res []map[string]any
		for i := (page - 1) * per; i < len(ids) && i < page*per; i++ {
			res = append(res, map[string]any{"id": ids[i], "name": "N" + ids[i], "decision": "deny", "precedence": i + 1})
		}
		pages := (len(ids) + per - 1) / per
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": res, "result_info": map[string]any{"page": page, "per_page": per, "total_pages": pages, "total_count": len(ids)}})
	}))
	defer srv.Close()
	rec := map[string]RecordedPolicy{}
	for _, id := range []string{"p1", "p2", "p3", "p4", "p5"} {
		rec[id] = RecordedPolicy{ID: id, ApplicationID: "app1", AccountID: "A"}
	}
	rec["z"] = RecordedPolicy{ID: "z1", ApplicationID: "zapp", AccountID: "A", ZoneID: "Z"}
	rec["gone"] = RecordedPolicy{ID: "z2", ApplicationID: "zapp", AccountID: "A", ZoneID: "Z"}
	rec["s"] = RecordedPolicy{ID: "s1", ApplicationID: "skipme", AccountID: "A"}

	out := (&Client{BaseURL: srv.URL, PerPage: 2}).CheckApps(rec, map[string]bool{"skipme": true})
	require.Len(t, out, 3)
	assert.True(t, out[0].Complete(), "5 policies over 3 pages: %+v", out[0])
	assert.True(t, out[1].Skipped)
	z := out[2]
	assert.False(t, z.Complete())
	assert.Equal(t, "App zapp", z.AppName)
	assert.Equal(t, []string{"gone"}, z.Missing)
	require.Len(t, z.Extra, 1)
	assert.Equal(t, "zx", z.Extra[0].ID)
	d := z.Describe()
	assert.Contains(t, d, `App zapp (zapp): 2 policies in Cloudflare, 2 in your Terraform record`)
	assert.Contains(t, d, `not in Terraform: "Nzx" (deny, precedence 2, id zx)`)
	assert.Contains(t, d, "gone is in the record but not attached in Cloudflare")
	assert.Contains(t, paths, "/zones/Z/access/apps/zapp/policies", "zone apps list through the zone endpoint")
	for _, p := range paths {
		assert.NotContains(t, p, "skipme")
	}
}
