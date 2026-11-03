package zero_trust_dlp_custom_profile

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic custom profile with single entry",
				Input: `resource "cloudflare_dlp_profile" "example" {
  account_id          = "123456789"
  name                = "Credit Card Profile"
  description         = "Profile for detecting credit cards"
  type                = "custom"
  allowed_match_count = 5

  entry {
    id      = "abc123"
    name    = "Visa Card"
    enabled = true
    pattern {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_custom_profile" "example" {
  account_id          = "123456789"
  name                = "Credit Card Profile"
  description         = "Profile for detecting credit cards"
  allowed_match_count = 5

  entries = [{
    name    = "Visa Card"
    enabled = true
    pattern = {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
  }]
}

moved {
  from = cloudflare_dlp_profile.example
  to   = cloudflare_zero_trust_dlp_custom_profile.example
}`,
			},
			{
				Name: "Multiple entries",
				Input: `resource "cloudflare_dlp_profile" "multi" {
  account_id          = "123456789"
  name                = "Multi Pattern Profile"
  type                = "custom"
  allowed_match_count = 10

  entry {
    id      = "entry1"
    name    = "Pattern One"
    enabled = true
    pattern {
      regex = "test1"
    }
  }

  entry {
    name    = "Pattern Two"
    enabled = false
    pattern {
      regex      = "test2"
      validation = "luhn"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_custom_profile" "multi" {
  account_id          = "123456789"
  name                = "Multi Pattern Profile"
  allowed_match_count = 10

  entries = [{
    name    = "Pattern One"
    enabled = true
    pattern = {
      regex = "test1"
    }
    }, {
    name    = "Pattern Two"
    enabled = false
    pattern = {
      regex      = "test2"
      validation = "luhn"
    }
  }]
}

moved {
  from = cloudflare_dlp_profile.multi
  to   = cloudflare_zero_trust_dlp_custom_profile.multi
}`,
			},
			{
				Name: "Minimal profile",
				Input: `resource "cloudflare_dlp_profile" "minimal" {
  account_id          = "123"
  name                = "Minimal"
  type                = "custom"
  allowed_match_count = 1

  entry {
    name    = "Simple"
    enabled = true
    pattern {
      regex = "[0-9]{4}"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_custom_profile" "minimal" {
  account_id          = "123"
  name                = "Minimal"
  allowed_match_count = 1

  entries = [{
    name    = "Simple"
    enabled = true
    pattern = {
      regex = "[0-9]{4}"
    }
  }]
}

moved {
  from = cloudflare_dlp_profile.minimal
  to   = cloudflare_zero_trust_dlp_custom_profile.minimal
}`,
			},
			{
				Name: "Profile without description",
				Input: `resource "cloudflare_dlp_profile" "no_desc" {
  account_id          = "123"
  name                = "No Description"
  type                = "custom"
  allowed_match_count = 0

  entry {
    name    = "Test"
    enabled = false
    pattern {
      regex = "test"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_custom_profile" "no_desc" {
  account_id          = "123"
  name                = "No Description"
  allowed_match_count = 0

  entries = [{
    name    = "Test"
    enabled = false
    pattern = {
      regex = "test"
    }
  }]
}

moved {
  from = cloudflare_dlp_profile.no_desc
  to   = cloudflare_zero_trust_dlp_custom_profile.no_desc
}`,
			},
			{
				Name: "Zero Trust DLP profile v4 name",
				Input: `resource "cloudflare_zero_trust_dlp_profile" "example" {
  account_id          = "123456789"
  name                = "Test Profile"
  description         = "Test with zero_trust name"
  type                = "custom"
  allowed_match_count = 3

  entry {
    id      = "test-id"
    name    = "Test Entry"
    enabled = true
    pattern {
      regex = "test[0-9]+"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_custom_profile" "example" {
  account_id          = "123456789"
  name                = "Test Profile"
  description         = "Test with zero_trust name"
  allowed_match_count = 3

  entries = [{
    name    = "Test Entry"
    enabled = true
    pattern = {
      regex = "test[0-9]+"
    }
  }]
}

moved {
  from = cloudflare_zero_trust_dlp_profile.example
  to   = cloudflare_zero_trust_dlp_custom_profile.example
}`,
			},
			{
				Name: "Predefined profile migration - profile_id from resource id attribute",
				Input: `resource "cloudflare_dlp_profile" "predefined" {
  id                  = "aws-keys-uuid"
  account_id          = "123456789"
  name                = "AWS Keys"
  type                = "predefined"
  allowed_match_count = 3

  entry {
    id      = "aws-access-key"
    name    = "AWS Access Key ID"
    enabled = true
  }

  entry {
    id      = "aws-secret-key"
    name    = "AWS Secret Access Key"
    enabled = true
  }

  entry {
    id      = "aws-session-token"
    name    = "AWS Session Token"
    enabled = false
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "predefined" {
  account_id          = "123456789"
  allowed_match_count = 3

  enabled_entries = ["aws-access-key", "aws-secret-key"]
  profile_id      = "aws-keys-uuid"
}

moved {
  from = cloudflare_dlp_profile.predefined
  to   = cloudflare_zero_trust_dlp_predefined_profile.predefined
}`,
			},
			{
				Name: "Predefined profile migration - profile_id from import-address annotation",
				Input: `# tf-migrate:import-address=123456789/c8932cc4-3312-4152-8041-f3f257122dc4
resource "cloudflare_dlp_profile" "creds" {
  account_id          = "123456789"
  name                = "Credentials and Secrets"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "d8fcfc9c-773c-405e-8426-21ecbb67ba93"
    name    = "Amazon AWS Access Key ID"
    enabled = false
  }

  entry {
    id      = "2c0e33e1-71da-40c8-aad3-32e674ad3d96"
    name    = "Amazon AWS Secret Access Key"
    enabled = false
  }
}`,
				Expected: `# tf-migrate:import-address=123456789/c8932cc4-3312-4152-8041-f3f257122dc4
resource "cloudflare_zero_trust_dlp_predefined_profile" "creds" {
  account_id          = "123456789"
  profile_id          = "c8932cc4-3312-4152-8041-f3f257122dc4"
  allowed_match_count = 0

  enabled_entries = []
}

moved {
  from = cloudflare_dlp_profile.creds
  to   = cloudflare_zero_trust_dlp_predefined_profile.creds
}`,
			},
			{
				Name: "Predefined profile - expression id and enabled are kept (BUGS-2156)",
				Input: `resource "cloudflare_dlp_profile" "health" {
  id                  = "583a2366-b16e-4f5c-90db-494710f4f71c"
  account_id          = var.account_id
  name                = "Health Information"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = local.fda_drug_names
    name    = "FDA Drug Names"
    enabled = true
  }

  entry {
    id      = "9237e83f-fde6-4ff5-84de-0493d7c354d6"
    name    = "FDA Active Ingredients"
    enabled = var.fda_enabled
  }

  entry {
    id      = "5a38ac66-a939-4f66-ae12-50d67e7dcc92"
    name    = "ICD-10"
    enabled = false
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "health" {
  account_id          = var.account_id
  allowed_match_count = 0

  enabled_entries = compact([(local.fda_drug_names), (var.fda_enabled) ? "9237e83f-fde6-4ff5-84de-0493d7c354d6" : null])
  profile_id      = "583a2366-b16e-4f5c-90db-494710f4f71c"
}

moved {
  from = cloudflare_dlp_profile.health
  to   = cloudflare_zero_trust_dlp_predefined_profile.health
}`,
			},
			{
				Name: "Predefined profile - dynamic entry blocks become a for expression (BUGS-2156)",
				Input: `resource "cloudflare_zero_trust_dlp_profile" "health" {
  id                  = "583a2366-b16e-4f5c-90db-494710f4f71c"
  account_id          = var.account_id
  name                = "Health Information"
  type                = "predefined"
  allowed_match_count = 0

  dynamic "entry" {
    for_each = var.entries
    iterator = e
    content {
      id      = e.value.id
      name    = e.key
      enabled = e.value.enabled
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "health" {
  account_id          = var.account_id
  allowed_match_count = 0

  enabled_entries = [for e in [for key, value in (var.entries) : { key = key, value = value }] : (e.value.id) if (e.value.enabled)]
  profile_id      = "583a2366-b16e-4f5c-90db-494710f4f71c"
}

moved {
  from = cloudflare_zero_trust_dlp_profile.health
  to   = cloudflare_zero_trust_dlp_predefined_profile.health
}`,
			},
			{
				Name: "Predefined profile - ignore_changes on entry defers to the live profile (BUGS-2156)",
				Input: `resource "cloudflare_dlp_profile" "health" {
  id                  = "583a2366-b16e-4f5c-90db-494710f4f71c"
  account_id          = var.account_id
  name                = "Health Information"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "9237e83f-fde6-4ff5-84de-0493d7c354d6"
    name    = "FDA Active Ingredients"
    enabled = false
  }

  lifecycle {
    ignore_changes = [entry]
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "health" {
  account_id          = var.account_id
  allowed_match_count = 0
  profile_id          = "583a2366-b16e-4f5c-90db-494710f4f71c"

  lifecycle {
    ignore_changes = [enabled_entries]
  }
}

moved {
  from = cloudflare_dlp_profile.health
  to   = cloudflare_zero_trust_dlp_predefined_profile.health
}`,
			},

			{
				Name: "Predefined profile - literal and dynamic entries are concatenated, quoted bools are literals",
				Input: `resource "cloudflare_dlp_profile" "mixed" {
  id                  = "p1"
  account_id          = var.account_id
  name                = "Mixed"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "literal-on"
    name    = "On"
    enabled = "true"
  }

  entry {
    id      = "literal-off"
    name    = "Off"
    enabled = "false"
  }

  dynamic "entry" {
    for_each = var.entries
    content {
      id      = entry.value
      name    = entry.key
      enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "mixed" {
  account_id          = var.account_id
  allowed_match_count = 0
  enabled_entries     = concat(["literal-on"], [for entry in [for key, value in (var.entries) : { key = key, value = value }] : (entry.value) if (true)])
  profile_id          = "p1"
}

moved {
  from = cloudflare_dlp_profile.mixed
  to   = cloudflare_zero_trust_dlp_predefined_profile.mixed
}`,
			},
			{
				Name: "Predefined profile - unconvertible dynamic entry keeps the entry blocks and writes no enabled_entries",
				Input: `resource "cloudflare_dlp_profile" "unconvertible" {
  id                  = "p1"
  account_id          = var.account_id
  name                = "Unconvertible"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "literal-on"
    name    = "On"
    enabled = true
  }

  dynamic "entry" {
    for_each = var.entries
    content {
      name    = entry.key
      enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "unconvertible" {
  account_id          = var.account_id
  allowed_match_count = 0
  profile_id          = "p1"

  entry {
    id      = "literal-on"
    name    = "On"
    enabled = true
  }

  dynamic "entry" {
    for_each = var.entries
    content {
      name    = entry.key
      enabled = true
    }
  }
}

moved {
  from = cloudflare_dlp_profile.unconvertible
  to   = cloudflare_zero_trust_dlp_predefined_profile.unconvertible
}`,
			},
			{
				Name: "Predefined profile - ignore_changes = all defers to the live profile",
				Input: `resource "cloudflare_dlp_profile" "all" {
  id                  = "p1"
  account_id          = var.account_id
  name                = "All"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "literal-on"
    name    = "On"
    enabled = true
  }

  lifecycle {
    ignore_changes = all
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "all" {
  account_id          = var.account_id
  allowed_match_count = 0
  profile_id          = "p1"

  lifecycle {
    ignore_changes = all
  }
}

moved {
  from = cloudflare_dlp_profile.all
  to   = cloudflare_zero_trust_dlp_predefined_profile.all
}`,
			},
			{
				Name: "Predefined profile - ignore_changes traversal into entry widens to enabled_entries",
				Input: `resource "cloudflare_dlp_profile" "traversal" {
  id                  = "p1"
  account_id          = var.account_id
  name                = "Traversal"
  type                = "predefined"
  allowed_match_count = 0

  entry {
    id      = "literal-on"
    name    = "On"
    enabled = true
  }

  lifecycle {
    ignore_changes = [entry[0].enabled, allowed_match_count, entry]
  }
}`,
				Expected: `resource "cloudflare_zero_trust_dlp_predefined_profile" "traversal" {
  account_id          = var.account_id
  allowed_match_count = 0
  profile_id          = "p1"

  lifecycle {
    ignore_changes = [enabled_entries, allowed_match_count]
  }
}

moved {
  from = cloudflare_dlp_profile.traversal
  to   = cloudflare_zero_trust_dlp_predefined_profile.traversal
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}

func TestPreprocessing(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	// Test that preprocessing does nothing (we handle everything through HCL CFGFile)
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "Preprocessing returns input unchanged",
			input: `resource "cloudflare_dlp_profile" "test" {
  account_id = "123"
  name = "Test"
  type = "custom"
  allowed_match_count = 5

  entry {
    id = "e1"
    name = "Entry 1"
    enabled = true
    pattern {
      regex = "test"
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.Preprocess(tt.input)
			if result != tt.input {
				t.Errorf("Preprocessing should return input unchanged, but got:\n%s", result)
			}
		})
	}
}
