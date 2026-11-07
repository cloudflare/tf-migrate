package ruleset

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each expected output was checked with terraform validate and plan against
// provider v5.28.1.
func TestDynamicBlocks(t *testing.T) {
	tests := []testhelpers.ConfigTestCase{
		{
			// https://github.com/cloudflare/tf-migrate/issues/304
			Name: "nested dynamic ratelimit inside dynamic rules",
			Input: `resource "cloudflare_ruleset" "http_ratelimit" {
  account_id = var.account_id
  name       = "Rate limit configuration"
  kind       = "custom"
  phase      = "http_ratelimit"

  dynamic "rules" {
    for_each = local.account_ratelimit_rules_with_ref
    content {
      action     = try(rules.value.action, null)
      expression = try(rules.value.expression, null)
      ref        = rules.value.ref

      dynamic "ratelimit" {
        for_each = [rules.value.ratelimit]
        content {
          characteristics     = try(ratelimit.value.characteristics, null)
          period              = try(ratelimit.value.period, null)
          requests_per_period = try(ratelimit.value.requests_per_period, null)
        }
      }
    }
  }
}`,
			Expected: `resource "cloudflare_ruleset" "http_ratelimit" {
  account_id = var.account_id
  name       = "Rate limit configuration"
  kind       = "custom"
  phase      = "http_ratelimit"

  rules = [for rules in local.account_ratelimit_rules_with_ref : {
    action     = try(rules.action, null)
    expression = try(rules.expression, null)
    ref        = rules.ref
    ratelimit = one([for ratelimit in [rules.ratelimit] : {
      characteristics     = try(ratelimit.characteristics, null)
      period              = try(ratelimit.period, null)
      requests_per_period = try(ratelimit.requests_per_period, null)
    }])
  }]
}`,
		},
		{
			Name: "dynamic ratelimit inside a static rule",
			Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_ratelimit"

  rules {
    action     = "block"
    expression = "true"
    dynamic "ratelimit" {
      for_each = var.enable_ratelimit ? [1] : []
      content {
        characteristics     = ["ip.src", "cf.colo.id"]
        period              = 60
        requests_per_period = 100
      }
    }
  }
}`,
			Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_ratelimit"

  rules = [
    {
      action     = "block"
      expression = "true"
      ratelimit = one([for ratelimit in var.enable_ratelimit ? [1] : [] : {
        characteristics     = ["ip.src", "cf.colo.id"]
        period              = 60
        requests_per_period = 100
      }])
    }
  ]
}`,
		},
		{
			Name: "dynamic list block nested in action_parameters stays a list",
			Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules {
    action     = "execute"
    expression = "true"
    action_parameters {
      id = "efb7b8c949ac4650a09736fc376e9aee"
      overrides {
        dynamic "categories" {
          for_each = var.disabled_categories
          content {
            category = categories.value
            enabled  = false
          }
        }
      }
    }
  }
}`,
			Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules = [
    {
      action     = "execute"
      expression = "true"
      action_parameters = {
        id = "efb7b8c949ac4650a09736fc376e9aee"
        overrides = {
          categories = [for categories in var.disabled_categories : {
            category = categories
            enabled  = false
          }]
        }
      }
    }
  ]
}`,
		},
		{
			Name: "dynamic rules content gets the static rule rewrites",
			Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  dynamic "rules" {
    for_each = var.hosts
    content {
      action     = "set_cache_settings"
      expression = rules.value.expression
      action_parameters {
        cache = true
        cache_key {
          custom_key {
            query_string {
              exclude = ["*"]
            }
          }
        }
        headers {
          name      = "X-Frame-Options"
          operation = "set"
          value     = "DENY"
        }
      }
    }
  }
}`,
			Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [for rules in var.hosts : {
    action     = "set_cache_settings"
    expression = rules.expression
    action_parameters = {
      cache = true
      headers = {
        "X-Frame-Options" = {
          operation = "set"
          value     = "DENY"
        }
      }
      cache_key = {
        custom_key = {
          query_string = {
            exclude = {
              all = true
            }
          }
        }
      }
    }
  }]
}`,
		},
		{
			Name: "static and dynamic rules keep every rule in source order",
			Input: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules {
    action     = "block"
    expression = "first"
  }

  dynamic "rules" {
    for_each = var.hosts
    iterator = host
    content {
      action      = "block"
      expression  = "(http.host eq \"${host.value}\")"
      description = host.key
    }
  }

  rules {
    action     = "block"
    expression = "last"
  }
}`,
			Expected: `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = concat(
    [{
      action     = "block"
      expression = "first"
    }],
    [for host_key, host in var.hosts : {
      action      = "block"
      expression  = "(http.host eq \"${host}\")"
      description = host_key
    }],
    [{
      action     = "block"
      expression = "last"
    }],
  )
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, &V4ToV5Migrator{})
}

func TestDynamicBlockWithoutForEachIsLeftWithWarning(t *testing.T) {
	input := `resource "cloudflare_ruleset" "example" {
  zone_id = "test123"
  name    = "test-ruleset"
  kind    = "zone"
  phase   = "http_ratelimit"

  rules {
    action     = "block"
    expression = "true"
    dynamic "ratelimit" {
      content {
        period = 60
      }
    }
  }
}`
	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())
	ctx := &transform.Context{CFGFile: file}

	_, err := (&V4ToV5Migrator{}).TransformConfig(ctx, file.Body().Blocks()[0])
	require.NoError(t, err)

	require.Len(t, ctx.Diagnostics, 1)
	assert.Equal(t, hcl.DiagWarning, ctx.Diagnostics[0].Severity)
	assert.Contains(t, ctx.Diagnostics[0].Summary, "Dynamic ratelimit block not migrated on cloudflare_ruleset.example")
	assert.Contains(t, ctx.Diagnostics[0].Detail, "has no for_each")
}
