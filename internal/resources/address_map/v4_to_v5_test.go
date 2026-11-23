package address_map

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func TestMixedCollections(t *testing.T) {
	file := parse(t, `resource "cloudflare_address_map" "example" {
  account_id = var.account_id
  enabled = false
  for_each = var.accounts
  lifecycle { ignore_changes = [memberships] }

  ips { ip = "192.0.2.1" }
  dynamic "ips" {
    for_each = var.addresses
    iterator = entry
    content {
      ip = entry.key == "use-fallback" ? var.entry.value : entry.value
    }
  }
  ips { ip = "2001:db8::1" }

  dynamic "memberships" {
    for_each = var.zones
    content {
      identifier = memberships.value
      kind = "zone"
    }
  }
}`)
	block := file.Body().Blocks()[0]
	m := &V4ToV5Migrator{}
	result, err := m.TransformConfig(nil, block)
	require.NoError(t, err)
	require.False(t, result.RemoveOriginal)
	require.Equal(t, []*hclwrite.Block{block}, result.Blocks)
	output := file.Bytes()
	syntax, diags := hclsyntax.ParseConfig(output, "main.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "%s", diags)
	body := syntax.Body.(*hclsyntax.Body).Blocks[0].Body
	require.Contains(t, body.Attributes, "for_each")
	require.Len(t, body.Blocks, 1)
	require.Equal(t, "lifecycle", body.Blocks[0].Type)

	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{"var": cty.ObjectVal(map[string]cty.Value{
			"addresses": cty.MapVal(map[string]cty.Value{
				"duplicate":    cty.StringVal("192.0.2.1"),
				"use-fallback": cty.StringVal("192.0.2.99"),
			}),
			"entry": cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal("192.0.2.2")}),
			"zones": cty.SetVal([]cty.Value{cty.StringVal("test-zone")}),
		})},
		Functions: map[string]function.Function{"concat": stdlib.ConcatFunc, "distinct": stdlib.DistinctFunc},
	}
	ips, diags := body.Attributes["ips"].Expr.Value(ctx)
	require.False(t, diags.HasErrors(), "%s", diags)
	require.Equal(t, []cty.Value{cty.StringVal("192.0.2.1"), cty.StringVal("192.0.2.2"), cty.StringVal("2001:db8::1")}, ips.AsValueSlice())
	memberships, diags := body.Attributes["memberships"].Expr.Value(ctx)
	require.False(t, diags.HasErrors(), "%s", diags)
	require.Equal(t, []cty.Value{cty.ObjectVal(map[string]cty.Value{
		"identifier": cty.StringVal("test-zone"), "kind": cty.StringVal("zone"),
	})}, memberships.AsValueSlice())

	file = parse(t, string(output))
	_, err = m.TransformConfig(nil, file.Body().Blocks()[0])
	require.NoError(t, err)
	require.Equal(t, string(output), string(file.Bytes()))
}

func TestOmittedCollections(t *testing.T) {
	file := parse(t, `resource "cloudflare_address_map" "example" {
  account_id = var.account_id
  enabled = false
}`)
	_, err := (&V4ToV5Migrator{}).TransformConfig(nil, file.Body().Blocks()[0])
	require.NoError(t, err)
	require.Contains(t, string(file.Bytes()), "ips         = []")
	require.Contains(t, string(file.Bytes()), "memberships = []")
}

func TestMalformedMembershipDoesNotDropIPs(t *testing.T) {
	file := parse(t, `resource "cloudflare_address_map" "example" {
  ips { ip = "192.0.2.1" }
  memberships { kind = "zone" }
}`)
	original := string(file.Bytes())
	_, err := (&V4ToV5Migrator{}).TransformConfig(nil, file.Body().Blocks()[0])
	require.ErrorContains(t, err, "memberships: missing identifier")
	require.Equal(t, original, string(file.Bytes()))
}

func parse(t *testing.T, config string) *hclwrite.File {
	t.Helper()
	file, diags := hclwrite.ParseConfig([]byte(config), "main.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "%s", diags)
	return file
}
