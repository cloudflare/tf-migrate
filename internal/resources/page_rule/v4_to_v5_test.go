package page_rule

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions {
    cache_level = "bypass"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  status  = "active"
  actions = {
    cache_level = "bypass"
  }
}`,
			},
			{
				Name: "With forwarding_url nested block",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/old/*"
  actions {
    forwarding_url {
      url         = "https://example.com/new/"
      status_code = 301
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/old/*"
  status  = "active"
  actions = {
    forwarding_url = {
      url         = "https://example.com/new/"
      status_code = 301
    }
  }
}`,
			},
			{
				Name: "With cache_key_fields deeply nested",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions {
    cache_key_fields {
      cookie {
        check_presence = ["sessionid"]
      }
      host {
        resolved = true
      }
      user {
        device_type = true
        geo         = false
      }
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  status  = "active"
  actions = {
    cache_key_fields = {
      cookie = {
        check_presence = ["sessionid"]
      }
      host = {
        resolved = true
      }
      user = {
        device_type = true
        geo         = false
        lang        = false
      }
    }
  }
}`,
			},
			{
				Name: "With cache_ttl_by_status blocks to map",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions {
    cache_ttl_by_status {
      codes = "200"
      ttl   = 3600
    }
    cache_ttl_by_status {
      codes = "404"
      ttl   = 300
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  status  = "active"
  actions = {
    cache_ttl_by_status = {
      "200" = "3600"
      "404" = "300"
    }
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
