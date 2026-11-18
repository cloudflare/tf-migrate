package hcl

import (
	"strconv"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// policiesTransformer mirrors the zero_trust_access_application caller.
func policiesTransformer(element hclwrite.Tokens, index int) map[string]hclwrite.Tokens {
	return map[string]hclwrite.Tokens{
		"id": element,
		"precedence": {&hclwrite.Token{
			Type:  hclsyntax.TokenNumberLit,
			Bytes: []byte(strconv.Itoa(index + 1)),
		}},
	}
}

func convertPolicies(t *testing.T, policies string) (bool, string) {
	t.Helper()
	input := "resource \"test\" \"example\" {\n  policies = " + policies + "\n}\n"
	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)

	ok := ConvertArrayAttributeToObjectArray(file.Body().Blocks()[0].Body(), "policies", policiesTransformer)
	out := string(hclwrite.Format(file.Bytes()))

	// Whatever happens, the result must still be valid HCL.
	_, parseDiags := hclsyntax.ParseConfig([]byte(out), "out.tf", hcl.InitialPos)
	require.False(t, parseDiags.HasErrors(), "output is not valid HCL: %v\n%s", parseDiags, out)
	return ok, out
}

// Elements that contain their own brackets, parentheses, braces or commas must
// stay whole. Before the fix, the first ] or , anywhere ended the element.
func TestConvertArrayAttributeToObjectArray_NestedElements(t *testing.T) {
	tests := []struct {
		name     string
		policies string
		wantIDs  []string
	}{
		{
			name:     "index with each.key",
			policies: `[cloudflare_zero_trust_access_policy.allow[each.key].id, cloudflare_zero_trust_access_policy.deny.id]`,
			wantIDs: []string{
				`cloudflare_zero_trust_access_policy.allow[each.key].id`,
				`cloudflare_zero_trust_access_policy.deny.id`,
			},
		},
		{
			name:     "function call with a comma",
			policies: `[lookup(var.policy_ids, "allow"), "policy-2"]`,
			wantIDs:  []string{`lookup(var.policy_ids, "allow")`, `"policy-2"`},
		},
		{
			name:     "index inside a template",
			policies: `["${var.ids[0]}", var.ids[1]]`,
			wantIDs:  []string{`"${var.ids[0]}"`, `var.ids[1]`},
		},
		{
			name:     "conditional with nested list access",
			policies: `[var.strict ? local.ids["strict"] : local.ids["open"]]`,
			wantIDs:  []string{`var.strict ? local.ids["strict"] : local.ids["open"]`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, out := convertPolicies(t, tt.policies)
			require.True(t, ok, "conversion should succeed:\n%s", out)
			for i, id := range tt.wantIDs {
				assert.Contains(t, out, "id         = "+id, "element %d not kept whole:\n%s", i, out)
				assert.Contains(t, out, "precedence = "+strconv.Itoa(i+1), out)
			}
			assert.NotContains(t, out, "precedence = "+strconv.Itoa(len(tt.wantIDs)+1),
				"too many elements:\n%s", out)
		})
	}
}

// Comments between elements are not elements.
func TestConvertArrayAttributeToObjectArray_Comments(t *testing.T) {
	ok, out := convertPolicies(t, `[
    # primary
    "policy-1", # trailing
    // secondary
    "policy-2",
    /* last */ "policy-3",
  ]`)
	require.True(t, ok, out)
	for i, id := range []string{`"policy-1"`, `"policy-2"`, `"policy-3"`} {
		assert.Contains(t, out, "id         = "+id, out)
		assert.Contains(t, out, "precedence = "+strconv.Itoa(i+1), out)
	}
	assert.NotContains(t, out, "precedence = 4", "a comment became an element:\n%s", out)
	assert.NotContains(t, out, "id         = #", out)
	assert.NotContains(t, out, "id         = //", out)
}

// Anything that is not a plain list literal is left untouched rather than
// half-converted. Before the fix, concat() lost every list after the first
// and a for expression became { id = for ... }.
func TestConvertArrayAttributeToObjectArray_NotAListLiteral(t *testing.T) {
	for _, policies := range []string{
		`concat([var.a], [var.b])`,
		`[for p in var.policies : p.id]`,
		`var.policy_ids`,
		`local.ids[0]`,
		`[var.a, var.b][0]`,
		`[]`,
	} {
		t.Run(policies, func(t *testing.T) {
			ok, out := convertPolicies(t, policies)
			assert.False(t, ok, "should not convert:\n%s", out)
			assert.Contains(t, out, "policies = "+policies, "attribute must be left unchanged:\n%s", out)
		})
	}
}
