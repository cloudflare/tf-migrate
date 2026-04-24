package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

func newTestLogger() hclog.Logger { return hclog.NewNullLogger() }

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

func TestScanForInvalidAttributeReferences(t *testing.T) {
	refs := []transform.InvalidAttributeReference{
		{
			ResourceType: "cloudflare_zero_trust_tunnel_cloudflared",
			Attribute:    "tunnel_token",
			Suggestion:   "tunnel_token is not a valid attribute; use tunnel_secret instead",
		},
	}

	t.Run("detects_tunnel_token_reference", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "consumers.tf")
		content := `resource "vault_generic_secret" "token" {
  data_json = jsonencode({
    token = cloudflare_zero_trust_tunnel_cloudflared.my_tunnel.tunnel_token
  })
}
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}

		diags := scanForInvalidAttributeReferences(newTestLogger(), []string{file}, refs)
		if len(diags) != 1 {
			t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
		}
		if diags[0].Severity != hcl.DiagWarning {
			t.Errorf("expected DiagWarning, got %v", diags[0].Severity)
		}
		if !contains(diags[0].Summary, "tunnel_token") {
			t.Errorf("expected summary to mention tunnel_token, got: %s", diags[0].Summary)
		}
		if !contains(diags[0].Detail, "tunnel_secret") {
			t.Errorf("expected detail to mention tunnel_secret, got: %s", diags[0].Detail)
		}
	})

	t.Run("no_false_positive_for_valid_attribute", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "consumers.tf")
		// tunnel_secret is valid — must NOT produce a warning
		content := `resource "vault_generic_secret" "secret" {
  data_json = jsonencode({
    secret = cloudflare_zero_trust_tunnel_cloudflared.my_tunnel.tunnel_secret
  })
}
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}

		diags := scanForInvalidAttributeReferences(newTestLogger(), []string{file}, refs)
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics for valid attribute, got %d: %v", len(diags), diags)
		}
	})

	t.Run("multiple_matches_produce_multiple_warnings", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "consumers.tf")
		content := `resource "vault_generic_secret" "a" {
  data_json = jsonencode({
    a = cloudflare_zero_trust_tunnel_cloudflared.tunnel_a.tunnel_token
    b = cloudflare_zero_trust_tunnel_cloudflared.tunnel_b.tunnel_token
  })
}
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}

		diags := scanForInvalidAttributeReferences(newTestLogger(), []string{file}, refs)
		if len(diags) != 2 {
			t.Fatalf("expected 2 diagnostics (one per match), got %d", len(diags))
		}
	})

	t.Run("no_match_in_unrelated_resource_type", func(t *testing.T) {
		tmp := t.TempDir()
		file := filepath.Join(tmp, "consumers.tf")
		// Different resource type — must not match
		content := `resource "vault_generic_secret" "other" {
  data_json = jsonencode({
    tok = cloudflare_some_other_resource.foo.tunnel_token
  })
}
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}

		diags := scanForInvalidAttributeReferences(newTestLogger(), []string{file}, refs)
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics for different resource type, got %d", len(diags))
		}
	})
}
