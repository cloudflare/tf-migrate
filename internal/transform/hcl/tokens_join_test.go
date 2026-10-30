package hcl

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// attrTokensFor parses `v = <expr>` and returns the tokens of the value.
func attrTokensFor(t *testing.T, expr string) hclwrite.Tokens {
	t.Helper()
	src := "v = " + expr + "\n"
	f, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("failed to parse %q: %s", src, diags.Error())
	}
	attr := f.Body().GetAttribute("v")
	if attr == nil {
		t.Fatalf("no attribute parsed from %q", src)
	}
	return attr.Expr().BuildTokens(nil)
}

// TestJoinQuotedLiteral_PercentEncoded is the direct regression test for
// BUGS-2160: HCL splits string literals at '%' and '$', so an implementation
// that returns the first TokenQuotedLit truncates the value there. For a URL
// blocklist that silently shortens an entry so it no longer matches.
func TestJoinQuotedLiteral_PercentEncoded(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "BUGS-2160 reported value",
			expr: `"https://onedrive.live.com/download?authkey=ABC&resid=BD6077F58A73009F%21108"`,
			want: `https://onedrive.live.com/download?authkey=ABC&resid=BD6077F58A73009F%21108`,
		},
		{
			name: "percent at start",
			expr: `"%21leading"`,
			want: `%21leading`,
		},
		{
			name: "percent at end",
			expr: `"trailing%"`,
			want: `trailing%`,
		},
		{
			name: "multiple percent-encoded sequences",
			expr: `"https://x.test/a%20b%2Fc%3Fd"`,
			want: `https://x.test/a%20b%2Fc%3Fd`,
		},
		{
			name: "escaped percent",
			expr: `"100%%"`,
			want: `100%%`,
		},
		{
			name: "escaped dollar",
			expr: `"cost$$5"`,
			want: `cost$$5`,
		},
		{
			name: "dollar not starting interpolation",
			expr: `"a$b"`,
			want: `a$b`,
		},
		{
			name: "logpull_options style query string",
			expr: `"fields=RayID,EdgeStartTimestamp&timestamps=rfc3339&filter=%7B%22key%22%3A1%7D"`,
			want: `fields=RayID,EdgeStartTimestamp&timestamps=rfc3339&filter=%7B%22key%22%3A1%7D`,
		},
		{
			name: "no special characters",
			expr: `"https://example.com/plain"`,
			want: `https://example.com/plain`,
		},
		{
			name: "empty string",
			expr: `""`,
			want: ``,
		},
		{
			name: "backslash escapes are not split",
			expr: `"line\nnext"`,
			want: `line\nnext`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := JoinQuotedLiteral(attrTokensFor(t, tt.expr))
			if !ok {
				t.Fatalf("JoinQuotedLiteral(%s) returned ok=false, want a plain string", tt.expr)
			}
			if got != tt.want {
				t.Errorf("JoinQuotedLiteral(%s)\n got: %q\nwant: %q", tt.expr, got, tt.want)
			}
		})
	}
}

// TestJoinQuotedLiteral_RejectsNonPlainStrings asserts that anything which
// cannot be faithfully represented as a string reports ok=false, so callers
// preserve the raw tokens instead of silently flattening the expression.
func TestJoinQuotedLiteral_RejectsNonPlainStrings(t *testing.T) {
	exprs := []string{
		`var.record_type`,
		`local.foo.bar`,
		`"prefix-${var.name}"`,
		`"${var.name}"`,
		`"%{ if true }yes%{ endif }"`,
		`format("%s", var.x)`,
		`42`,
		`true`,
		`["a", "b"]`,
		`{ a = "b" }`,
	}

	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			got, ok := JoinQuotedLiteral(attrTokensFor(t, expr))
			if ok {
				t.Errorf("JoinQuotedLiteral(%s) returned ok=true with %q, want ok=false so the caller preserves tokens", expr, got)
			}
		})
	}
}

// TestExtractStringFromAttribute_PercentEncoded covers the shared public
// helper used by 40+ call sites across 18 resources, which had the same
// first-token-only truncation.
func TestExtractStringFromAttribute_PercentEncoded(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{`"https://x.test/a?resid=ABC%21108"`, `https://x.test/a?resid=ABC%21108`},
		{`"fields=a&b%2Fc&timestamps=rfc3339"`, `fields=a&b%2Fc&timestamps=rfc3339`},
		{`"price is 100%"`, `price is 100%`},
		{`"plain"`, `plain`},
		// Long-standing traversal behaviour, deliberately unchanged: only the
		// first identifier is returned.
		{`var.record_type`, `var`},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			src := "v = " + tt.expr + "\n"
			f, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("parse error: %s", diags.Error())
			}
			got := ExtractStringFromAttribute(f.Body().GetAttribute("v"))
			if got != tt.want {
				t.Errorf("ExtractStringFromAttribute(%s)\n got: %q\nwant: %q", tt.expr, got, tt.want)
			}
		})
	}
}

// TestMergeAttributeAndBlocksToObjectArray_PreservesPercentEncoding is the
// end-to-end regression test for BUGS-2160 at the call site that corrupted the
// SIRT blocklist: zero_trust_list items being reshaped into the v5 object form.
func TestMergeAttributeAndBlocksToObjectArray_PreservesPercentEncoding(t *testing.T) {
	const urlWithPercent = `https://onedrive.live.com/download?authkey=ABC&resid=BD6077F58A73009F%21108`

	src := `resource "cloudflare_teams_list" "test" {
  name = "blocklist"
  type = "URL"
  items = ["` + urlWithPercent + `", "https://plain.test/x"]

  items_with_description {
    value       = "https://described.test/a%2Fb"
    description = "has percent encoding"
  }
}
`

	f, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse error: %s", diags.Error())
	}

	block := f.Body().Blocks()[0]
	MergeAttributeAndBlocksToObjectArray(
		block.Body(),
		"items",
		"items_with_description",
		"items",
		"value",
		[]string{"description"},
		true,
	)

	out := string(hclwrite.Format(f.Bytes()))

	for _, want := range []string{
		urlWithPercent,
		`https://described.test/a%2Fb`,
		`https://plain.test/x`,
	} {
		if !contains(out, want) {
			t.Errorf("output lost or truncated %q\n\n--- output ---\n%s", want, out)
		}
	}

	// The truncated prefix must not appear as a complete value.
	if contains(out, `"https://onedrive.live.com/download?authkey=ABC&resid=BD6077F58A73009F"`) {
		t.Errorf("value was truncated at %%21\n\n--- output ---\n%s", out)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
