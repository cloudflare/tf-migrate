package api_token

import (
	"fmt"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func migratePermissionGroups(t *testing.T, input string) (hclsyntax.Expression, *transform.Context) {
	t.Helper()
	file, diags := hclwrite.ParseConfig([]byte("permission_groups = "+input+"\n"), "permissions.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "%s", diags.Error())
	ctx := &transform.Context{Filename: "permissions.tf"}
	(&V4ToV5Migrator{}).transformPermissionGroups(ctx, file.Body())
	_, diags = hclwrite.ParseConfig(file.Bytes(), "output.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Generated invalid HCL: %s\n%s", diags.Error(), file.Bytes())
	expr, diags := hclsyntax.ParseExpression(file.Body().GetAttribute("permission_groups").Expr().BuildTokens(nil).Bytes(), "output.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "%s", diags.Error())
	return expr, ctx
}

// Evaluate generated expressions against overlapping permission names. A
// name-only lookup or wrong scope must fail these tests, even if it parses.
func TestPermissionGroupsEvaluation(t *testing.T) {
	var groups []cty.Value
	for _, scope := range []string{"zone", "account", "user", "r2"} {
		groups = append(groups, cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal(scope + "-id"), "name": cty.StringVal("Shared Name"),
			"scopes": cty.ListVal([]cty.Value{cty.StringVal(permissionGroupScopes[scope])}),
		}))
	}
	groups = append(groups, cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("unique-id"), "name": cty.StringVal("Unique $ % \" Name"),
		"scopes": cty.ListVal([]cty.Value{cty.StringVal(permissionGroupScopes["zone"])}),
	}))
	eval := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"data": cty.ObjectVal(map[string]cty.Value{
				"cloudflare_api_token_permission_groups_list": cty.ObjectVal(map[string]cty.Value{
					"all": cty.ObjectVal(map[string]cty.Value{"result": cty.ListVal(groups)}),
				}),
			}),
			"local": cty.ObjectVal(map[string]cty.Value{
				"names": cty.ListVal([]cty.Value{cty.StringVal("Shared Name"), cty.StringVal("skip")}),
				"ids":   cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
				"id":    cty.StringVal("local-id"),
			}),
		},
		Functions: map[string]function.Function{
			"contains": stdlib.ContainsFunc,
			"one": function.New(&function.Spec{
				Params: []function.Parameter{{Name: "values", Type: cty.DynamicPseudoType}},
				Type:   function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
					if args[0].LengthInt() == 0 {
						return cty.NullVal(cty.String), nil
					}
					if args[0].LengthInt() != 1 {
						return cty.NilVal, fmt.Errorf("ambiguous permission name")
					}
					return args[0].Index(cty.NumberIntVal(0)), nil
				},
			}),
		},
	}
	tests := []struct {
		name, input string
		ids         []string
		warning     bool
	}{
		{"zone", `[data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, []string{"zone-id"}, false},
		{"account", `[data.cloudflare_api_token_permission_groups.all.account["Shared Name"]]`, []string{"account-id"}, false},
		{"user", `[data.cloudflare_api_token_permission_groups.all.user["Shared Name"]]`, []string{"user-id"}, false},
		{"r2", `[data.cloudflare_api_token_permission_groups.all.r2["Shared Name"]]`, []string{"r2-id"}, false},
		{"deprecated map and escaped name", `[data.cloudflare_api_token_permission_groups.all.permissions["Unique $ % \" Name"]]`, []string{"unique-id"}, false},
		{"literal first", `["literal-id", data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, []string{"literal-id", "zone-id"}, false},
		{"reference first", `[data.cloudflare_api_token_permission_groups.all.zone["Shared Name"], "literal-id"]`, []string{"zone-id", "literal-id"}, false},
		{"generic expression mixed with lookup", `[local.id, data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, []string{"local-id", "zone-id"}, true},
		{"comprehension and filter", `[for p in local.names : data.cloudflare_api_token_permission_groups.all.zone[p] if p != "skip"]`, []string{"zone-id"}, false},
		{"key and value iterators", `[for i, pg in local.names : data.cloudflare_api_token_permission_groups.all.zone[pg] if i == 0]`, []string{"zone-id"}, false},
		{"generic comprehension", `[for p in local.ids : p]`, []string{"a", "b"}, true},
		{"collection reference", `local.ids`, []string{"a", "b"}, true},
		{"literal sorting", `["z", "a"]`, []string{"a", "z"}, false},
		{"existing objects", `[{ id = "z" }, { id = "a" }]`, []string{"a", "z"}, false},
		{"empty", `[]`, []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, ctx := migratePermissionGroups(t, tt.input)
			assert.Equal(t, tt.warning, len(ctx.Diagnostics) > 0, "%s", ctx.Diagnostics.Error())
			value, diags := expr.Value(eval)
			require.False(t, diags.HasErrors(), "%s", diags.Error())
			ids := []string{}
			for it := value.ElementIterator(); it.Next(); {
				_, object := it.Element()
				ids = append(ids, object.GetAttr("id").AsString())
			}
			assert.Equal(t, tt.ids, ids)
		})
	}
	expr, _ := migratePermissionGroups(t, `[data.cloudflare_api_token_permission_groups.all.permissions["Shared Name"]]`)
	_, diags := expr.Value(eval)
	assert.True(t, diags.HasErrors(), "Ambiguous names must not select the first ID")
}

func TestPermissionGroupsUnsupportedExpressions(t *testing.T) {
	for _, input := range []string{
		`[data.cloudflare_api_token_permission_groups.all.unknown["Name"]]`,
		`[upper(data.cloudflare_api_token_permission_groups.all.zone["Name"])]`,
		`[data.cloudflare_api_token_permission_groups.all.zone["Name"] == "x" ? "a" : "b"]`,
		`{for p in local.names : p => p}`,
	} {
		t.Run(input, func(t *testing.T) {
			_, ctx := migratePermissionGroups(t, input)
			require.Len(t, ctx.Diagnostics, 1)
			assert.Equal(t, hcl.DiagWarning, ctx.Diagnostics[0].Severity)
			assert.Contains(t, ctx.Diagnostics[0].Detail, "permissions.tf")
		})
	}
}
