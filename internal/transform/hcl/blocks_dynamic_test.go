package hcl

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFirstBlock(t *testing.T, src string) *hclwrite.Block {
	t.Helper()
	f, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse failed: %v", diags)
	require.NotEmpty(t, f.Body().Blocks(), "no blocks parsed")
	return f.Body().Blocks()[0]
}

func TestFindDynamicBlockFor(t *testing.T) {
	src := `resource "x" "y" {
  dynamic "items" {
    for_each = var.items
    content {
      id = items.key
    }
  }
}`
	body := parseFirstBlock(t, src).Body()

	// A dynamic block's type is "dynamic" and the generated type is its label,
	// so the ordinary block lookup must not find it.
	assert.Empty(t, FindBlocksByType(body, "items"))

	assert.NotNil(t, FindDynamicBlockFor(body, "items"))
	assert.True(t, HasDynamicBlockFor(body, "items"))
	assert.Nil(t, FindDynamicBlockFor(body, "other"))
	assert.False(t, HasDynamicBlockFor(body, "other"))
}

func TestRewriteDynamicIteratorRefs(t *testing.T) {
	cases := []struct{ name, expr, want string }{
		{"key becomes key var", "headers.key", "k"},
		{"value becomes value var", "headers.value", "v"},
		{"further traversal preserved", "headers.value.name", "v.name"},
		{"unrelated reference untouched", "var.headers", "var.headers"},
		{"attribute named like the iterator untouched", "var.headers.value", "var.headers.value"},
		{"different iterator untouched", "rules.key", "rules.key"},
		{"non key/value attribute untouched", "headers.other", "headers.other"},
		{"inside interpolation", `"${headers.key}-suffix"`, `"${k}-suffix"`},
		{"both in one expression", "join(headers.key, headers.value)", "join(k, v)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block := parseFirstBlock(t, "x {\n  a = "+tc.expr+"\n}")
			attr := block.Body().GetAttribute("a")
			got := RewriteDynamicIteratorRefs(attr.Expr().BuildTokens(nil), "headers", "k", "v")
			assert.Equal(t, tc.want, strings.TrimSpace(string(got.Bytes())))
		})
	}
}

// A dynamic block must become a for expression. Converting it as though
// "dynamic" were a field name discards the label naming the generated type and
// leaves the iterator references without a loop to bind to.
func TestConvertDynamicBlockToForList(t *testing.T) {
	src := `resource "x" "y" {
  dynamic "managed_request_headers" {
    for_each = var.headers
    content {
      id      = managed_request_headers.key
      enabled = managed_request_headers.value
    }
  }
}`
	block := parseFirstBlock(t, src)
	dyn := block.Body().Blocks()[0]

	ok, reason := ConvertDynamicBlockToForList(block.Body(), dyn, "managed_request_headers")
	require.True(t, ok, "conversion should succeed, got: %s", reason)

	out := string(hclwrite.Format(block.BuildTokens(nil).Bytes()))
	assert.Contains(t, out, "for k, v in var.headers")
	assert.Contains(t, out, "id      = k")
	assert.Contains(t, out, "enabled = v")
	assert.NotContains(t, out, "dynamic", "the dynamic block must be consumed, got:\n%s", out)
}

func TestConvertDynamicBlockToForListCustomIterator(t *testing.T) {
	src := `resource "x" "y" {
  dynamic "items" {
    for_each = var.items
    iterator = "entry"
    content {
      id = entry.value
    }
  }
}`
	block := parseFirstBlock(t, src)
	dyn := block.Body().Blocks()[0]

	ok, reason := ConvertDynamicBlockToForList(block.Body(), dyn, "items")
	require.True(t, ok, "conversion should succeed, got: %s", reason)

	out := string(hclwrite.Format(block.BuildTokens(nil).Bytes()))
	assert.Contains(t, out, "id = v", "the custom iterator must be rebound, got:\n%s", out)
}

// Guessing at a shape we cannot derive would produce config the user never
// wrote, so these cases must refuse and report why.
func TestConvertDynamicBlockToForListRefusesUnsafeInput(t *testing.T) {
	cases := []struct{ name, src, wantReason string }{
		{
			name: "no for_each",
			src: `resource "x" "y" {
  dynamic "h" {
    content {
      id = h.key
    }
  }
}`,
			wantReason: "for_each",
		},
		{
			name: "no content block",
			src: `resource "x" "y" {
  dynamic "h" {
    for_each = var.z
  }
}`,
			wantReason: "content block",
		},
		{
			name: "nested blocks inside content",
			src: `resource "x" "y" {
  dynamic "h" {
    for_each = var.z
    content {
      nested {
        a = 1
      }
    }
  }
}`,
			wantReason: "nested blocks",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block := parseFirstBlock(t, tc.src)
			dyn := block.Body().Blocks()[0]

			ok, reason := ConvertDynamicBlockToForList(block.Body(), dyn, "h")
			assert.False(t, ok, "must refuse rather than emit wrong config")
			assert.Contains(t, reason, tc.wantReason)

			out := string(block.BuildTokens(nil).Bytes())
			assert.Contains(t, out, "dynamic", "a refused block must be left in place")
		})
	}
}

// Writing an empty array next to a dynamic block declares "none of these",
// which discards everything the dynamic block generates.
func TestConvertBlocksToArrayAttributeSkipsEmptyDefaultForDynamic(t *testing.T) {
	src := `resource "x" "y" {
  dynamic "items" {
    for_each = var.items
    content {
      id = items.key
    }
  }
}`
	block := parseFirstBlock(t, src)
	ConvertBlocksToArrayAttribute(block.Body(), "items", true)

	out := string(hclwrite.Format(block.BuildTokens(nil).Bytes()))
	assert.NotRegexp(t, `items\s+= \[\]`, out,
		"must not declare an empty list alongside a dynamic block, got:\n%s", out)
	assert.Contains(t, out, `dynamic "items"`, "the dynamic block must be left intact")
}

// With no blocks of either kind the empty-array default is still correct,
// since v5 requires the attribute to be present.
func TestConvertBlocksToArrayAttributeStillDefaultsWithoutDynamic(t *testing.T) {
	block := parseFirstBlock(t, "resource \"x\" \"y\" {\n  zone_id = \"abc\"\n}")
	ConvertBlocksToArrayAttribute(block.Body(), "items", true)

	out := string(hclwrite.Format(block.BuildTokens(nil).Bytes()))
	assert.Regexp(t, `items\s+= \[\]`, out)
}
