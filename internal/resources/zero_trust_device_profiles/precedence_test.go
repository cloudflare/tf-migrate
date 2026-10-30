package zero_trust_device_profiles

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
)

// exprTokens parses "precedence = <expr>" and returns the tokens of <expr>.
func exprTokens(t *testing.T, expr string) hclwrite.Tokens {
	t.Helper()
	f, diags := hclwrite.ParseConfig([]byte("precedence = "+expr+"\n"), "t.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse %q: %s", expr, diags)
	attr := f.Body().GetAttribute("precedence")
	require.NotNil(t, attr, "no precedence attribute for %q", expr)
	return attr.Expr().BuildTokens(nil)
}

func TestLiteralNumber(t *testing.T) {
	t.Parallel()

	literals := map[string]float64{
		"0":     0,
		"10":    10,
		"100":   100,
		"100.5": 100.5,
	}
	for expr, want := range literals {
		t.Run("literal "+expr, func(t *testing.T) {
			got, ok := literalNumber(exprTokens(t, expr))
			require.True(t, ok, "%q should be a literal", expr)
			require.Equal(t, want, got)
		})
	}

	// Every one of these was folded to a literal before: gjson reads a leading
	// "index" as the number 0, giving 900, while "local" is not valid JSON at
	// all, so the old code kept its 100.0 default and gave 1000.
	expressions := []string{
		`index(local.warp_profiles_order, "a")`,
		`local.base_precedence`,
		`each.value.precedence`,
		`var.precedence`,
		`var.use_high ? 5 : 20`,
		`100 + var.offset`,
		`merge(local.a, local.b)["p"]`,
		`length(local.profiles)`,
		`-5`,
		`"100"`,
	}
	for _, expr := range expressions {
		t.Run("expression "+expr, func(t *testing.T) {
			_, ok := literalNumber(exprTokens(t, expr))
			require.False(t, ok, "%q must not be treated as a literal", expr)
		})
	}
}

func TestOffsetExpression(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{
		`index(local.warp_profiles_order, "a")`,
		`each.value.precedence`,
		`var.use_high ? 5 : 20`,
	} {
		t.Run(expr, func(t *testing.T) {
			got := string(offsetExpression(exprTokens(t, expr)).Bytes())
			require.Equal(t, "900+("+expr+")", got)
		})
	}
}
