package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Attribute renames must reach every form of reference to an instance, not
// only <type>.<name>.<attr>. Uses the dcv_delegation data source rename
// id -> uuid and its invalid hostname attribute.
func TestAttributeRenameReferenceForms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.tf")
	input := `data "cloudflare_dcv_delegation" "fe" {
  for_each = var.zones
  zone_id  = each.value
}

data "cloudflare_dcv_delegation" "cnt" {
  count   = 1
  zone_id = var.zone_id
}

data "cloudflare_dcv_delegation" "single" {
  zone_id = var.zone_id
}

output "key"    { value = data.cloudflare_dcv_delegation.fe["a"].id }
output "count"  { value = data.cloudflare_dcv_delegation.cnt[0].id }
output "splat"  { value = data.cloudflare_dcv_delegation.cnt[*].id }
output "legacy" { value = data.cloudflare_dcv_delegation.cnt.*.id }
output "for"    { value = [for d in data.cloudflare_dcv_delegation.fe : d.id] }
output "single" { value = data.cloudflare_dcv_delegation.single.id }
output "attrs"  { value = [for v in data.cloudflare_dcv_delegation.single : v.id] }

resource "cloudflare_dns_record" "cname" {
  for_each = data.cloudflare_dcv_delegation.fe
  zone_id  = each.value.zone_id
  name     = "_acme-challenge"
  type     = "CNAME"
  content  = "${each.value.id}.dcv.cloudflare.com"
  ttl      = 1
}

resource "cloudflare_dns_record" "host" {
  for_each = data.cloudflare_dcv_delegation.fe
  zone_id  = each.value.zone_id
  name     = "_acme-challenge"
  type     = "CNAME"
  content  = each.value.hostname
  ttl      = 1
}
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config{configDir: dir, sourceVersion: "v4", targetVersion: "v5"}
	diags, err := applyGlobalPostprocessing(newTestLogger(), cfg, []string{path})
	if err != nil {
		t.Fatalf("applyGlobalPostprocessing failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)

	for _, want := range []string{
		`data.cloudflare_dcv_delegation.fe["a"].uuid`,
		`data.cloudflare_dcv_delegation.cnt[0].uuid`,
		`data.cloudflare_dcv_delegation.cnt[*].uuid`,
		`data.cloudflare_dcv_delegation.cnt.*.uuid`,
		`[for d in data.cloudflare_dcv_delegation.fe : d.uuid]`,
		`data.cloudflare_dcv_delegation.single.uuid`,
		`"${each.value.uuid}.dcv.cloudflare.com"`,
		// single has no count or for_each, so iterating it yields its
		// attributes, not instances: v.id is left alone.
		`[for v in data.cloudflare_dcv_delegation.single : v.id]`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}

	var hostWarned bool
	for _, d := range diags {
		if strings.Contains(d.Summary, "each.value.hostname") {
			hostWarned = true
		}
	}
	if !hostWarned {
		t.Errorf("expected a warning for each.value.hostname, got %v", diags)
	}
}
