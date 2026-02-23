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
