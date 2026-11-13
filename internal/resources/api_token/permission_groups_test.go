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
	return migratePolicy(t, input, "")
}

// migratePolicy runs transformPermissionGroups on a policy body that may also
// carry a v4 resources map, which drives scope inference.
func migratePolicy(t *testing.T, input, resources string) (hclsyntax.Expression, *transform.Context) {
	t.Helper()
	src := "permission_groups = " + input + "\n"
	if resources != "" {
		src += "resources = " + resources + "\n"
	}
	file, diags := hclwrite.ParseConfig([]byte(src), "permissions.tf", hcl.InitialPos)
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
func permissionGroupsEvalContext() *hcl.EvalContext {
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
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"data": cty.ObjectVal(map[string]cty.Value{
				"cloudflare_api_token_permission_groups_list": cty.ObjectVal(map[string]cty.Value{
					"all": cty.ObjectVal(map[string]cty.Value{"result": cty.ListVal(groups)}),
					"by_key": cty.ObjectVal(map[string]cty.Value{
						"zone": cty.ObjectVal(map[string]cty.Value{"result": cty.ListVal(groups)}),
					}),
					"by_count": cty.TupleVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"result": cty.ListVal(groups)})}),
				}),
			}),
			"each":  cty.ObjectVal(map[string]cty.Value{"key": cty.StringVal("zone")}),
			"count": cty.ObjectVal(map[string]cty.Value{"index": cty.NumberIntVal(0)}),
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
}

func TestPermissionGroupsEvaluation(t *testing.T) {
	eval := permissionGroupsEvalContext()
	tests := []struct {
		name, input string
		ids         []string
		warning     bool
	}{
		{"zone", `[data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, []string{"zone-id"}, false},
		{"account", `[data.cloudflare_api_token_permission_groups.all.account["Shared Name"]]`, []string{"account-id"}, false},
		{"user", `[data.cloudflare_api_token_permission_groups.all.user["Shared Name"]]`, []string{"user-id"}, false},
		{"r2", `[data.cloudflare_api_token_permission_groups.all.r2["Shared Name"]]`, []string{"r2-id"}, false},
		{"deprecated map and escaped name", `[data.cloudflare_api_token_permission_groups.all.permissions["Unique $ % \" Name"]]`, []string{"unique-id"}, true},
		{"literal first", `["literal-id", data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, []string{"literal-id", "zone-id"}, false},
		{"reference first", `[data.cloudflare_api_token_permission_groups.all.zone["Shared Name"], "literal-id"]`, []string{"zone-id", "literal-id"}, false},
		{"generic expression mixed with lookup", `[local.id, data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, []string{"local-id", "zone-id"}, true},
		{"comprehension and filter", `[for p in local.names : data.cloudflare_api_token_permission_groups.all.zone[p] if p != "skip"]`, []string{"zone-id"}, false},
		{"key and value iterators", `[for i, pg in local.names : data.cloudflare_api_token_permission_groups.all.zone[pg] if i == 0]`, []string{"zone-id"}, false},
		{"for_each instance", `[data.cloudflare_api_token_permission_groups.by_key[each.key].zone["Shared Name"]]`, []string{"zone-id"}, false},
		{"constant string instance", `[data.cloudflare_api_token_permission_groups.by_key["zone"].account["Shared Name"]]`, []string{"account-id"}, false},
		{"count instance", `[data.cloudflare_api_token_permission_groups.by_count[count.index].user["Shared Name"]]`, []string{"user-id"}, false},
		{"constant numeric instance", `[data.cloudflare_api_token_permission_groups.by_count[0].r2["Shared Name"]]`, []string{"r2-id"}, false},
		{"dynamic instance and key", `[data.cloudflare_api_token_permission_groups.by_key[each.key].zone[local.names[0]]]`, []string{"zone-id"}, false},
		{"indexed comprehension and iterator collision", `[for i, pg in local.names : data.cloudflare_api_token_permission_groups.by_key[each.key].zone[pg] if i == 0]`, []string{"zone-id"}, false},
		{"indexed deprecated map", `[data.cloudflare_api_token_permission_groups.by_key[each.key].permissions["Unique $ % \" Name"]]`, []string{"unique-id"}, true},
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

// A name-only .permissions[...] lookup takes its scope from the policy's
// resources when every key requires the same scope. Otherwise it stays
// name-only and must warn, because names like "Logs Write" exist at both
// account and zone scope.
func TestPermissionGroupsScopeFromResources(t *testing.T) {
	eval := permissionGroupsEvalContext()
	shared := `[data.cloudflare_api_token_permission_groups.all.permissions["Shared Name"]]`
	tests := []struct {
		name, input, resources string
		ids                    []string // nil: evaluation must fail (ambiguous)
		warning                bool
	}{
		{"account wildcard", shared, `{ "com.cloudflare.api.account.*" = "*" }`, []string{"account-id"}, false},
		{"specific account", shared, `{ "com.cloudflare.api.account.abc123" = "*" }`, []string{"account-id"}, false},
		{"zone wildcard", shared, `{ "com.cloudflare.api.account.zone.*" = "*" }`, []string{"zone-id"}, false},
		{"several zones", shared, `{ "com.cloudflare.api.account.zone.a" = "*", "com.cloudflare.api.account.zone.b" = "*" }`, []string{"zone-id"}, false},
		{"user", shared, `{ "com.cloudflare.api.user.u1" = "*" }`, []string{"user-id"}, false},
		{"r2 bucket", shared, `{ "com.cloudflare.edge.r2.bucket.abc_default_b" = "*" }`, []string{"r2-id"}, false},
		{"comprehension takes scope", `[for p in local.names : data.cloudflare_api_token_permission_groups.all.permissions[p] if p != "skip"]`, `{ "com.cloudflare.api.account.zone.*" = "*" }`, []string{"zone-id"}, false},
		{"explicit category wins", `[data.cloudflare_api_token_permission_groups.all.zone["Shared Name"]]`, `{ "com.cloudflare.api.account.*" = "*" }`, []string{"zone-id"}, false},
		{"mixed account and zone", shared, `{ "com.cloudflare.api.account.*" = "*", "com.cloudflare.api.account.zone.*" = "*" }`, nil, true},
		{"resources from variable", shared, `var.resources`, nil, true},
		{"interpolated key", shared, `{ "com.cloudflare.api.account.${var.account_id}" = "*" }`, nil, true},
		{"unknown resource prefix", shared, `{ "com.example.other" = "*" }`, nil, true},
		{"no resources", shared, ``, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, ctx := migratePolicy(t, tt.input, tt.resources)
			if tt.warning {
				require.Len(t, ctx.Diagnostics, 1, "%s", ctx.Diagnostics.Error())
				assert.Equal(t, hcl.DiagWarning, ctx.Diagnostics[0].Severity)
				assert.Contains(t, ctx.Diagnostics[0].Summary, "by name only")
				assert.Contains(t, ctx.Diagnostics[0].Detail, "permissions.tf")
			} else {
				assert.Empty(t, ctx.Diagnostics, "%s", ctx.Diagnostics.Error())
			}
			value, diags := expr.Value(eval)
			if tt.ids == nil {
				assert.True(t, diags.HasErrors(), "ambiguous name must not select an ID")
				return
			}
			require.False(t, diags.HasErrors(), "%s", diags.Error())
			ids := []string{}
			for it := value.ElementIterator(); it.Next(); {
				_, object := it.Element()
				ids = append(ids, object.GetAttr("id").AsString())
			}
			assert.Equal(t, tt.ids, ids)
		})
	}
}
