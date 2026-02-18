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
				Name: "Predefined profile migration",
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
  name                = "AWS Keys"
  allowed_match_count = 3

  enabled_entries = ["aws-access-key", "aws-secret-key"]
  profile_id      = "aws-keys-uuid"
}

moved {
  from = cloudflare_dlp_profile.predefined
  to   = cloudflare_zero_trust_dlp_predefined_profile.predefined
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation_Removed", func(t *testing.T) {
		t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
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
