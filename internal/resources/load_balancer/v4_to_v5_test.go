package load_balancer

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Load balancer with TTL",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]
  fallback_pool_id = "example-fallback-pool-id"
  ttl              = 30
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "example-zone-id"
  name          = "example-lb.example.com"
  ttl           = 30
  default_pools = ["example-pool-id"]
  fallback_pool = "example-fallback-pool-id"
}`,
			},
			{
				Name: "Load balancer with steering policy",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id           = "example-zone-id"
  name              = "example-lb.example.com"
  default_pool_ids  = ["example-pool-id"]
  fallback_pool_id  = "example-fallback-pool-id"
  steering_policy   = "geo"
  session_affinity  = "cookie"
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  steering_policy  = "geo"
  session_affinity = "cookie"
  default_pools    = ["example-pool-id"]
  fallback_pool    = "example-fallback-pool-id"
}`,
			},
			{
				Name: "Load balancer with adaptive_routing block",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  adaptive_routing {
    failover_across_pools = false
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  adaptive_routing = {
    failover_across_pools = false
  }
}`,
			},
			{
				Name: "Load balancer with location_strategy block",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  location_strategy {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
}`,
			},
			{
				Name: "Load balancer with random_steering block",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  random_steering {
    default_weight = 0.5
    pool_weights = {
      pool1 = 0.3
      pool2 = 0.7
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  random_steering = {
    default_weight = 0.5
    pool_weights = {
      pool1 = 0.3
      pool2 = 0.7
    }
  }
}`,
			},
			{
				Name: "Load balancer with all optional single-object blocks",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]

  adaptive_routing {
    failover_across_pools = false
  }

  location_strategy {
    prefer_ecs = "proximity"
    mode       = "pop"
  }

  random_steering {
    default_weight = 0.5
  }

  session_affinity_attributes {
    samesite = "Lax"
    secure   = "Always"
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id = "example-zone-id"
  name    = "example-lb.example.com"

  default_pools = ["example-pool-id"]
  session_affinity_attributes = {
    samesite = "Lax"
    secure   = "Always"
  }
  adaptive_routing = {
    failover_across_pools = false
  }
  location_strategy = {
    prefer_ecs = "proximity"
    mode       = "pop"
  }
  random_steering = {
    default_weight = 0.5
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("RulesTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Rules with fixed_response only",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "zone-id"
  name             = "example.com"
  default_pool_ids = ["pool-id"]
  fallback_pool_id = "fallback-id"

  rules {
    name      = "return 200"
    condition = "dns.qry.type == 28"

    fixed_response {
      message_body = "hello"
      status_code  = 200
      content_type = "html"
      location     = "www.example.com"
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "zone-id"
  name          = "example.com"
  default_pools = ["pool-id"]
  fallback_pool = "fallback-id"
  rules = [
    {
      name      = "return 200"
      condition = "dns.qry.type == 28"
      fixed_response = {
        message_body = "hello"
        status_code  = 200
        content_type = "html"
        location     = "www.example.com"
      }
    }
  ]
}`,
			},
			{
				Name: "Rules with overrides containing session_affinity_attributes and adaptive_routing",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "zone-id"
  name             = "example.com"
  default_pool_ids = ["pool-id"]
  fallback_pool_id = "fallback-id"

  rules {
    name      = "override rule"
    condition = "dns.qry.type == 28"

    overrides {
      steering_policy = "geo"

      session_affinity_attributes {
        samesite               = "Auto"
        secure                 = "Auto"
        zero_downtime_failover = "sticky"
      }

      adaptive_routing {
        failover_across_pools = true
      }
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "zone-id"
  name          = "example.com"
  default_pools = ["pool-id"]
  fallback_pool = "fallback-id"
  rules = [
    {
      name      = "override rule"
      condition = "dns.qry.type == 28"
      overrides = {
        steering_policy = "geo"
        session_affinity_attributes = {
          samesite               = "Auto"
          secure                 = "Auto"
          zero_downtime_failover = "sticky"
        }
        adaptive_routing = {
          failover_across_pools = true
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Rules with overrides containing location_strategy and random_steering",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "zone-id"
  name             = "example.com"
  default_pool_ids = ["pool-id"]
  fallback_pool_id = "fallback-id"

  rules {
    name      = "steering rule"
    condition = "dns.qry.type == 28"

    overrides {
      location_strategy {
        prefer_ecs = "always"
        mode       = "resolver_ip"
      }

      random_steering {
        default_weight = 0.2
        pool_weights = {
          "pool-id" = 0.4
        }
      }
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "zone-id"
  name          = "example.com"
  default_pools = ["pool-id"]
  fallback_pool = "fallback-id"
  rules = [
    {
      name      = "steering rule"
      condition = "dns.qry.type == 28"
      overrides = {
        location_strategy = {
          prefer_ecs = "always"
          mode       = "resolver_ip"
        }
        random_steering = {
          default_weight = 0.2
          pool_weights = {
            "pool-id" = 0.4
          }
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Rules with overrides containing region_pools",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "zone-id"
  name             = "example.com"
  default_pool_ids = ["pool-id"]
  fallback_pool_id = "fallback-id"

  rules {
    name      = "region rule"
    condition = "dns.qry.type == 28"

    overrides {
      region_pools {
        region   = "ENAM"
        pool_ids = ["pool-id"]
      }

      region_pools {
        region   = "WNAM"
        pool_ids = ["pool-id-2"]
      }
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "zone-id"
  name          = "example.com"
  default_pools = ["pool-id"]
  fallback_pool = "fallback-id"
  rules = [
    {
      name      = "region rule"
      condition = "dns.qry.type == 28"
      overrides = {
        region_pools = {
          "ENAM" = ["pool-id"]
          "WNAM" = ["pool-id-2"]
        }
      }
    }
  ]
}`,
			},
			{
				Name: "Multiple rules with mixed overrides and fixed_response",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "zone-id"
  name             = "example.com"
  default_pool_ids = ["pool-id"]
  fallback_pool_id = "fallback-id"

  rules {
    name      = "rule 1"
    condition = "dns.qry.type == 28"

    overrides {
      steering_policy = "geo"

      adaptive_routing {
        failover_across_pools = true
      }

      region_pools {
        region   = "ENAM"
        pool_ids = ["pool-id"]
      }
    }
  }

  rules {
    name      = "rule 2"
    condition = "dns.qry.type == 1"

    fixed_response {
      message_body = "not found"
      status_code  = 404
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "zone-id"
  name          = "example.com"
  default_pools = ["pool-id"]
  fallback_pool = "fallback-id"
  rules = [
    {
      name      = "rule 1"
      condition = "dns.qry.type == 28"
      overrides = {
        steering_policy = "geo"
        adaptive_routing = {
          failover_across_pools = true
        }
        region_pools = {
          "ENAM" = ["pool-id"]
        }
      }
    },
    {
      name      = "rule 2"
      condition = "dns.qry.type == 1"
      fixed_response = {
        message_body = "not found"
        status_code  = 404
      }
    }
  ]
}`,
			},
			{
				Name: "Rules with all overrides sub-blocks (comprehensive)",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "zone-id"
  name             = "example.com"
  default_pool_ids = ["pool-id"]
  fallback_pool_id = "fallback-id"

  rules {
    name      = "comprehensive rule"
    condition = "dns.qry.type == 28"
    priority  = 1
    disabled  = false

    overrides {
      steering_policy      = "geo"
      session_affinity     = "cookie"
      session_affinity_ttl = 3600
      fallback_pool        = "fallback-override-id"
      default_pools        = ["pool-override-id"]

      session_affinity_attributes {
        samesite = "Lax"
        secure   = "Always"
      }

      adaptive_routing {
        failover_across_pools = true
      }

      location_strategy {
        prefer_ecs = "always"
        mode       = "resolver_ip"
      }

      random_steering {
        default_weight = 0.3
      }

      region_pools {
        region   = "WNAM"
        pool_ids = ["pool-id"]
      }

      pop_pools {
        pop      = "LAX"
        pool_ids = ["pool-id"]
      }

      country_pools {
        country  = "US"
        pool_ids = ["pool-id"]
      }
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "zone-id"
  name          = "example.com"
  default_pools = ["pool-id"]
  fallback_pool = "fallback-id"
  rules = [
    {
      name      = "comprehensive rule"
      condition = "dns.qry.type == 28"
      priority  = 1
      disabled  = false
      overrides = {
        steering_policy      = "geo"
        session_affinity     = "cookie"
        session_affinity_ttl = 3600
        fallback_pool        = "fallback-override-id"
        default_pools        = ["pool-override-id"]
        session_affinity_attributes = {
          samesite = "Lax"
          secure   = "Always"
        }
        adaptive_routing = {
          failover_across_pools = true
        }
        location_strategy = {
          prefer_ecs = "always"
          mode       = "resolver_ip"
        }
        random_steering = {
          default_weight = 0.3
        }
        region_pools = {
          "WNAM" = ["pool-id"]
        }
        pop_pools = {
          "LAX" = ["pool-id"]
        }
        country_pools = {
          "US" = ["pool-id"]
        }
      }
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// BuildDynamicForListTokens emits spacing inside its tokens, which the
	// semantic comparison of RunConfigTransformTests doesn't normalise, so
	// these compare the formatted output.
	t.Run("DynamicRules", func(t *testing.T) {
		for _, tt := range []testhelpers.ConfigTestCase{
			{
				Name: "dynamic rules with nested overrides",
				Input: `resource "cloudflare_load_balancer" "lb" {
  zone_id          = "0123456789abcdef0123456789abcdef"
  name             = "lb.example.com"
  fallback_pool_id = "p"
  default_pool_ids = ["p"]
  dynamic "rules" {
    for_each = var.pools
    content {
      name      = rules.key
      condition = "http.request.uri.path eq \"/${rules.key}\""
      overrides {
        steering_policy = "off"
      }
    }
  }
}
`,
				Expected: `resource "cloudflare_load_balancer" "lb" {
  zone_id       = "0123456789abcdef0123456789abcdef"
  name          = "lb.example.com"
  default_pools = ["p"]
  fallback_pool = "p"
  rules = [for k, v in var.pools : {
    name      = k
    condition = "http.request.uri.path eq \"/${k}\""
    overrides = {
      steering_policy = "off"
    }
  }]
}
`,
			},
			{
				Name: "static and dynamic rules keep source order",
				Input: `resource "cloudflare_load_balancer" "mixed" {
  zone_id          = "0123456789abcdef0123456789abcdef"
  name             = "lb2.example.com"
  fallback_pool_id = "p"
  default_pool_ids = ["p"]
  rules {
    name      = "first"
    condition = "true"
    fixed_response {
      status_code = 200
    }
  }
  dynamic "rules" {
    for_each = var.pools
    iterator = r
    content {
      name     = r.key
      priority = r.value
      overrides {
        region_pools {
          region   = "WNAM"
          pool_ids = ["p"]
        }
      }
    }
  }
  rules {
    name      = "last"
    condition = "true"
    disabled  = true
  }
}
`,
				Expected: `resource "cloudflare_load_balancer" "mixed" {
  zone_id       = "0123456789abcdef0123456789abcdef"
  name          = "lb2.example.com"
  default_pools = ["p"]
  fallback_pool = "p"
  rules = concat(
    [{
      name      = "first"
      condition = "true"
      fixed_response = {
        status_code = 200
      }
    }],
    [for k, v in var.pools : {
      name     = k
      priority = v
      overrides = {
        region_pools = {
          "WNAM" = ["p"]
        }
      }
    }],
    [{
      name      = "last"
      condition = "true"
      disabled  = true
    }],
  )
}
`,
			},
			{
				Name: "Basic load balancer",
				Input: `resource "cloudflare_load_balancer" "example" {
  zone_id          = "example-zone-id"
  name             = "example-lb.example.com"
  default_pool_ids = ["example-pool-id"]
  fallback_pool_id = "example-fallback-pool-id"
}`,
				Expected: `resource "cloudflare_load_balancer" "example" {
  zone_id       = "example-zone-id"
  name          = "example-lb.example.com"
  default_pools = ["example-pool-id"]
  fallback_pool = "example-fallback-pool-id"
}`,
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				file, diags := hclwrite.ParseConfig([]byte(tt.Input), "test.tf", hcl.InitialPos)
				require.False(t, diags.HasErrors())
				ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
				_, err := migrator.TransformConfig(ctx, file.Body().Blocks()[0])
				require.NoError(t, err)
				assert.Empty(t, ctx.Diagnostics)
				assert.Equal(t, string(hclwrite.Format([]byte(tt.Expected))), string(hclwrite.Format(file.Bytes())))
			})
		}
	})

	t.Run("NestedDynamicInRuleWarns", func(t *testing.T) {
		input := `resource "cloudflare_load_balancer" "lb" {
  zone_id          = "z"
  name             = "lb.example.com"
  fallback_pool_id = "p"
  default_pool_ids = ["p"]
  dynamic "rules" {
    for_each = var.rules
    content {
      name = rules.key
      dynamic "overrides" {
        for_each = [1]
        content {
          steering_policy = "off"
        }
      }
    }
  }
}
`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		require.False(t, diags.HasErrors())
		ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
		_, err := migrator.TransformConfig(ctx, file.Body().Blocks()[0])
		require.NoError(t, err)
		require.Len(t, ctx.Diagnostics, 1)
		assert.Contains(t, ctx.Diagnostics[0].Summary, "cloudflare_load_balancer.lb")
		assert.Contains(t, ctx.Diagnostics[0].Detail, "nested dynamic block")
		// Left in place, unchanged.
		assert.Contains(t, string(file.Bytes()), `dynamic "rules" {`)
		assert.Contains(t, string(file.Bytes()), `dynamic "overrides" {`)
	})
}
