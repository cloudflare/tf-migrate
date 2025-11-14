package zone

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
				Name: "Basic zone datasource lookup by zone_id - no changes needed",
				Input: `
data "cloudflare_zone" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
}`,
				Expected: `data "cloudflare_zone" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
}`,
			},
			{
				Name: "Zone datasource with filter - no changes needed",
				Input: `
data "cloudflare_zone" "example" {
  filter {
    name = "example.com"
  }
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter {
    name = "example.com"
  }
}`,
			},
			{
				Name: "Zone datasource with complex filter",
				Input: `
data "cloudflare_zone" "example" {
  filter {
    account {
      id = "f037e56e89293a057740de681ac9abbe"
    }
    name      = "example.com"
    status    = "active"
    order     = "name"
    direction = "asc"
    match     = "all"
  }
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter {
    account {
      id = "f037e56e89293a057740de681ac9abbe"
    }
    name      = "example.com"
    status    = "active"
    order     = "name"
    direction = "asc"
    match     = "all"
  }
}`,
			},
			{
				Name: "Multiple zone datasources in one file",
				Input: `
data "cloudflare_zone" "zone1" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
}

data "cloudflare_zone" "zone2" {
  filter {
    name = "another-example.com"
  }
}`,
				Expected: `data "cloudflare_zone" "zone1" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
}

data "cloudflare_zone" "zone2" {
  filter {
    name = "another-example.com"
  }
}`,
			},
			{
				Name: "Zone datasource with filter using operators",
				Input: `
data "cloudflare_zone" "example" {
  filter {
    name = "starts_with:example"
  }
}`,
				Expected: `data "cloudflare_zone" "example" {
  filter {
    name = "starts_with:example"
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
				Name: "Basic zone state - only schema_version added",
				Input: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "active",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"],
  "created_on": "2023-01-15T10:00:00Z"
}`,
				Expected: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "active",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"],
  "created_on": "2023-01-15T10:00:00Z",
  "schema_version": 0
}`,
			},
			{
				Name: "Zone with complete data including nested objects",
				Input: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "active",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"],
  "original_name_servers": ["ns1.example.com", "ns2.example.com"],
  "vanity_name_servers": [],
  "permissions": ["#zone:read", "#zone:edit"],
  "activated_on": "2023-01-15T10:30:00Z",
  "created_on": "2023-01-15T10:00:00Z",
  "modified_on": "2023-01-15T10:30:00Z",
  "original_dnshost": "example-dns",
  "original_registrar": "example-registrar",
  "verification_key": "verification-key-here",
  "cname_suffix": null,
  "account": {
    "id": "f037e56e89293a057740de681ac9abbe",
    "name": "Example Account"
  },
  "owner": {
    "id": "7c5dae5552338874e5053f2534d2767a",
    "name": "John Doe",
    "type": "user"
  },
  "plan": {
    "id": "0feeeeeeeeeeeeeeeeeeeeeeeeeeeeee1",
    "name": "Free Plan",
    "price": 0,
    "currency": "USD",
    "frequency": "monthly",
    "is_subscribed": true,
    "can_subscribe": false,
    "legacy_id": "free",
    "legacy_discount": false,
    "externally_managed": false
  },
  "meta": {
    "cdn_only": false,
    "custom_certificate_quota": 0,
    "dns_only": false,
    "foundation_dns": false,
    "page_rule_quota": 3,
    "phishing_detected": false,
    "step": 2
  },
  "tenant": null,
  "tenant_unit": null
}`,
				Expected: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "active",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": ["ns1.cloudflare.com", "ns2.cloudflare.com"],
  "original_name_servers": ["ns1.example.com", "ns2.example.com"],
  "vanity_name_servers": [],
  "permissions": ["#zone:read", "#zone:edit"],
  "activated_on": "2023-01-15T10:30:00Z",
  "created_on": "2023-01-15T10:00:00Z",
  "modified_on": "2023-01-15T10:30:00Z",
  "original_dnshost": "example-dns",
  "original_registrar": "example-registrar",
  "verification_key": "verification-key-here",
  "cname_suffix": null,
  "account": {
    "id": "f037e56e89293a057740de681ac9abbe",
    "name": "Example Account"
  },
  "owner": {
    "id": "7c5dae5552338874e5053f2534d2767a",
    "name": "John Doe",
    "type": "user"
  },
  "plan": {
    "id": "0feeeeeeeeeeeeeeeeeeeeeeeeeeeeee1",
    "name": "Free Plan",
    "price": 0,
    "currency": "USD",
    "frequency": "monthly",
    "is_subscribed": true,
    "can_subscribe": false,
    "legacy_id": "free",
    "legacy_discount": false,
    "externally_managed": false
  },
  "meta": {
    "cdn_only": false,
    "custom_certificate_quota": 0,
    "dns_only": false,
    "foundation_dns": false,
    "page_rule_quota": 3,
    "phishing_detected": false,
    "step": 2
  },
  "tenant": null,
  "tenant_unit": null,
  "schema_version": 0
}`,
			},
			{
				Name: "Minimal zone state",
				Input: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "pending",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": [],
  "created_on": "2023-01-15T10:00:00Z"
}`,
				Expected: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "pending",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": [],
  "created_on": "2023-01-15T10:00:00Z",
  "schema_version": 0
}`,
			},
			{
				Name: "Zone state with deprecated fields preserved",
				Input: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "active",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": ["ns1.cloudflare.com"],
  "permissions": ["#zone:read"],
  "plan": {
    "id": "plan-id",
    "name": "Free Plan"
  },
  "created_on": "2023-01-15T10:00:00Z"
}`,
				Expected: `{
  "id": "023e105f4ecef8ad9ca31a8372d0c353",
  "zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
  "name": "example.com",
  "status": "active",
  "type": "full",
  "paused": false,
  "development_mode": 0,
  "name_servers": ["ns1.cloudflare.com"],
  "permissions": ["#zone:read"],
  "plan": {
    "id": "plan-id",
    "name": "Free Plan"
  },
  "created_on": "2023-01-15T10:00:00Z",
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
