package zero_trust_access_identity_provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

func migrate(t *testing.T, input string) string {
	t.Helper()
	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())
	ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
	out := hclwrite.NewEmptyFile()
	m := NewV4ToV5Migrator()
	for _, b := range file.Body().Blocks() {
		res, err := m.TransformConfig(ctx, b)
		require.NoError(t, err)
		for _, nb := range res.Blocks {
			out.Body().AppendBlock(nb)
		}
	}
	assert.Empty(t, ctx.Diagnostics)
	return strings.TrimSpace(string(hclwrite.Format(out.Bytes())))
}

func TestV4ToV5(t *testing.T) {
	for _, tt := range []struct{ name, input, expected string }{
		{
			name: "lookup by name",
			input: `data "cloudflare_zero_trust_access_identity_provider" "idp" {
  account_id = var.account_id
  name       = "MyIdentity"
}`,
			expected: `data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
  account_id = var.account_id
}
data "cloudflare_zero_trust_access_identity_provider" "idp" {
  account_id           = var.account_id
  identity_provider_id = [for p in data.cloudflare_zero_trust_access_identity_providers.idp_lookup.result : p.id if p.name == "MyIdentity"][0]
}`,
		},
		{
			name: "deprecated v4 name, zone scope",
			input: `data "cloudflare_access_identity_provider" "idp" {
  zone_id = var.zone_id
  name    = var.idp_name
}`,
			expected: `data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
  zone_id = var.zone_id
}
data "cloudflare_zero_trust_access_identity_provider" "idp" {
  zone_id              = var.zone_id
  identity_provider_id = [for p in data.cloudflare_zero_trust_access_identity_providers.idp_lookup.result : p.id if p.name == var.idp_name][0]
}`,
		},
		{
			name: "for_each over accounts repeats the lookup",
			input: `data "cloudflare_zero_trust_access_identity_provider" "idp" {
  for_each   = var.accounts
  account_id = each.value
  name       = "Okta"
}`,
			expected: `data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
  for_each   = var.accounts
  account_id = each.value
}
data "cloudflare_zero_trust_access_identity_provider" "idp" {
  for_each             = var.accounts
  account_id           = each.value
  identity_provider_id = [for p in data.cloudflare_zero_trust_access_identity_providers.idp_lookup[each.key].result : p.id if p.name == "Okta"][0]
}`,
		},
		{
			name: "for_each over names shares one lookup",
			input: `data "cloudflare_zero_trust_access_identity_provider" "idp" {
  for_each   = toset(["a", "b"])
  account_id = var.account_id
  name       = each.key
}`,
			expected: `data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
  account_id = var.account_id
}
data "cloudflare_zero_trust_access_identity_provider" "idp" {
  for_each             = toset(["a", "b"])
  account_id           = var.account_id
  identity_provider_id = [for p in data.cloudflare_zero_trust_access_identity_providers.idp_lookup.result : p.id if p.name == each.key][0]
}`,
		},
		{
			name: "lookup name already taken",
			input: `data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
  account_id = var.account_id
}
data "cloudflare_zero_trust_access_identity_provider" "idp" {
  account_id = var.account_id
  name       = "MyIdentity"
}`,
			expected: `data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
  account_id = var.account_id
}
data "cloudflare_zero_trust_access_identity_providers" "idp_lookup_2" {
  account_id = var.account_id
}
data "cloudflare_zero_trust_access_identity_provider" "idp" {
  account_id           = var.account_id
  identity_provider_id = [for p in data.cloudflare_zero_trust_access_identity_providers.idp_lookup_2.result : p.id if p.name == "MyIdentity"][0]
}`,
		},
		{
			name: "already v5 is unchanged",
			input: `data "cloudflare_zero_trust_access_identity_provider" "idp" {
  account_id           = var.account_id
  identity_provider_id = "abc"
}`,
			expected: `data "cloudflare_zero_trust_access_identity_provider" "idp" {
  account_id           = var.account_id
  identity_provider_id = "abc"
}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			norm := func(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), "\n\n", "\n") }
			assert.Equal(t, norm(string(hclwrite.Format([]byte(tt.expected)))), norm(migrate(t, tt.input)))
		})
	}
}
