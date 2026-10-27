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
    "inspection":{"mode":"dynamic"}
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
	require.False(t, value.Type().HasAttribute("inspection"))
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
