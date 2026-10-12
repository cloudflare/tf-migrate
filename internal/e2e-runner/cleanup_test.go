package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsStateShowIDNull(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name: "corrupted state (known bug signature)",
			output: `# module.leaked_credential_check_rule.cloudflare_leaked_credential_check_rule.basic:
resource "cloudflare_leaked_credential_check_rule" "basic" {
    id       = null
    password = null
    username = null
    zone_id  = "cd581854c1f59f8c686ee796d0eddce2"
}
`,
			want: true,
		},
		{
			name: "healthy state with a real id",
			output: `# module.leaked_credential_check_rule.cloudflare_leaked_credential_check_rule.basic:
resource "cloudflare_leaked_credential_check_rule" "basic" {
    id       = "53d55269c789408baa1b052e2239eacd"
    password = (sensitive value)
    username = (sensitive value)
    zone_id  = "cd581854c1f59f8c686ee796d0eddce2"
}
`,
			want: false,
		},
		{
			name:   "unrelated null field does not false-positive",
			output: "resource \"x\" \"y\" {\n    name = null\n    id   = \"real-id\"\n}\n",
			want:   false,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStateShowIDNull(tt.output)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPureCreateAddresses(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "pure create is included",
			output: "  # module.queue.cloudflare_queue.minimal will be created\n",
			want:   []string{"module.queue.cloudflare_queue.minimal"},
		},
		{
			name: "update in-place, destroy, and import are excluded",
			output: "" +
				"  # module.list.cloudflare_list.redirect_list will be updated in-place\n" +
				"  # module.x.cloudflare_x.y will be destroyed\n" +
				"  # module.argo.cloudflare_argo_tiered_caching.both_with_lifecycle_tiered will be imported\n",
			want: nil,
		},
		{
			name:   "replace (must be replaced) is excluded — a real resource already exists",
			output: "  # module.certificate_pack.cloudflare_certificate_pack.by_config[\"google-90d\"] must be replaced\n",
			want:   nil,
		},
		{
			name: "mixed set returns only the pure creates, in order",
			output: "" +
				"  # module.a.t.one will be created\n" +
				"  # module.a.t.two will be updated in-place\n" +
				"  # module.a.t.three must be replaced\n" +
				"  # module.a.t.four will be created\n",
			want: []string{"module.a.t.one", "module.a.t.four"},
		},
		{
			name:   "a line merely containing \"will be created\" as trailing prose is not matched without the leading '#' header",
			output: "This resource will be created automatically.\n",
			want:   nil,
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPureCreateAddresses(tt.output)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindImportToAddresses(t *testing.T) {
	t.Run("root-hoisted import block (already module-qualified) is used as-is", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "main.tf", `
import {
  to = module.argo.cloudflare_argo_tiered_caching.both_with_lifecycle_tiered
  id = "cd581854c1f59f8c686ee796d0eddce2"
}
`)
		got, err := findImportToAddresses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "module.argo.cloudflare_argo_tiered_caching.both_with_lifecycle_tiered"
		if !got[want] {
			t.Errorf("expected %q to be protected, got %v", want, got)
		}
	})

	t.Run("module-local import block gets the module prefix reconstructed", func(t *testing.T) {
		dir := t.TempDir()
		mustMkdir(t, filepath.Join(dir, "leaked_credential_check_rule"))
		writeTFFile(t, filepath.Join(dir, "leaked_credential_check_rule"), "leaked_credential_check_rule.tf", `
import {
  to = cloudflare_leaked_credential_check_rule.basic
  id = "zone123/detection456"
}
`)
		got, err := findImportToAddresses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "module.leaked_credential_check_rule.cloudflare_leaked_credential_check_rule.basic"
		if !got[want] {
			t.Errorf("expected %q to be protected, got %v", want, got)
		}
	})

	t.Run("no import blocks returns an empty, non-nil-error result", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "main.tf", `
resource "cloudflare_queue" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "test"
}
`)
		got, err := findImportToAddresses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected no protected addresses, got %v", got)
		}
	})

	t.Run("moved blocks are ignored by the import scanner", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "main.tf", `
moved {
  from = cloudflare_record.example
  to   = cloudflare_dns_record.example
}
`)
		got, err := findImportToAddresses(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected moved blocks to be ignored, got %v", got)
		}
	})
}

// writeTFFile is a small helper shared by this file's tests to write a .tf
// file under dir, creating parent directories if needed.
func writeTFFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dir, err)
	}
}
