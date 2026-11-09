package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestScanForIndexedInvalidAttributeReferences(t *testing.T) {
	file := filepath.Join(t.TempDir(), "indexed.tf")
	content := `
locals {
  direct = data.cloudflare_api_token_permission_groups_list.all.zone["DNS Read"]
  counted = data.cloudflare_api_token_permission_groups_list.all[0].zone["DNS Read"]
  keyed = data.cloudflare_api_token_permission_groups_list.all["zone"].zone["DNS Read"]
  dynamic = data.cloudflare_api_token_permission_groups_list.all[each.key].zone["DNS Read"]
  nested = data.cloudflare_api_token_permission_groups_list.all[local.keys[0]].zone["DNS Read"]
  valid = data.cloudflare_api_token_permission_groups_list.all[each.key].result
  longer_attribute = data.cloudflare_api_token_permission_groups_list.all[each.key].zone_extra
  text = "data.cloudflare_api_token_permission_groups_list.all[each.key].zone"
  # data.cloudflare_api_token_permission_groups_list.all[each.key].zone
}`
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	refs := []transform.InvalidAttributeReference{{
		ResourceType: "data.cloudflare_api_token_permission_groups_list",
		Attribute:    "zone", Suggestion: "Use result with a name and scope filter",
	}}
	diags := scanForInvalidAttributeReferences(newTestLogger(), []string{file}, refs)
	if len(diags) != 5 {
		t.Fatalf("expected exactly five real references, got %d: %v", len(diags), diags)
	}
	for _, diag := range diags {
		if diag.Subject == nil || diag.Subject.Filename != "indexed.tf" {
			t.Errorf("missing source location: %#v", diag.Subject)
		}
		if !contains(diag.Detail, "In indexed.tf") {
			t.Errorf("missing filename: %s", diag.Detail)
		}
	}
}
