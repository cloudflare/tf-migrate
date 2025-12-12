package bot_management

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic resource with common fields",
				Input: `
resource "cloudflare_bot_management" "example" {
  zone_id                        = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js                      = true
  fight_mode                     = false
  auto_update_model              = true
  suppress_session_score         = false
}`,
				Expected: `resource "cloudflare_bot_management" "example" {
  zone_id                = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js              = true
  fight_mode             = false
  auto_update_model      = true
  suppress_session_score = false
}`,
			},
			{
				Name: "Resource with SBFM fields",
				Input: `
resource "cloudflare_bot_management" "sbfm" {
  zone_id                        = "0da42c8d2132a9ddaf714f9e7c920711"
  sbfm_definitely_automated      = "block"
  sbfm_likely_automated          = "managed_challenge"
  sbfm_verified_bots             = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress             = true
}`,
				Expected: `resource "cloudflare_bot_management" "sbfm" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = true
}`,
			},
			{
				Name: "Resource with AI bots protection",
				Input: `
resource "cloudflare_bot_management" "ai_protection" {
  zone_id            = "0da42c8d2132a9ddaf714f9e7c920711"
  ai_bots_protection = "block"
}`,
				Expected: `resource "cloudflare_bot_management" "ai_protection" {
  zone_id            = "0da42c8d2132a9ddaf714f9e7c920711"
  ai_bots_protection = "block"
}`,
			},
			{
				Name: "Resource with all v4 fields",
				Input: `
resource "cloudflare_bot_management" "complete" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js                       = true
  fight_mode                      = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "block"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"
}`,
				Expected: `resource "cloudflare_bot_management" "complete" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js                       = true
  fight_mode                      = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "block"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"
}`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `
resource "cloudflare_bot_management" "first" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js  = true
  fight_mode = false
}

resource "cloudflare_bot_management" "second" {
  zone_id            = "28fea702d1075b10ba9c8620b86218ec"
  ai_bots_protection = "block"
}`,
				Expected: `resource "cloudflare_bot_management" "first" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js  = true
  fight_mode = false
}

resource "cloudflare_bot_management" "second" {
  zone_id            = "28fea702d1075b10ba9c8620b86218ec"
  ai_bots_protection = "block"
}`,
			},
			{
				Name: "Resource with computed field in config (using_latest_model)",
				Input: `
resource "cloudflare_bot_management" "with_computed" {
  zone_id           = "0da42c8d2132a9ddaf714f9e7c920711"
  auto_update_model = true
}`,
				Expected: `resource "cloudflare_bot_management" "with_computed" {
  zone_id           = "0da42c8d2132a9ddaf714f9e7c920711"
  auto_update_model = true
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Full state with basic fields",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "enable_js": true,
        "fight_mode": false,
        "auto_update_model": true,
        "suppress_session_score": false
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "enable_js": true,
        "fight_mode": false,
        "auto_update_model": true,
        "suppress_session_score": false
      }
    }]
  }]
}`,
			},
			{
				Name: "Full state with all v4 fields",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "complete",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "enable_js": true,
        "fight_mode": true,
        "sbfm_definitely_automated": "block",
        "sbfm_likely_automated": "block",
        "sbfm_verified_bots": "allow",
        "sbfm_static_resource_protection": true,
        "optimize_wordpress": false,
        "suppress_session_score": false,
        "auto_update_model": true,
        "using_latest_model": true,
        "ai_bots_protection": "disabled"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "complete",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "enable_js": true,
        "fight_mode": true,
        "sbfm_definitely_automated": "block",
        "sbfm_likely_automated": "block",
        "sbfm_verified_bots": "allow",
        "sbfm_static_resource_protection": true,
        "optimize_wordpress": false,
        "suppress_session_score": false,
        "auto_update_model": true,
        "using_latest_model": true,
        "ai_bots_protection": "disabled"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with computed field - using_latest_model",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "auto_update_model": true,
        "using_latest_model": false
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "auto_update_model": true,
        "using_latest_model": false
      }
    }]
  }]
}`,
			},
			{
				Name: "State with AI bots protection values",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "ai",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "ai_bots_protection": "block"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "ai",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "ai_bots_protection": "block"
      }
    }]
  }]
}`,
			},
			{
				Name: "Minimal state with only required fields",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "minimal",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "minimal",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711"
      }
    }]
  }]
}`,
			},
			{
				Name: "Multiple instances of the same resource",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "multi",
    "instances": [
      {
        "attributes": {
          "id": "0da42c8d2132a9ddaf714f9e7c920711",
          "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
          "enable_js": true,
          "fight_mode": false
        }
      },
      {
        "attributes": {
          "id": "28fea702d1075b10ba9c8620b86218ec",
          "zone_id": "28fea702d1075b10ba9c8620b86218ec",
          "enable_js": false,
          "fight_mode": true
        }
      }
    ]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "multi",
    "instances": [
      {
        "attributes": {
          "id": "0da42c8d2132a9ddaf714f9e7c920711",
          "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
          "enable_js": true,
          "fight_mode": false
        }
      },
      {
        "attributes": {
          "id": "28fea702d1075b10ba9c8620b86218ec",
          "zone_id": "28fea702d1075b10ba9c8620b86218ec",
          "enable_js": false,
          "fight_mode": true
        }
      }
    ]
  }]
}`,
			},
			{
				Name: "State with empty resources array",
				Input: `{
  "resources": []
}`,
				Expected: `{
  "resources": []
}`,
			},
			{
				Name: "State without instances",
				Input: `{
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "empty",
    "instances": []
  }]
}`,
				Expected: `{
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "empty",
    "instances": []
  }]
}`,
			},
			{
				Name: "Single instance state (for internal testing)",
				Input: `{
  "attributes": {
    "id": "0da42c8d2132a9ddaf714f9e7c920711",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "enable_js": true,
    "fight_mode": false,
    "auto_update_model": true
  }
}`,
				Expected: `{
  "attributes": {
    "id": "0da42c8d2132a9ddaf714f9e7c920711",
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "enable_js": true,
    "fight_mode": false,
    "auto_update_model": true
  }
}`,
			},
			{
				Name: "State with SBFM configuration",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "sbfm",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "sbfm_definitely_automated": "managed_challenge",
        "sbfm_likely_automated": "managed_challenge",
        "sbfm_verified_bots": "allow",
        "sbfm_static_resource_protection": false,
        "optimize_wordpress": true
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_bot_management",
    "name": "sbfm",
    "instances": [{
      "attributes": {
        "id": "0da42c8d2132a9ddaf714f9e7c920711",
        "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
        "sbfm_definitely_automated": "managed_challenge",
        "sbfm_likely_automated": "managed_challenge",
        "sbfm_verified_bots": "allow",
        "sbfm_static_resource_protection": false,
        "optimize_wordpress": true
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
