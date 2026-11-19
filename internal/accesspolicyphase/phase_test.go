package accesspolicyphase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		"for_each": `resource "cloudflare_access_policy" "p" {` + "\n" + `for_each = {}` + "\n" + `application_id = cloudflare_access_application.app.id` + "\n" + `account_id = "a"` + "\n" + `precedence = 1
}`,
		"zone_id": `resource "cloudflare_access_policy" "p" {` + "\n" + `application_id = cloudflare_access_application.app.id` + "\n" + `zone_id = "z"` + "\n" + `precedence = 1
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
  "b": {"id": "p2", "application_id": "app1", "account_id": "A", "precedence": 2}
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
	assert.Equal(t, 1, puts)

	again := c.MakeReusable(rec, false)
	assert.Equal(t, "already reusable", again[0].Result)
	assert.Equal(t, 1, puts, "re-run converts nothing")

	_, err = ParseRecord([]byte(`{"a": {"id": "p1"}}`))
	assert.Error(t, err)
}
