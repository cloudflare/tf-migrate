package zerotrustsettings

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"
)

func TestRestoreMigrationSettings(t *testing.T) {
	files := map[string][]byte{"main.tf": []byte(`
resource "cloudflare_zero_trust_organization" "example" {
  account_id = var.account_id
  deny_unmatched_requests = false
  deny_unmatched_requests_exempted_zone_names = []
}
resource "cloudflare_zero_trust_gateway_settings" "example" {
  account_id = var.account_id
  settings = { tls_decrypt = { enabled = false }, sandbox = null }
}
resource "cloudflare_zero_trust_device_settings" "example_proxy" {
  account_id = var.account_id
  gateway_proxy_enabled = true
  external_emergency_signal_enabled = false
}
`)}
	var snapshot Snapshot
	require.NoError(t, json.Unmarshal([]byte(`{
  "cloudflare_zero_trust_organization.example": {"success":true,"result":{
    "name":"do not copy", "deny_unmatched_requests":true,
    "deny_unmatched_requests_exempted_zone_names":["example.com"],
    "mfa_required_for_all_apps":true,
    "mfa_config":{"allowed_authenticators":["totp"],"session_duration":"24h"},
    "mfa_piv_key_requirements":{"require_fips_device":true,"pin_policy":"always"}
  }},
  "cloudflare_zero_trust_gateway_settings.example": {"settings":{
    "host_selector":{"enabled":true}, "sandbox":{"enabled":true,"fallback_action":"block"},
    "inspection":{"mode":"dynamic"}, "max_ttl_secs":900,
    "antivirus":{"enabled_download_phase":true},
    "block_page":{"name":"do not copy","mode":"customized_block_page",
      "include_context":true,"suppress_footer":false,"target_uri":"https://example.com/blocked",
      "read_only":true,"version":3}
  }},
  "cloudflare_zero_trust_device_settings.example_proxy": {
    "external_emergency_signal_enabled":true,
    "external_emergency_signal_url":"https://192.0.2.1/${literal}",
    "external_emergency_signal_fingerprint":"abc",
    "external_emergency_signal_interval":"7m"
  }
}`), &snapshot))
	updated, err := Restore(files, snapshot)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	file, diags := hclsyntax.ParseConfig(updated["main.tf"], "main.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	blocks := file.Body.(*hclsyntax.Body).Blocks
	org := blocks[0].Body
	value, d := org.Attributes["deny_unmatched_requests"].Expr.Value(nil)
	require.False(t, d.HasErrors())
	require.False(t, value.True(), "explicit disable must win")
	value, d = org.Attributes["deny_unmatched_requests_exempted_zone_names"].Expr.Value(nil)
	require.False(t, d.HasErrors())
	require.Zero(t, value.LengthInt(), "explicit empty list must win")
	require.NotNil(t, org.Attributes["mfa_config"])
	require.NotNil(t, org.Attributes["mfa_piv_key_requirements"])
	require.NotNil(t, org.Attributes["mfa_required_for_all_apps"])
	require.Nil(t, org.Attributes["name"])
	value, d = blocks[1].Body.Attributes["settings"].Expr.Value(nil)
	require.False(t, d.HasErrors())
	require.True(t, value.GetAttr("sandbox").IsNull(), "explicit null must win")
	require.True(t, value.GetAttr("host_selector").GetAttr("enabled").True())
	require.False(t, value.GetAttr("tls_decrypt").GetAttr("enabled").True())
	require.Equal(t, "dynamic", value.GetAttr("inspection").GetAttr("mode").AsString())
	// block_page is absent from the migrated config, so it is created holding
	// only the v5-only fields; v4-expressible and computed fields stay out.
	blockPage := value.GetAttr("block_page")
	require.Equal(t, "customized_block_page", blockPage.GetAttr("mode").AsString())
	require.True(t, blockPage.GetAttr("include_context").True())
	require.False(t, blockPage.GetAttr("suppress_footer").True(), "explicit false must be restored")
	require.Equal(t, "https://example.com/blocked", blockPage.GetAttr("target_uri").AsString())
	require.False(t, blockPage.Type().HasAttribute("name"), "v4-expressible field must not be copied")
	require.False(t, blockPage.Type().HasAttribute("read_only"), "computed field must not be copied")
	require.False(t, blockPage.Type().HasAttribute("version"), "computed field must not be copied")
	// Fields outside the allowlist are never copied, even when present live.
	require.False(t, value.Type().HasAttribute("antivirus"))
	value, d = blocks[2].Body.Attributes["external_emergency_signal_enabled"].Expr.Value(nil)
	require.False(t, d.HasErrors())
	require.False(t, value.True())
	value, d = blocks[2].Body.Attributes["external_emergency_signal_url"].Expr.Value(nil)
	require.False(t, d.HasErrors(), d.Error())
	require.Equal(t, "https://192.0.2.1/${literal}", value.AsString())
	// Once restored, these are ordinary explicit HCL values. A second run must
	// not keep adding fields or restore a user's subsequently edited value.
	again, err := Restore(updated, snapshot)
	require.NoError(t, err)
	require.Empty(t, again)
}

// The BUGS-2159 case: v4 had a block_page block, so the migrated config already
// carries the eight v4 fields and the v5-only ones must merge into that object
// rather than replace it.
func TestRestoreMergesIntoExistingBlockPage(t *testing.T) {
	files := map[string][]byte{"main.tf": []byte(`resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = var.account_id
  settings = {
    block_page = {
      # keep this comment
      name    = "Cloudflare"
      enabled = var.block_page_enabled
      mode    = "redirect_uri"
    }
  }
}`)}
	snapshot := Snapshot{"cloudflare_zero_trust_gateway_settings.test": json.RawMessage(`{"settings":{
    "block_page":{"name":"live name","mode":"customized_block_page",
      "include_context":true,"suppress_footer":true,"target_uri":"https://example.com/blocked"}
  }}`)}

	updated, err := Restore(files, snapshot)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	text := string(updated["main.tf"])

	file, diags := hclsyntax.ParseConfig(updated["main.tf"], "main.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	settings := file.Body.(*hclsyntax.Body).Blocks[0].Body.Attributes["settings"]
	object, ok := settings.Expr.(*hclsyntax.ObjectConsExpr)
	require.True(t, ok)
	blockPage, ok := object.Items[0].ValueExpr.(*hclsyntax.ObjectConsExpr)
	require.True(t, ok, "block_page must remain an object")

	keys := map[string]bool{}
	for _, item := range blockPage.Items {
		value, d := item.KeyExpr.Value(nil)
		require.False(t, d.HasErrors())
		keys[value.AsString()] = true
	}
	for _, name := range []string{"name", "enabled", "mode", "include_context", "suppress_footer", "target_uri"} {
		require.True(t, keys[name], "block_page must contain %s", name)
	}

	// Existing configuration wins and is untouched: the comment, the expression,
	// and the explicitly configured mode all survive.
	require.Contains(t, text, "# keep this comment")
	require.Contains(t, text, "var.block_page_enabled")
	require.Contains(t, text, `"redirect_uri"`)
	require.NotContains(t, text, "customized_block_page")
	require.NotContains(t, text, "live name")

	// Restored values are now ordinary explicit HCL, so a second pass is a no-op.
	again, err := Restore(updated, snapshot)
	require.NoError(t, err)
	require.Empty(t, again)
}

func TestRestoreRejectsNonObjectBlockPage(t *testing.T) {
	for _, blockPage := range []string{"var.block_page", `"string"`} {
		t.Run(blockPage, func(t *testing.T) {
			config := `resource "cloudflare_zero_trust_gateway_settings" "test" { settings = { block_page = ` + blockPage + ` } }`
			files := map[string][]byte{"main.tf": []byte(config)}
			output, err := Restore(files, Snapshot{
				"cloudflare_zero_trust_gateway_settings.test": json.RawMessage(`{"settings":{"block_page":{"mode":"customized_block_page"}}}`),
			})
			require.Error(t, err)
			require.Nil(t, output)
			require.Equal(t, config, string(files["main.tf"]))
		})
	}
}

func TestRestoreFalseEmptyAndAbsentValues(t *testing.T) {
	input := map[string][]byte{"main.tf": []byte(`resource "cloudflare_zero_trust_organization" "test" {}`)}
	output, err := Restore(input, Snapshot{
		"cloudflare_zero_trust_organization.test": json.RawMessage(`{
"deny_unmatched_requests":false,"deny_unmatched_requests_exempted_zone_names":[],
"mfa_config":null,"mfa_required_for_all_apps":false}`),
	})
	require.NoError(t, err)
	text := string(output["main.tf"])
	require.Contains(t, text, "false")
	require.Contains(t, text, "[]")
	require.NotContains(t, text, "mfa_config")
	require.NotContains(t, text, "mfa_piv_key_requirements")
}

func TestRestoreRejectsAmbiguousSnapshots(t *testing.T) {
	for _, tc := range []struct{ name, config, response, address string }{
		{"missing resource", ``, `{}`, "cloudflare_zero_trust_organization.test"},
		{"count", `resource "cloudflare_zero_trust_organization" "test" { count = 2 }`, `{}`, "cloudflare_zero_trust_organization.test"},
		{"for_each", `resource "cloudflare_zero_trust_organization" "test" { for_each = var.accounts }`, `{}`, "cloudflare_zero_trust_organization.test"},
		{"API failure", `resource "cloudflare_zero_trust_organization" "test" {}`, `{"success":false,"result":{}}`, "cloudflare_zero_trust_organization.test"},
		{"null result", `resource "cloudflare_zero_trust_organization" "test" {}`, `{"success":true,"result":null}`, "cloudflare_zero_trust_organization.test"},
		{"settings expression", `resource "cloudflare_zero_trust_gateway_settings" "test" { settings = var.settings }`, `{"settings":{"host_selector":{"enabled":true}}}`, "cloudflare_zero_trust_gateway_settings.test"},
		{"missing settings", `resource "cloudflare_zero_trust_gateway_settings" "test" {}`, `{}`, "cloudflare_zero_trust_gateway_settings.test"},
		{"unsupported resource", `resource "cloudflare_zone" "test" {}`, `{}`, "cloudflare_zone.test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string][]byte{"main.tf": []byte(tc.config)}
			output, err := Restore(files, Snapshot{tc.address: json.RawMessage(tc.response)})
			require.Error(t, err)
			require.Nil(t, output)
			require.Equal(t, tc.config, string(files["main.tf"]))
		})
	}
}

func TestRestoreGatewayObjectFormattingAndExpressions(t *testing.T) {
	for _, settings := range []string{
		`{}`,
		`{ tls_decrypt = { enabled = var.enabled } }`,
		"{\n# keep this comment\ntls_decrypt = { enabled = var.enabled }\n}",
	} {
		t.Run(settings, func(t *testing.T) {
			files := map[string][]byte{"main.tf": []byte(`resource "cloudflare_zero_trust_gateway_settings" "test" { settings = ` + settings + ` }`)}
			output, err := Restore(files, Snapshot{"cloudflare_zero_trust_gateway_settings.test": json.RawMessage(`{"settings":{"host_selector":{"enabled":false},"sandbox":{"enabled":true,"fallback_action":"block"}}}`)})
			require.NoError(t, err)
			_, d := hclsyntax.ParseConfig(output["main.tf"], "main.tf", hcl.InitialPos)
			require.False(t, d.HasErrors(), d.Error())
			if strings.Contains(settings, "var.enabled") {
				require.Contains(t, string(output["main.tf"]), "var.enabled")
			}
			if strings.Contains(settings, "# keep") {
				require.Contains(t, string(output["main.tf"]), "# keep this comment")
			}
		})
	}
}
