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
  items {
    value = "192.168.1.1"
  }
  items {
    value = "10.0.0.0/8"
  }
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
  items {
    value       = "172.16.0.0/12"
    description = "Private network range"
  }
  items {
    value       = "203.0.113.0/24"
    description = "Test network"
  }
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
  items {
    value = "192.168.1.1"
  }
  items {
    value = "10.0.0.0/8"
  }
  items {
    value       = "172.16.0.0/12"
    description = "Private network"
  }
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
  items {
    value = "example.com"
  }
  items {
    value = "*.cloudflare.com"
  }
  items {
    value = "api.github.com"
  }
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
  items {
    value       = "https://malicious.example.com/path"
    description = "Known phishing"
  }
  items {
    value       = "http://phishing.site.com"
    description = "Suspicious site"
  }
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
  items {
    value = "admin@example.com"
  }
  items {
    value = "security@cloudflare.com"
  }
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
  items {
    value = "00:11:22:33:44:55:66"
  }
  items {
    value = "AA:BB:CC:DD:EE:FF"
  }
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
  items {
    value = "192.168.1.1"
  }
}

resource "cloudflare_zero_trust_list" "list2" {
  account_id = "abc123"
  name       = "List 2"
  type       = "DOMAIN"
  items {
    value = "example.com"
  }
  items {
    value       = "test.com"
    description = "Test domain"
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Simple IP list state transformation",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_teams_list",
					"name": "ip_list",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "IP Allowlist",
							"type": "IP",
							"items": ["192.168.1.1", "10.0.0.0/8"]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_list",
					"name": "ip_list",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "IP Allowlist",
							"type": "IP",
							"items": [
								{"value": "192.168.1.1"},
								{"value": "10.0.0.0/8"}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "State with items_with_description",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_teams_list",
					"name": "mixed",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "Mixed List",
							"type": "IP",
							"items": ["192.168.1.1"],
							"items_with_description": [{
								"value": "172.16.0.0/12",
								"description": "Private network"
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_list",
					"name": "mixed",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "Mixed List",
							"type": "IP",
							"items": [
								{"value": "192.168.1.1"},
								{"value": "172.16.0.0/12", "description": "Private network"}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "Empty items list in state",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_teams_list",
					"name": "empty",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "Empty List",
							"type": "IP",
							"items": []
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_list",
					"name": "empty",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "Empty List",
							"type": "IP"
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "State with existing ID field",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_teams_list",
					"name": "with_id",
					"instances": [{
						"attributes": {
							"id": "existing-id-123",
							"account_id": "abc123",
							"name": "List with ID",
							"type": "DOMAIN",
							"items": ["example.com"],
							"description": "Test list"
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_list",
					"name": "with_id",
					"instances": [{
						"attributes": {
							"id": "existing-id-123",
							"account_id": "abc123",
							"name": "List with ID",
							"type": "DOMAIN",
							"items": [
								{"value": "example.com"}
							],
							"description": "Test list"
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "Multiple resources in state",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_teams_list",
					"name": "list1",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "List 1",
							"type": "IP",
							"items": ["192.168.1.1"]
						},
						"schema_version": 0
					}]
				}, {
					"type": "cloudflare_teams_list",
					"name": "list2",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "List 2",
							"type": "EMAIL",
							"items": ["admin@example.com"]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_list",
					"name": "list1",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "List 1",
							"type": "IP",
							"items": [
								{"value": "192.168.1.1"}
							]
						},
						"schema_version": 0
					}]
				}, {
					"type": "cloudflare_zero_trust_list",
					"name": "list2",
					"instances": [{
						"attributes": {
							"account_id": "abc123",
							"name": "List 2",
							"type": "EMAIL",
							"items": [
								{"value": "admin@example.com"}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}