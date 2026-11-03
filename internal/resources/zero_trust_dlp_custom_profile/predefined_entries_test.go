package zero_trust_dlp_custom_profile

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// transformPredefined migrates one v4 predefined profile and returns the
// migrated resource block and the warnings raised.
func transformPredefined(t *testing.T, entries, lifecycle string) (*hclwrite.Block, []string) {
	t.Helper()
	src := `resource "cloudflare_dlp_profile" "p" {
  id                  = "p1"
  account_id          = "a1"
  name                = "P"
  type                = "predefined"
  allowed_match_count = 0
` + entries + lifecycle + "\n}\n"
	file, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	ctx := &transform.Context{Content: []byte(src), Filename: "test.tf", CFGFile: file}
	result, err := (&V4ToV5Migrator{}).TransformConfig(ctx, file.Body().Blocks()[0])
	require.NoError(t, err)
	var warnings []string
	for _, d := range ctx.Diagnostics {
		warnings = append(warnings, d.Summary+": "+d.Detail)
	}
	return result.Blocks[0], warnings
}

const literalEntry = `
  entry {
    id      = "on"
    name    = "On"
    enabled = true
  }
`

func TestPredefinedEntryWarnings(t *testing.T) {
	_, warnings := transformPredefined(t, literalEntry, "")
	require.Empty(t, warnings)

	_, warnings = transformPredefined(t, literalEntry, "lifecycle {\n ignore_changes = [entry]\n}")
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "enabled_entries not migrated")
	require.Contains(t, warnings[0], "ignored changes to entry,")

	_, warnings = transformPredefined(t, literalEntry, "lifecycle {\n ignore_changes = all\n}")
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "ignored changes to all attributes")

	// ignore_changes = all with no entry block has nothing to lose.
	_, warnings = transformPredefined(t, "", "lifecycle {\n ignore_changes = all\n}")
	require.Empty(t, warnings)

	_, warnings = transformPredefined(t, literalEntry, "lifecycle {\n ignore_changes = [entry[0].enabled]\n}")
	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "ignore_changes widened")

	block, warnings := transformPredefined(t, literalEntry+`
  dynamic "entry" {
    for_each = var.entries
  }
`, "")
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Dynamic entry block requires manual migration")
	require.Contains(t, warnings[0], "left in place")
	require.Nil(t, block.Body().GetAttribute("enabled_entries"))
	require.Len(t, block.Body().Blocks(), 2, "entry and dynamic entry blocks must be kept")
}

// Evaluates the generated enabled_entries the way Terraform would, to check the
// output selects the same entries the v4 blocks enabled.
func TestPredefinedEnabledEntriesEvaluate(t *testing.T) {
	block, warnings := transformPredefined(t, literalEntry+`
  entry {
    id      = local.ids[0]
    name    = "Conditional"
    enabled = var.on || var.force
  }

  entry {
    id      = "off"
    name    = "Off"
    enabled = false
  }

  dynamic "entry" {
    for_each = var.by_name
    content {
      id      = entry.value.id
      name    = entry.key
      enabled = entry.key != "skip" && entry.value.enabled
    }
  }

  dynamic "entry" {
    for_each = toset(var.extra)
    iterator = e
    content {
      id      = e.key
      name    = e.value
      enabled = true
    }
  }
`, "")
	require.Empty(t, warnings)
	source := hclwrite.Format(block.BuildTokens(nil).Bytes())
	file, diags := hclsyntax.ParseConfig(source, "out.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	attr := file.Body.(*hclsyntax.Body).Blocks[0].Body.Attributes["enabled_entries"]
	require.NotNil(t, attr, string(source))

	entry := func(id string, enabled bool) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal(id), "enabled": cty.BoolVal(enabled)})
	}
	eval := func(on bool) []string {
		ctx := &hcl.EvalContext{
			Variables: map[string]cty.Value{
				"var": cty.ObjectVal(map[string]cty.Value{
					"on":    cty.BoolVal(on),
					"force": cty.False,
					"by_name": cty.MapVal(map[string]cty.Value{
						"a":    entry("map-a", true),
						"b":    entry("map-b", false),
						"skip": entry("map-skip", true),
					}),
					"extra": cty.ListVal([]cty.Value{cty.StringVal("set-x")}),
				}),
				"local": cty.ObjectVal(map[string]cty.Value{
					"ids": cty.TupleVal([]cty.Value{cty.StringVal("cond")}),
				}),
			},
			Functions: map[string]function.Function{
				"compact": stdlib.CompactFunc,
				"concat":  stdlib.ConcatFunc,
				"toset":   stdlib.MakeToFunc(cty.Set(cty.String)),
			},
		}
		value, diags := attr.Expr.Value(ctx)
		require.False(t, diags.HasErrors(), "%s\n%s", diags.Error(), source)
		var ids []string
		for it := value.ElementIterator(); it.Next(); {
			_, v := it.Element()
			ids = append(ids, v.AsString())
		}
		return ids
	}
	require.Equal(t, "on,cond,map-a,set-x", strings.Join(eval(true), ","))
	require.Equal(t, "on,map-a,set-x", strings.Join(eval(false), ","))
}

func TestAlreadyMigratedPredefinedProfileIsUnchanged(t *testing.T) {
	src := `resource "cloudflare_zero_trust_dlp_predefined_profile" "v5" {
  account_id      = "a1"
  profile_id      = "p1"
  enabled_entries = ["x"]
}
`
	file, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	ctx := &transform.Context{Content: []byte(src), Filename: "test.tf", CFGFile: file}
	result, err := (&V4ToV5Migrator{}).TransformConfig(ctx, file.Body().Blocks()[0])
	require.NoError(t, err)
	require.Len(t, result.Blocks, 1, "no moved block")
	require.Equal(t, src, string(hclwrite.Format(result.Blocks[0].BuildTokens(nil).Bytes())))
	require.Empty(t, ctx.Diagnostics)
}
