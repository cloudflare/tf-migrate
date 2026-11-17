package main

import (
	"strings"
	"testing"
)

func TestRestorePhaseOneFile(t *testing.T) {
	input := `resource "cloudflare_zone" "cftftest_zone" {
  zone = "example.com"
}

# tf-migrate: resource "cloudflare_zone_settings_override" "cftftest_settings" {
# tf-migrate:   zone_id = cloudflare_zone.cftftest_zone.id
# tf-migrate:   settings {
# tf-migrate:     brotli = "on"
# tf-migrate:   }
# tf-migrate: }

removed {
  from = cloudflare_zone_settings_override.cftftest_settings
  lifecycle {
    destroy = false
  }
}

removed {
  from = cloudflare_record.cftftest_user_owned
}
`

	want := `resource "cloudflare_zone" "cftftest_zone" {
  zone = "example.com"
}

resource "cloudflare_zone_settings_override" "cftftest_settings" {
  zone_id = cloudflare_zone.cftftest_zone.id
  settings {
    brotli = "on"
  }
}

removed {
  from = cloudflare_record.cftftest_user_owned
}
`

	got := restorePhaseOneFile(input)
	if got != want {
		t.Fatalf("restorePhaseOneFile mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if n := strings.Count(got, "cloudflare_zone_settings_override.cftftest_settings"); n != 0 {
		t.Errorf("phase-1 removed {} block was not stripped (%d references remain)", n)
	}
}

// Two zone_settings_override resources in one file, back to back: each
// phase-1 removed {} block must be stripped, and nothing else touched.
func TestRestorePhaseOneFileTwoBlocks(t *testing.T) {
	input := `# tf-migrate: resource "cloudflare_zone_settings_override" "a" {
# tf-migrate:   zone_id = var.zone_a
# tf-migrate: }

removed {
  from = cloudflare_zone_settings_override.a
  lifecycle {
    destroy = false
  }
}

# tf-migrate: resource "cloudflare_zone_settings_override" "b" {
# tf-migrate:   zone_id = var.zone_b
# tf-migrate: }

removed {
  from = cloudflare_zone_settings_override.b
  lifecycle {
    destroy = false
  }
}
`

	want := `resource "cloudflare_zone_settings_override" "a" {
  zone_id = var.zone_a
}

resource "cloudflare_zone_settings_override" "b" {
  zone_id = var.zone_b
}
`

	got := restorePhaseOneFile(input)
	if got != want {
		t.Fatalf("restorePhaseOneFile mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if n := strings.Count(got, "removed {"); n != 0 {
		t.Errorf("%d phase-1 removed {} block(s) were not stripped", n)
	}
}
