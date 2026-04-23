package zero_trust_list

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
				Name: "Simple IP list with string array items",
				Input: `
resource "cloudflare_teams_list" "ip_list" {
  account_id = "abc123"
  name       = "IP Allowlist"
  type       = "IP"
  items      = ["192.168.1.1", "10.0.0.0/8"]
}`,
				Expected: `resource "cloudflare_zero_trust_list" "ip_list" {
  account_id = "abc123"
  name       = "IP Allowlist"
  type       = "IP"
  items = [{
    description = null
    value       = "192.168.1.1"
    }, {
    description = null
    value       = "10.0.0.0/8"
  }]
}

moved {
  from = cloudflare_teams_list.ip_list
  to   = cloudflare_zero_trust_list.ip_list
}`,
			},
			{
				Name: "Items with description blocks",
				Input: `
resource "cloudflare_teams_list" "ip_list" {
  account_id = "abc123"
  name       = "IP Allowlist"
  type       = "IP"

  items_with_description {
    value       = "172.16.0.0/12"
    description = "Private network range"
  }

  items_with_description {
    value       = "203.0.113.0/24"
    description = "Test network"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_list" "ip_list" {
  account_id = "abc123"
  name       = "IP Allowlist"
  type       = "IP"

  items = [{
    description = "Private network range"
    value       = "172.16.0.0/12"
    }, {
    description = "Test network"
    value       = "203.0.113.0/24"
  }]
}

moved {
  from = cloudflare_teams_list.ip_list
  to   = cloudflare_zero_trust_list.ip_list
}`,
			},
			{
				Name: "Mixed items and items_with_description",
				Input: `
resource "cloudflare_teams_list" "mixed_list" {
  account_id = "abc123"
  name       = "Mixed IP List"
  type       = "IP"
  items      = ["192.168.1.1", "10.0.0.0/8"]

  items_with_description {
    value       = "172.16.0.0/12"
    description = "Private network"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_list" "mixed_list" {
  account_id = "abc123"
  name       = "Mixed IP List"
  type       = "IP"

  items = [{
    description = "Private network"
    value       = "172.16.0.0/12"
    }, {
    description = null
    value       = "192.168.1.1"
    }, {
    description = null
    value       = "10.0.0.0/8"
  }]
}

moved {
  from = cloudflare_teams_list.mixed_list
  to   = cloudflare_zero_trust_list.mixed_list
}`,
			},
			{
				Name: "Domain list",
				Input: `
resource "cloudflare_teams_list" "domains" {
  account_id = "abc123"
  name       = "Allowed Domains"
  type       = "DOMAIN"
  items      = ["example.com", "*.cloudflare.com", "api.github.com"]
}`,
				Expected: `resource "cloudflare_zero_trust_list" "domains" {
  account_id = "abc123"
  name       = "Allowed Domains"
  type       = "DOMAIN"
  items = [{
    description = null
    value       = "example.com"
    }, {
    description = null
    value       = "*.cloudflare.com"
    }, {
    description = null
    value       = "api.github.com"
  }]
}

moved {
  from = cloudflare_teams_list.domains
  to   = cloudflare_zero_trust_list.domains
}`,
			},
			{
				Name: "URL list with items_with_description",
				Input: `
resource "cloudflare_teams_list" "urls" {
  account_id = "abc123"
  name       = "Blocked URLs"
  type       = "URL"

  items_with_description {
    value       = "https://malicious.example.com/path"
    description = "Known phishing"
  }

  items_with_description {
    value       = "http://phishing.site.com"
    description = "Suspicious site"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_list" "urls" {
  account_id = "abc123"
  name       = "Blocked URLs"
  type       = "URL"

  items = [{
    description = "Known phishing"
    value       = "https://malicious.example.com/path"
    }, {
    description = "Suspicious site"
    value       = "http://phishing.site.com"
  }]
}

moved {
  from = cloudflare_teams_list.urls
  to   = cloudflare_zero_trust_list.urls
}`,
			},
			{
				Name: "Email list",
				Input: `
resource "cloudflare_teams_list" "emails" {
  account_id  = "abc123"
  name        = "VIP Emails"
  type        = "EMAIL"
  description = "Important email addresses"
  items       = ["admin@example.com", "security@cloudflare.com"]
}`,
				Expected: `resource "cloudflare_zero_trust_list" "emails" {
  account_id  = "abc123"
  name        = "VIP Emails"
  type        = "EMAIL"
  description = "Important email addresses"
  items = [{
    description = null
    value       = "admin@example.com"
    }, {
    description = null
    value       = "security@cloudflare.com"
  }]
}

moved {
  from = cloudflare_teams_list.emails
  to   = cloudflare_zero_trust_list.emails
}`,
			},
			{
				Name: "Serial number list",
				Input: `
resource "cloudflare_teams_list" "serials" {
  account_id = "abc123"
  name       = "Certificate Serials"
  type       = "SERIAL"
  items      = ["00:11:22:33:44:55:66", "AA:BB:CC:DD:EE:FF"]
}`,
				Expected: `resource "cloudflare_zero_trust_list" "serials" {
  account_id = "abc123"
  name       = "Certificate Serials"
  type       = "SERIAL"
  items = [{
    description = null
    value       = "00:11:22:33:44:55:66"
    }, {
    description = null
    value       = "AA:BB:CC:DD:EE:FF"
  }]
}

moved {
  from = cloudflare_teams_list.serials
  to   = cloudflare_zero_trust_list.serials
}`,
			},
			{
				Name: "Empty items list",
				Input: `
resource "cloudflare_teams_list" "empty" {
  account_id = "abc123"
  name       = "Empty List"
  type       = "IP"
  items      = []
}`,
				Expected: `resource "cloudflare_zero_trust_list" "empty" {
  account_id = "abc123"
  name       = "Empty List"
  type       = "IP"
}

moved {
  from = cloudflare_teams_list.empty
  to   = cloudflare_zero_trust_list.empty
}`,
			},
			{
				Name: "Multiple resources in single file",
				Input: `
resource "cloudflare_teams_list" "list1" {
  account_id = "abc123"
  name       = "List 1"
  type       = "IP"
  items      = ["192.168.1.1"]
}

resource "cloudflare_teams_list" "list2" {
  account_id = "abc123"
  name       = "List 2"
  type       = "DOMAIN"
  items      = ["example.com"]

  items_with_description {
    value       = "test.com"
    description = "Test domain"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_list" "list1" {
  account_id = "abc123"
  name       = "List 1"
  type       = "IP"
  items = [{
    description = null
    value       = "192.168.1.1"
  }]
}

moved {
  from = cloudflare_teams_list.list1
  to   = cloudflare_zero_trust_list.list1
}

resource "cloudflare_zero_trust_list" "list2" {
  account_id = "abc123"
  name       = "List 2"
  type       = "DOMAIN"

  items = [{
    description = "Test domain"
    value       = "test.com"
    }, {
    description = null
    value       = "example.com"
  }]
}

moved {
  from = cloudflare_teams_list.list2
  to   = cloudflare_zero_trust_list.list2
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}

// TestV4ToV5Transformation_ForExpression tests that items containing a Terraform
// for expression (comprehension) are left completely untouched. Attempting to
// parse [for k, v in map : v] as a static element list would split it at the
// comma, producing invalid HCL (value =\n  _ in …).
func TestV4ToV5Transformation_ForExpression(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "for expression with two iteration variables preserved verbatim",
			Input: `
resource "cloudflare_teams_list" "vault_cidrs" {
  account_id  = var.account_id
  name        = "vault_cidrs"
  description = "CIDR list of Vault endpoints"
  type        = "IP"
  items       = [for cidr, _ in local.vault_cidrs : cidr]
}`,
			Expected: `resource "cloudflare_zero_trust_list" "vault_cidrs" {
  account_id  = var.account_id
  name        = "vault_cidrs"
  description = "CIDR list of Vault endpoints"
  type        = "IP"
  items       = [for cidr, _ in local.vault_cidrs : cidr]
}

moved {
  from = cloudflare_teams_list.vault_cidrs
  to   = cloudflare_zero_trust_list.vault_cidrs
}`,
		},
		{
			Name: "for expression with single iteration variable preserved verbatim",
			Input: `
resource "cloudflare_teams_list" "cidrs" {
  account_id = var.account_id
  name       = "cidrs"
  type       = "IP"
  items      = [for cidr in local.cidrs : cidr]
}`,
			Expected: `resource "cloudflare_zero_trust_list" "cidrs" {
  account_id = var.account_id
  name       = "cidrs"
  type       = "IP"
  items      = [for cidr in local.cidrs : cidr]
}

moved {
  from = cloudflare_teams_list.cidrs
  to   = cloudflare_zero_trust_list.cidrs
}`,
		},
		{
			Name: "for expression with object projection preserved verbatim",
			Input: `
resource "cloudflare_teams_list" "tunnels" {
  account_id = var.account_id
  name       = "tunnels"
  type       = "IP"
  items      = [for cidr, _ in merge(local.pdx_cidrs, local.ams_cidrs) : cidr]
}`,
			Expected: `resource "cloudflare_zero_trust_list" "tunnels" {
  account_id = var.account_id
  name       = "tunnels"
  type       = "IP"
  items      = [for cidr, _ in merge(local.pdx_cidrs, local.ams_cidrs) : cidr]
}

moved {
  from = cloudflare_teams_list.tunnels
  to   = cloudflare_zero_trust_list.tunnels
}`,
		},
		{
			Name: "already-v5-named resource with for expression preserved verbatim",
			Input: `
resource "cloudflare_zero_trust_list" "vault_cidrs" {
  account_id  = var.account_id
  name        = "vault_cidrs"
  description = "CIDR list of Vault endpoints"
  type        = "IP"
  items       = [for cidr, _ in local.vault_cidrs : cidr]
}`,
			Expected: `
resource "cloudflare_zero_trust_list" "vault_cidrs" {
  account_id  = var.account_id
  name        = "vault_cidrs"
  description = "CIDR list of Vault endpoints"
  type        = "IP"
  items       = [for cidr, _ in local.vault_cidrs : cidr]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// TestV4ToV5Transformation_ItemsWithDescriptionAsAttribute tests Bug #001:
// items_with_description written as an HCL attribute (not a block) must be migrated.
// These cover the three real-world patterns found in the production config.
func TestV4ToV5Transformation_ItemsWithDescriptionAsAttribute(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			// Real-world Case A: items_with_description = local.<name>
			// The expression is opaque — cannot be evaluated statically.
			// Expected: rename attribute to items verbatim, preserve the expression.
			Name: "items_with_description as opaque local reference is renamed to items",
			Input: `
resource "cloudflare_zero_trust_list" "do_not_inspect_tunnels" {
  account_id             = var.account_id
  name                   = "Do Not Inspect Tunnels"
  description            = "List of tunnels that should not undergo http inspection"
  type                   = "IP"
  items_with_description = local.do_not_inspect_tunnels
}`,
			Expected: `resource "cloudflare_zero_trust_list" "do_not_inspect_tunnels" {
  account_id  = var.account_id
  name        = "Do Not Inspect Tunnels"
  description = "List of tunnels that should not undergo http inspection"
  type        = "IP"
  items       = local.do_not_inspect_tunnels
}`,
		},
		{
			// Real-world Case B: items_with_description = [...] (inline object list)
			// AND a separate items = [...] attribute.
			// Expected: merged into a single items attribute; items_with_description first.
			// Field order mirrors source order (value before description in the iwd attr).
			// Existing items entries are re-parsed so their field order (description, value)
			// is preserved from the original items attribute.
			Name: "items_with_description as inline object list with resource references merged with items",
			Input: `
resource "cloudflare_zero_trust_list" "do_not_inspect_IPs_employees" {
  account_id  = var.account_id
  name        = "IP addresses to never inspect - Cloudflare Employees"
  type        = "IP"
  items_with_description = [
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_staging_tunnel_ipv4.network
      description = "Athens Staging IPv4"
    },
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_tunnel_ipv4.network
      description = "Athens IPv4"
    }
  ]
  items = [{
    description = null
    value       = "8.14.199.1"
    }, {
    description = null
    value       = "8.14.199.2"
  }]
}`,
			Expected: `resource "cloudflare_zero_trust_list" "do_not_inspect_IPs_employees" {
  account_id  = var.account_id
  name        = "IP addresses to never inspect - Cloudflare Employees"
  type        = "IP"
  items = [
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_staging_tunnel_ipv4.network
      description = "Athens Staging IPv4"
    },
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_tunnel_ipv4.network
      description = "Athens IPv4"
    },
    {
      value       = "8.14.199.1"
      description = null
    },
    {
      value       = "8.14.199.2"
      description = null
    }
  ]
}`,
		},
		{
			// Real-world Case C: same as B but on a v4-named resource (cloudflare_teams_list).
			// Must also produce a moved block.
			// Field order: iwd items preserve source order (value, description);
			// items entries preserve source order (description, value → reparsed as value, description).
			Name: "v4-named resource with items_with_description as inline object list gets moved block",
			Input: `
resource "cloudflare_teams_list" "do_not_inspect_IPs_contractors" {
  account_id  = var.account_id
  name        = "IP addresses to never inspect - Contractors"
  type        = "IP"
  items_with_description = [
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_tunnel_ipv4.network
      description = "Athens IPv4"
    }
  ]
  items = [{
    description = null
    value       = "10.0.0.1"
  }]
}`,
			Expected: `resource "cloudflare_zero_trust_list" "do_not_inspect_IPs_contractors" {
  account_id  = var.account_id
  name        = "IP addresses to never inspect - Contractors"
  type        = "IP"
  items = [
    {
      value       = cloudflare_zero_trust_tunnel_cloudflared_route.athens_tunnel_ipv4.network
      description = "Athens IPv4"
    },
    {
      value       = "10.0.0.1"
      description = null
    }
  ]
}
moved {
  from = cloudflare_teams_list.do_not_inspect_IPs_contractors
  to   = cloudflare_zero_trust_list.do_not_inspect_IPs_contractors
}`,
		},
		{
			// items_with_description as inline object list, no separate items attr.
			// Field order in output matches original source order (value before description).
			Name: "items_with_description as inline object list only, static strings",
			Input: `
resource "cloudflare_zero_trust_list" "example" {
  account_id = var.account_id
  name       = "Example"
  type       = "IP"
  items_with_description = [
    {
      value       = "192.168.1.1"
      description = "Gateway"
    },
    {
      value       = "10.0.0.1"
      description = "Internal"
    }
  ]
}`,
			Expected: `resource "cloudflare_zero_trust_list" "example" {
  account_id = var.account_id
  name       = "Example"
  type       = "IP"
  items = [{
    value       = "192.168.1.1"
    description = "Gateway"
    }, {
    value       = "10.0.0.1"
    description = "Internal"
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// TestV4ToV5Transformation_AlreadyV5Named tests the scenario from BUGS-2009:
// The user has already run tf-migrate once (or manually renamed resources),
// so the resource type is already "cloudflare_zero_trust_list" (v5 name),
// but items_with_description blocks are still in v4 block syntax.
// tf-migrate must still convert the blocks even when the resource name is already v5.
func TestV4ToV5Transformation_AlreadyV5Named(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "v5-named resource with simple items array",
			Input: `
resource "cloudflare_zero_trust_list" "already_v5_simple" {
  account_id = "abc123"
  name       = "Already V5 Simple"
  type       = "IP"
  items      = ["10.0.0.1", "10.0.0.2"]
}`,
			Expected: `
resource "cloudflare_zero_trust_list" "already_v5_simple" {
  account_id = "abc123"
  name       = "Already V5 Simple"
  type       = "IP"

  items = [{
    description = null
    value       = "10.0.0.1"
    }, {
    description = null
    value       = "10.0.0.2"
  }]
}`,
		},
		{
			Name: "v5-named resource with items_with_description blocks",
			Input: `
resource "cloudflare_zero_trust_list" "already_v5_blocks" {
  account_id = "abc123"
  name       = "Already V5 Blocks"
  type       = "DOMAIN"

  items_with_description {
    value       = "example.com"
    description = "Example domain"
  }

  items_with_description {
    value       = "test.com"
    description = "Test domain"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_list" "already_v5_blocks" {
  account_id = "abc123"
  name       = "Already V5 Blocks"
  type       = "DOMAIN"

  items = [{
    description = "Example domain"
    value       = "example.com"
    }, {
    description = "Test domain"
    value       = "test.com"
  }]
}`,
		},
		{
			Name: "v5-named resource with mixed items and blocks",
			Input: `
resource "cloudflare_zero_trust_list" "already_v5_mixed" {
  account_id = "abc123"
  name       = "Already V5 Mixed"
  type       = "EMAIL"
  items      = ["admin@example.com"]

  items_with_description {
    value       = "support@example.com"
    description = "Support email"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_list" "already_v5_mixed" {
  account_id = "abc123"
  name       = "Already V5 Mixed"
  type       = "EMAIL"

  items = [{
    description = "Support email"
    value       = "support@example.com"
    }, {
    description = null
    value       = "admin@example.com"
  }]
}`,
		},
		{
			Name: "v5-named resource with empty items array",
			Input: `
resource "cloudflare_zero_trust_list" "already_v5_empty" {
  account_id = "abc123"
  name       = "Already V5 Empty"
  type       = "IP"
  items      = []
}`,
			Expected: `
resource "cloudflare_zero_trust_list" "already_v5_empty" {
  account_id = "abc123"
  name       = "Already V5 Empty"
  type       = "IP"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
