package zone

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "basic zone transformation",
			Input: `
resource "cloudflare_zone" "example" {
  zone       = "example.com"
  account_id = "abc123"
}`,
			Expected: `
resource "cloudflare_zone" "example" {
  name = "example.com"
  account = {
    id = "abc123"
  }
}`,
		},
		{
			Name: "zone with all v4 attributes",
			Input: `
resource "cloudflare_zone" "example" {
  zone                = "example.com"
  account_id          = "abc123"
  paused              = true
  type                = "partial"
  jump_start          = true
  plan                = "enterprise"
  vanity_name_servers = ["ns1.example.com", "ns2.example.com"]
}`,
			Expected: `
resource "cloudflare_zone" "example" {
  paused              = true
  type                = "partial"
  vanity_name_servers = ["ns1.example.com", "ns2.example.com"]
  name                = "example.com"
  account = {
    id = "abc123"
  }
}`,
		},
		{
			Name: "zone with only removed attributes",
			Input: `
resource "cloudflare_zone" "example" {
  zone       = "example.com"
  account_id = "abc123"
  jump_start = false
  plan       = "free"
}`,
			Expected: `
resource "cloudflare_zone" "example" {
  name = "example.com"
  account = {
    id = "abc123"
  }
}`,
		},
		{
			Name: "zone with unicode domain",
			Input: `
resource "cloudflare_zone" "unicode" {
  zone       = "例え.テスト"
  account_id = "def456"
  type       = "full"
}`,
			Expected: `
resource "cloudflare_zone" "unicode" {
  type = "full"
  name = "例え.テスト"
  account = {
    id = "def456"
  }
}`,
		},
		{
			Name: "zone with variable reference for account_id",
			Input: `
resource "cloudflare_zone" "complex" {
  zone       = "complex.example.com"
  account_id = var.account_id
  jump_start = var.enable_jump_start
}`,
			Expected: `
resource "cloudflare_zone" "complex" {
  name = "complex.example.com"
  account = {
    id = var.account_id
  }
}`,
		},
		{
			Name: "zone with expression for account_id",
			Input: `
resource "cloudflare_zone" "expr" {
  zone       = "expr.example.com"
  account_id = data.cloudflare_accounts.main.accounts[0].id
  type       = "full"
}`,
			Expected: `
resource "cloudflare_zone" "expr" {
  type = "full"
  name = "expr.example.com"
  account = {
    id = data.cloudflare_accounts.main.accounts[0].id
  }
}`,
		},
		{
			Name: "multiple zones in same config",
			Input: `
resource "cloudflare_zone" "primary" {
  zone       = "primary.example.com"
  account_id = "account1"
  plan       = "pro"
}

resource "cloudflare_zone" "secondary" {
  zone       = "secondary.example.com"
  account_id = "account2"
  type       = "partial"
  jump_start = true
}`,
			Expected: `
resource "cloudflare_zone" "primary" {
  name = "primary.example.com"
  account = {
    id = "account1"
  }
}

resource "cloudflare_zone" "secondary" {
  type = "partial"
  name = "secondary.example.com"
  account = {
    id = "account2"
  }
}`,
		},
		{
			Name: "zone with paused = false (default value)",
			Input: `
resource "cloudflare_zone" "paused_false" {
  zone       = "example.com"
  account_id = "abc123"
  paused     = false
}`,
			Expected: `
resource "cloudflare_zone" "paused_false" {
  paused = false
  name   = "example.com"
  account = {
    id = "abc123"
  }
}`,
		},
		{
			Name: "zone with type = full (default value)",
			Input: `
resource "cloudflare_zone" "type_full" {
  zone       = "example.com"
  account_id = "abc123"
  type       = "full"
}`,
			Expected: `
resource "cloudflare_zone" "type_full" {
  type = "full"
  name = "example.com"
  account = {
    id = "abc123"
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "basic zone state transformation",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "zone123",
    "zone": "example.com",
    "account_id": "abc123",
    "paused": false,
    "type": "full",
    "status": "active",
    "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "zone123",
    "name": "example.com",
    "account": {
      "id": "abc123"
    },
    "paused": false,
    "type": "full",
    "status": "active",
    "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"]
  }
}`,
		},
		{
			Name: "zone state with removed attributes",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "zone456",
    "zone": "example.com",
    "account_id": "abc123",
    "jump_start": true,
    "plan": "enterprise",
    "paused": false,
    "type": "full"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "zone456",
    "name": "example.com",
    "account": {
      "id": "abc123"
    },
    "paused": false,
    "type": "full"
  }
}`,
		},
		{
			Name: "zone state with vanity name servers",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "zone789",
    "zone": "custom.example.com",
    "account_id": "def456",
    "type": "full",
    "vanity_name_servers": ["ns1.custom.example.com", "ns2.custom.example.com"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "zone789",
    "name": "custom.example.com",
    "account": {
      "id": "def456"
    },
    "type": "full",
    "vanity_name_servers": ["ns1.custom.example.com", "ns2.custom.example.com"]
  }
}`,
		},
		{
			Name: "zone state with meta object",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "zone999",
    "zone": "meta.example.com",
    "account_id": "ghi789",
    "meta": {
      "cdn_only": true,
      "custom_certificate_quota": 10,
      "page_rule_quota": 125
    }
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "zone999",
    "name": "meta.example.com",
    "account": {
      "id": "ghi789"
    },
    "meta": {
      "cdn_only": true,
      "custom_certificate_quota": 10,
      "page_rule_quota": 125
    }
  }
}`,
		},
		{
			Name: "minimal zone state (only required fields)",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "zoneminimal",
    "zone": "minimal.example.com",
    "account_id": "minimal123"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "zoneminimal",
    "name": "minimal.example.com",
    "account": {
      "id": "minimal123"
    }
  }
}`,
		},
		{
			Name: "invalid zone state (missing attributes)",
			Input: `{
  "schema_version": 1,
  "attributes": {}
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {}
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "cloudflare_zone resource",
			resourceType: "cloudflare_zone",
			expected:     true,
		},
		{
			name:         "different resource",
			resourceType: "cloudflare_dns_record",
			expected:     false,
		},
		{
			name:         "empty resource",
			resourceType: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	expected := "cloudflare_zone"
	result := migrator.GetResourceType()
	if result != expected {
		t.Errorf("GetResourceType() = %q, want %q", result, expected)
	}
}

