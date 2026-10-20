package zero_trust_gateway_policy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestGatewayPolicyPrecedencePreservedWithReviewWarning(t *testing.T) {
	for _, resourceType := range []string{"cloudflare_teams_rule", "cloudflare_zero_trust_gateway_policy"} {
		t.Run(resourceType, func(t *testing.T) {
			for _, precedence := range []string{"10116", "10000", "10116667", "var.precedence", "each.value.precedence", "local.base + 10"} {
				t.Run(precedence, func(t *testing.T) {
					input := fmt.Sprintf(`resource %q "cftftest_policy" {
  account_id = "account"
  name = "Specific allow"
  action = "allow"
  precedence = %s
}`, resourceType, precedence)
					file, diags := hclwrite.ParseConfig([]byte(input), "policy.tf", hcl.InitialPos)
					if diags.HasErrors() {
						t.Fatal(diags)
					}
					block := file.Body().Blocks()[0]
					before := string(block.Body().GetAttribute("precedence").Expr().BuildTokens(nil).Bytes())
					ctx := &transform.Context{CFGFile: file}
					result, err := (&V4ToV5Migrator{}).TransformConfig(ctx, block)
					if err != nil {
						t.Fatal(err)
					}
					after := string(result.Blocks[0].Body().GetAttribute("precedence").Expr().BuildTokens(nil).Bytes())
					if before != after {
						t.Errorf("precedence expression changed: %q -> %q", before, after)
					}
					if len(ctx.Diagnostics) != 1 || ctx.Diagnostics[0].Severity != hcl.DiagWarning {
						t.Fatalf("expected one review warning, got %v", ctx.Diagnostics)
					}
					warning := ctx.Diagnostics[0]
					if warning.Summary != "Action required: review Gateway policy precedence" {
						t.Errorf("unexpected warning summary: %q", warning.Summary)
					}
					for _, guidance := range []string{resourceType + ".cftftest_policy", "duplicate", "live order", "unique precedence", "Do not mix", "not merely state normalization"} {
						if !strings.Contains(warning.Detail, guidance) {
							t.Errorf("warning missing %q: %s", guidance, warning.Detail)
						}
					}
				})
			}
		})
	}
}

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal gateway policy",
				Input: `resource "cloudflare_teams_rule" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Test Policy"
  description = "Block policy"
  precedence  = 100
  action      = "block"
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Test Policy"
  description = "Block policy"
  precedence  = 100
  action      = "block"
}

moved {
  from = cloudflare_teams_rule.example
  to   = cloudflare_zero_trust_gateway_policy.example
}`,
			},
			{
				Name: "Gateway policy with rule_settings",
				Input: `resource "cloudflare_teams_rule" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Block Policy"
  action      = "block"
  precedence  = 200

  rule_settings {
    block_page_enabled = true
    block_page_reason  = "Access denied"
    override_ips       = ["1.1.1.1"]
  }
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Block Policy"
  action      = "block"
  precedence  = 200

  rule_settings = {
    block_page_enabled = true
    override_ips       = ["1.1.1.1"]
    block_reason       = "Access denied"
  }
}

moved {
  from = cloudflare_teams_rule.with_settings
  to   = cloudflare_zero_trust_gateway_policy.with_settings
}`,
			},
			{
				Name: "Gateway policy with nested blocks",
				Input: `resource "cloudflare_teams_rule" "with_nested" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test"
  action     = "allow"

  rule_settings {
    notification_settings {
      enabled = true
      message = "Custom notification"
    }
    l4override {
      ip   = "192.168.1.1"
      port = 8080
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_nested" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test"
  action     = "allow"

  rule_settings = {
    l4override = {
      ip   = "192.168.1.1"
      port = 8080
    }
    notification_settings = {
      enabled = true
      msg     = "Custom notification"
    }
  }
}

moved {
  from = cloudflare_teams_rule.with_nested
  to   = cloudflare_zero_trust_gateway_policy.with_nested
}`,
			},
			{
				Name: "Gateway policy with nested dns_resolvers",
				Input: `resource "cloudflare_teams_rule" "with_dns_resolvers" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "DNS Resolvers Policy"
  action     = "block"
  precedence = 210

  rule_settings {
    dns_resolvers {
      ipv4 {
        route_through_private_network = true
      }
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_dns_resolvers" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "DNS Resolvers Policy"
  action     = "block"
  precedence = 210

  rule_settings = {
    dns_resolvers = {
      ipv4 = [{
        route_through_private_network = true
      }]
    }
  }
}

moved {
  from = cloudflare_teams_rule.with_dns_resolvers
  to   = cloudflare_zero_trust_gateway_policy.with_dns_resolvers
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}

// TestV4ToV5Transformation_AlreadyV5Named tests the scenario from BUGS-2007:
// The user has already run tf-migrate once (or manually renamed resources),
// so the resource type is already "cloudflare_zero_trust_gateway_policy" (v5 name),
// but rule_settings and notification_settings are still in v4 block syntax.
// tf-migrate must still convert the blocks even when the resource name is already v5.
func TestV4ToV5Transformation_AlreadyV5Named(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "v5-named resource with rule_settings still in block syntax",
			Input: `resource "cloudflare_zero_trust_gateway_policy" "terraform_managed_resource_f65deb9d" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block Policy"
  action     = "block"
  precedence = 10

  rule_settings {
    block_page_enabled = true
    block_page_reason  = "Access denied"
  }
}`,
			Expected: `resource "cloudflare_zero_trust_gateway_policy" "terraform_managed_resource_f65deb9d" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block Policy"
  action     = "block"
  precedence = 10

  rule_settings = {
    block_page_enabled = true
    block_reason       = "Access denied"
  }
}`,
		},
		{
			Name: "v5-named resource with nested notification_settings block",
			Input: `resource "cloudflare_zero_trust_gateway_policy" "with_notification" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Notify Policy"
  action     = "block"
  precedence = 20

  rule_settings {
    notification_settings {
      enabled = true
      message = "You have been blocked"
    }
  }
}`,
			Expected: `resource "cloudflare_zero_trust_gateway_policy" "with_notification" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Notify Policy"
  action     = "block"
  precedence = 20

  rule_settings = {
    notification_settings = {
      enabled = true
      msg     = "You have been blocked"
    }
  }
}`,
		},
		{
			Name: "v5-named resource with l4override block",
			Input: `resource "cloudflare_zero_trust_gateway_policy" "l4_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "L4 Override"
  action     = "l4_override"
  precedence = 30

  rule_settings {
    l4override {
      ip   = "10.0.0.1"
      port = 8080
    }
  }
}`,
			Expected: `resource "cloudflare_zero_trust_gateway_policy" "l4_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "L4 Override"
  action     = "l4_override"
  precedence = 30

  rule_settings = {
    l4override = {
      ip   = "10.0.0.1"
      port = 8080
    }
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
