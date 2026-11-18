package hcl

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findRefs(t *testing.T, src string) []string {
	return findRefsIterable(t, src, nil)
}

func findRefsIterable(t *testing.T, src string, iterable func(string, string) bool) []string {
	t.Helper()
	file, diags := hclsyntax.ParseConfig([]byte(src), "main.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)
	refs := FindInstanceAttributeReferences(file.Body.(*hclsyntax.Body), []byte(src),
		[]string{"data.cloudflare_dcv_delegation"}, iterable)
	var got []string
	for _, r := range refs {
		require.Equal(t, r.Attribute, string(r.NameRange.SliceBytes([]byte(src))), "NameRange must cover only the name")
		got = append(got, string(r.Range.SliceBytes([]byte(src))))
	}
	return got
}

func TestFindInstanceAttributeReferences(t *testing.T) {
	src := `
output "direct"   { value = data.cloudflare_dcv_delegation.a.id }
output "key"      { value = data.cloudflare_dcv_delegation.fe["a"].id }
output "each_key" { value = data.cloudflare_dcv_delegation.fe[each.key].hostname }
output "count"    { value = data.cloudflare_dcv_delegation.cnt[0].id }
output "splat"    { value = data.cloudflare_dcv_delegation.cnt[*].id }
output "legacy"   { value = data.cloudflare_dcv_delegation.cnt.*.id }
output "tmpl"     { value = "${data.cloudflare_dcv_delegation.a.id}.dcv.cloudflare.com" }
output "for"      { value = { for k, d in data.cloudflare_dcv_delegation.fe : k => d.id if d.zone_id != "" } }

resource "cloudflare_dns_record" "cname" {
  for_each = data.cloudflare_dcv_delegation.fe
  zone_id  = each.value.zone_id
  content  = "${each.value.id}.dcv.cloudflare.com"
  lifecycle {
    precondition {
      condition     = each.value.hostname != ""
      error_message = "x"
    }
  }
}

resource "cloudflare_ruleset" "dyn" {
  dynamic "rules" {
    for_each = data.cloudflare_dcv_delegation.fe
    iterator = d
    content {
      expression = d.value.id
      action_parameters { id = d.value.hostname }
    }
  }
}
`
	assert.Equal(t, []string{
		"data.cloudflare_dcv_delegation.a.id",
		`data.cloudflare_dcv_delegation.fe["a"].id`,
		"data.cloudflare_dcv_delegation.fe[each.key].hostname",
		"data.cloudflare_dcv_delegation.cnt[0].id",
		"data.cloudflare_dcv_delegation.cnt[*].id",
		"data.cloudflare_dcv_delegation.cnt.*.id",
		"data.cloudflare_dcv_delegation.a.id",
		"d.id",
		"d.zone_id",
		"each.value.zone_id",
		"each.value.id",
		"each.value.hostname",
		"d.value.id",
		"d.value.hostname",
	}, findRefs(t, src))
}

// Only an iteration over exactly type.name is followed. Anything else is not
// known to yield instances of the type.
func TestFindInstanceAttributeReferences_NotFollowed(t *testing.T) {
	src := `
output "other_type" { value = data.cloudflare_zone.a.id }
output "resource"   { value = cloudflare_dcv_delegation.a.id }
output "whole"      { value = data.cloudflare_dcv_delegation.a }
output "derived"    { value = [for d in values(data.cloudflare_dcv_delegation.fe) : d.id] }
output "other_for"  { value = [for d in var.things : d.id] }

resource "cloudflare_dns_record" "other_each" {
  for_each = var.things
  content  = each.value.id
}

resource "cloudflare_dns_record" "indexed_each" {
  for_each = data.cloudflare_dcv_delegation.fe["a"]
  content  = each.value.id
}
`
	assert.Empty(t, findRefs(t, src))
}

// Without count or for_each, an index or an iteration reaches the
// resource's attributes, not instances, so they are not followed.
func TestFindInstanceAttributeReferences_NotIterable(t *testing.T) {
	src := `
output "direct" { value = data.cloudflare_dcv_delegation.single.id }
output "splat"  { value = data.cloudflare_dcv_delegation.single[*].id }
output "index"  { value = data.cloudflare_dcv_delegation.single["zone_id"].id }
output "for"    { value = [for v in data.cloudflare_dcv_delegation.single : v.id] }
resource "cloudflare_dns_record" "r" {
  for_each = data.cloudflare_dcv_delegation.single
  content  = each.value.id
}
`
	none := func(string, string) bool { return false }
	assert.Equal(t, []string{
		"data.cloudflare_dcv_delegation.single.id",
		"data.cloudflare_dcv_delegation.single[*].id",
	}, findRefsIterable(t, src, none))
}
