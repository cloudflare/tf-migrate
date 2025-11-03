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
				Name: "remove minify block from actions",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    minify = {
      html = "on"
      css  = "on"
      js   = "on"
    }
    ssl = "flexible"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    ssl         = "flexible"
  }
}`,
			},
			{
				Name: "remove minify with different formatting",
				Input: `resource "cloudflare_page_rule" "example" {
  actions = {
    minify = { html = "on", css = "on", js = "on" }
    cache_level = "aggressive"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  actions = {
    cache_level = "aggressive"
  }
}`,
			},
			{
				Name: "handle resource without minify",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    ssl         = "flexible"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    ssl         = "flexible"
  }
}`,
			},
			{
				Name: "consolidate multiple cache_ttl_by_status entries",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    cache_ttl_by_status = {
      codes = "200"
      ttl   = 3600
    }
    cache_ttl_by_status = {
      codes = "301"
      ttl   = 1800
    }
    cache_ttl_by_status = {
      codes = "404"
      ttl   = 300
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level         = "aggressive"
    cache_ttl_by_status = { "200" = 3600, "301" = 1800, "404" = 300 }
  }
}`,
			},
			{
				Name: "single cache_ttl_by_status",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    cache_ttl_by_status = {
      codes = "200"
      ttl   = 3600
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level         = "aggressive"
    cache_ttl_by_status = { "200" = 3600 }
  }
}`,
			},
			{
				Name: "no cache_ttl_by_status",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    ssl         = "flexible"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    cache_level = "aggressive"
    ssl         = "flexible"
  }
}`,
			},
			{
				Name: "complete transformation with minify and cache_ttl_by_status",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id  = "abc123"
  target   = "example.com/*"
  priority = 1
  actions = {
    cache_level = "aggressive"
    minify = {
      html = "on"
      css  = "on"
      js   = "on"
    }
    cache_ttl_by_status = {
      codes = "200"
      ttl   = 3600
    }
    cache_ttl_by_status = {
      codes = "301"
      ttl   = 1800
    }
    ssl = "flexible"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id  = "abc123"
  target   = "example.com/*"
  priority = 1
  actions = {
    cache_level         = "aggressive"
    cache_ttl_by_status = { "200" = 3600, "301" = 1800 }
    ssl                 = "flexible"
  }
}`,
			},
			{
				Name: "page rule with no transformable content",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    ssl                = "flexible"
    browser_check      = "on"
    email_obfuscation  = "on"
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    ssl               = "flexible"
    browser_check     = "on"
    email_obfuscation = "on"
  }
}`,
			},
			{
				Name: "page rule with complex nested structure",
				Input: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    forwarding_url = {
      url         = "https://www.example.com/$1"
      status_code = 301
    }
    minify = {
      html = "on"
      css  = "on"
      js   = "on"
    }
  }
}`,
				Expected: `resource "cloudflare_page_rule" "example" {
  zone_id = "abc123"
  target  = "example.com/*"
  actions = {
    forwarding_url = {
      url         = "https://www.example.com/$1"
      status_code = 301
    }
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
