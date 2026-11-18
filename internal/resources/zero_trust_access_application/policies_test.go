package zero_trust_access_application

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

func transformPolicies(t *testing.T, policies string) (string, hcl.Diagnostics) {
	t.Helper()
	input := `resource "cloudflare_zero_trust_access_application" "app" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "app"
  domain     = "app.example.com"
  type       = "self_hosted"
  policies   = ` + policies + `
}`
	file, diags := hclwrite.ParseConfig([]byte(input), "app.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)

	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "app.tf",
		Diagnostics:   make(hcl.Diagnostics, 0),
		Metadata:      make(map[string]interface{}),
		SourceVersion: "v4",
		TargetVersion: "v5",
	}
	result, err := NewV4ToV5Migrator().TransformConfig(ctx, file.Body().Blocks()[0])
	require.NoError(t, err)

	var out strings.Builder
	for _, b := range result.Blocks {
		out.Write(hclwrite.Format(b.BuildTokens(nil).Bytes()))
	}
	return out.String(), ctx.Diagnostics
}

func policiesWarning(diags hcl.Diagnostics) *hcl.Diagnostic {
	for _, d := range diags {
		if strings.Contains(d.Summary, "policies not converted") {
			return d
		}
	}
	return nil
}

func TestPolicies_ElementsWithBracketsStayWhole(t *testing.T) {
	out, diags := transformPolicies(t,
		`[cloudflare_zero_trust_access_policy.allow[each.key].id, lookup(var.ids, "deny")]`)

	assert.Contains(t, out, "id         = cloudflare_zero_trust_access_policy.allow[each.key].id")
	assert.Contains(t, out, `id         = lookup(var.ids, "deny")`)
	assert.Contains(t, out, "precedence = 2")
	assert.NotContains(t, out, "precedence = 3")
	assert.Nil(t, policiesWarning(diags))
}

func TestPolicies_NonLiteralIsLeftAndWarned(t *testing.T) {
	for _, policies := range []string{
		`concat([var.a], [var.b])`,
		`[for p in var.policies : p.id]`,
		`var.policy_ids`,
	} {
		t.Run(policies, func(t *testing.T) {
			out, diags := transformPolicies(t, policies)
			assert.Regexp(t, `policies\s+= `+regexp.QuoteMeta(policies)+`\n`, out, "must be left unchanged")

			w := policiesWarning(diags)
			require.NotNil(t, w, "expected a warning, got %v", diags)
			assert.Equal(t, hcl.DiagWarning, w.Severity)
			assert.Contains(t, w.Summary, "cloudflare_zero_trust_access_application.app")
			assert.Contains(t, w.Detail, "[for i, id in "+policies+" : { id = id, precedence = i + 1 }]")
		})
	}
}

func TestPolicies_EmptyListIsNotWarned(t *testing.T) {
	_, diags := transformPolicies(t, `[]`)
	assert.Nil(t, policiesWarning(diags))
}

// A second run over migrated output must leave the v5 shape alone. It used to
// wrap each object again: id = { id = ..., precedence = 1 }.
func TestPolicies_AlreadyConvertedIsLeftAlone(t *testing.T) {
	policies := `[
    {
      id         = cloudflare_zero_trust_access_policy.a.id
      precedence = 1
    },
    {
      id         = cloudflare_zero_trust_access_policy.b.id
      precedence = 2
    },
  ]`
	out, diags := transformPolicies(t, policies)
	assert.NotContains(t, out, "id = {", "objects were wrapped again:\n%s", out)
	assert.Equal(t, 2, strings.Count(out, "precedence = "), out)
	assert.Contains(t, out, "id         = cloudflare_zero_trust_access_policy.b.id")
	assert.Nil(t, policiesWarning(diags))
}
