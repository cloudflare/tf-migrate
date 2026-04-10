package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceResourceTypeRefsSkippingMovedBlocks(t *testing.T) {
	input := `output "keep_old" {
  value = cloudflare_access_policy.app_scoped.id
}

output "rename" {
  value = cloudflare_access_policy.account_level.id
}

removed {
  from = cloudflare_access_policy.app_scoped
  lifecycle {
    destroy = false
  }
}

moved {
  from = cloudflare_access_policy.some_other
  to   = cloudflare_zero_trust_access_policy.some_other
}
`

	excluded := map[string]struct{}{"app_scoped": {}}
	got := replaceResourceTypeRefsSkippingMovedBlocks(
		input,
		"cloudflare_access_policy",
		"cloudflare_zero_trust_access_policy",
		excluded,
	)

	if !contains(got, "cloudflare_access_policy.app_scoped.id") {
		t.Fatalf("expected app_scoped reference to remain old type, got:\n%s", got)
	}

	if !contains(got, "cloudflare_zero_trust_access_policy.account_level.id") {
		t.Fatalf("expected account_level reference to be renamed, got:\n%s", got)
	}

	if !contains(got, "from = cloudflare_access_policy.app_scoped") {
		t.Fatalf("expected removed block from-address to remain unchanged, got:\n%s", got)
	}

	if !contains(got, "from = cloudflare_access_policy.some_other") {
		t.Fatalf("expected moved block from-address to remain unchanged, got:\n%s", got)
	}
}

func TestCollectRemovedRefsByType(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "main.tf")

	content := `removed {
  from = cloudflare_access_policy.app_scoped
  lifecycle { destroy = false }
}

removed {
  from = cloudflare_zone_settings_override.legacy
  lifecycle { destroy = false }
}

moved {
  from = cloudflare_access_policy.other
  to   = cloudflare_zero_trust_access_policy.other
}
`

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed writing fixture: %v", err)
	}

	refs, err := collectRemovedRefsByType([]string{file})
	if err != nil {
		t.Fatalf("collectRemovedRefsByType returned error: %v", err)
	}

	if _, ok := refs["cloudflare_access_policy"]["app_scoped"]; !ok {
		t.Fatalf("expected cloudflare_access_policy.app_scoped in removed refs, got: %#v", refs)
	}

	if _, ok := refs["cloudflare_zone_settings_override"]["legacy"]; !ok {
		t.Fatalf("expected cloudflare_zone_settings_override.legacy in removed refs, got: %#v", refs)
	}

	if _, ok := refs["cloudflare_access_policy"]["other"]; ok {
		t.Fatalf("did not expect moved block address to be collected as removed ref: %#v", refs)
	}
}
