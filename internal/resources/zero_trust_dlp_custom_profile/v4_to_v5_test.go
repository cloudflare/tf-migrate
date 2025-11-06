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
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic state with single entry",
				Input: `{
  "type": "cloudflare_dlp_profile",
  "name": "example",
  "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
  "instances": [{
    "attributes": {
      "account_id": "123456789",
      "name": "Credit Card Profile",
      "description": "Profile for detecting credit cards",
      "type": "custom",
      "allowed_match_count": 5,
      "entry": [
        {
          "id": "abc123",
          "name": "Visa Card",
          "enabled": true,
          "pattern": [
            {
              "regex": "4[0-9]{12}(?:[0-9]{3})?",
              "validation": "luhn"
            }
          ]
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_custom_profile",
  "name": "example",
  "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
  "instances": [{
    "attributes": {
      "account_id": "123456789",
      "name": "Credit Card Profile",
      "description": "Profile for detecting credit cards",
      "allowed_match_count": 5,
      "entries": [
        {
          "name": "Visa Card",
          "enabled": true,
          "pattern": {
            "regex": "4[0-9]{12}(?:[0-9]{3})?",
            "validation": "luhn"
          }
        }
      ]
    },
    "schema_version": 0
  }]
}`,
			},
			{
				Name: "Multiple entries state",
				Input: `{
  "type": "cloudflare_dlp_profile",
  "name": "multi",
  "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Multi",
      "type": "custom",
      "allowed_match_count": 10,
      "entry": [
        {
          "id": "e1",
          "name": "Entry 1",
          "enabled": true,
          "pattern": [{"regex": "test1"}]
        },
        {
          "name": "Entry 2",
          "enabled": false,
          "pattern": [{"regex": "test2", "validation": "luhn"}]
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_custom_profile",
  "name": "multi",
  "provider": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Multi",
      "allowed_match_count": 10,
      "entries": [
        {
          "name": "Entry 1",
          "enabled": true,
          "pattern": {
            "regex": "test1"
          }
        },
        {
          "name": "Entry 2",
          "enabled": false,
          "pattern": {
            "regex": "test2",
            "validation": "luhn"
          }
        }
      ]
    },
    "schema_version": 0
  }]
}`,
			},
			{
				Name: "Minimal state",
				Input: `{
  "type": "cloudflare_dlp_profile",
  "name": "minimal",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Min",
      "type": "custom",
      "allowed_match_count": 0,
      "entry": [
        {
          "name": "Test",
          "enabled": true,
          "pattern": [{"regex": ".*"}]
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_custom_profile",
  "name": "minimal",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Min",
      "allowed_match_count": 0,
      "entries": [
        {
          "name": "Test",
          "enabled": true,
          "pattern": {
            "regex": ".*"
          }
        }
      ]
    },
    "schema_version": 0
  }]
}`,
			},
			{
				Name: "Predefined profile state transformation",
				Input: `{
  "type": "cloudflare_dlp_profile",
  "name": "predefined",
  "instances": [{
    "attributes": {
      "id": "aws-keys-uuid",
      "account_id": "123456789",
      "name": "AWS Keys",
      "type": "predefined",
      "allowed_match_count": 3,
      "entry": [
        {
          "id": "aws-access-key",
          "name": "AWS Access Key ID",
          "enabled": true
        },
        {
          "id": "aws-secret-key",
          "name": "AWS Secret Access Key",
          "enabled": true
        },
        {
          "id": "aws-session-token",
          "name": "AWS Session Token",
          "enabled": false
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_predefined_profile",
  "name": "predefined",
  "instances": [{
    "attributes": {
      "id": "aws-keys-uuid",
      "profile_id": "aws-keys-uuid",
      "account_id": "123456789",
      "name": "AWS Keys",
      "allowed_match_count": 3,
      "enabled_entries": ["aws-access-key", "aws-secret-key"]
    },
    "schema_version": 0
  }]
}`,
			},
			{
				Name: "Zero Trust DLP profile v4 name state",
				Input: `{
  "type": "cloudflare_zero_trust_dlp_profile",
  "name": "zerotrust",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "ZT Test",
      "type": "custom",
      "allowed_match_count": 2,
      "entry": [
        {
          "id": "zt1",
          "name": "ZT Entry",
          "enabled": true,
          "pattern": [{"regex": "zt[0-9]+"}]
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_custom_profile",
  "name": "zerotrust",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "ZT Test",
      "allowed_match_count": 2,
      "entries": [
        {
          "name": "ZT Entry",
          "enabled": true,
          "pattern": {
            "regex": "zt[0-9]+"
          }
        }
      ]
    },
    "schema_version": 0
  }]
}`,
			},
			{
				Name: "State with context_awareness.skip empty array",
				Input: `{
  "type": "cloudflare_dlp_profile",
  "name": "context_skip_test",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Test",
      "type": "custom",
      "allowed_match_count": 1,
      "context_awareness": {
        "skip": []
      },
      "entry": [
        {
          "name": "Test",
          "enabled": true,
          "pattern": [{"regex": "test"}]
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_custom_profile",
  "name": "context_skip_test",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Test",
      "allowed_match_count": 1,
      "entries": [
        {
          "name": "Test",
          "enabled": true,
          "pattern": {
            "regex": "test"
          }
        }
      ]
    },
    "schema_version": 0
  }]
}`,
			},
			{
				Name: "State with context_awareness empty array",
				Input: `{
  "type": "cloudflare_dlp_profile",
  "name": "context_test",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Test",
      "type": "custom",
      "allowed_match_count": 1,
      "context_awareness": [],
      "entry": [
        {
          "name": "Test",
          "enabled": true,
          "pattern": [{"regex": "test"}]
        }
      ]
    }
  }]
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_dlp_custom_profile",
  "name": "context_test",
  "instances": [{
    "attributes": {
      "account_id": "123",
      "name": "Test",
      "allowed_match_count": 1,
      "entries": [
        {
          "name": "Test",
          "enabled": true,
          "pattern": {
            "regex": "test"
          }
        }
      ]
    },
    "schema_version": 0
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

func TestPreprocessing(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	// Test that preprocessing does nothing (we handle everything through HCL AST)
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