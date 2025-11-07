package zone_dnssec

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
				Name: "Basic zone_dnssec with minimal fields",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
			},
			{
				Name: "Multiple zone_dnssec resources in one file",
				Input: `
resource "cloudflare_zone_dnssec" "example1" {
  zone_id = "abc123"
}

resource "cloudflare_zone_dnssec" "example2" {
  zone_id = "def456"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example1" {
  zone_id = "abc123"
}

resource "cloudflare_zone_dnssec" "example2" {
  zone_id = "def456"
}`,
			},
			{
				Name: "Zone DNSSEC with modified_on field (should be removed)",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id     = "abc123"
  modified_on = "2024-01-15T10:30:00Z"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "State with minimal fields",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with all fields populated",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "algorithm": "13",
        "digest": "ABC123DEF456",
        "digest_algorithm": "SHA256",
        "digest_type": "2",
        "ds": "12345 13 2 ABC123DEF456",
        "flags": 257.0,
        "key_tag": 12345.0,
        "key_type": "ECDSAP256SHA256",
        "modified_on": "2024-01-15T10:30:00Z",
        "public_key": "mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ=="
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "algorithm": "13",
        "digest": "ABC123DEF456",
        "digest_algorithm": "SHA256",
        "digest_type": "2",
        "ds": "12345 13 2 ABC123DEF456",
        "flags": 257.0,
        "key_tag": 12345.0,
        "key_type": "ECDSAP256SHA256",
        "modified_on": "2024-01-15T10:30:00Z",
        "public_key": "mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ=="
      }
    }]
  }]
}`,
			},
			{
				Name: "State with float64 fields",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "flags": 256.0,
        "key_tag": 42345.0
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "flags": 256.0,
        "key_tag": 42345.0
      }
    }]
  }]
}`,
			},
			{
				Name: "State with integer fields (v4 format) converted to float64",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "flags": 257,
        "key_tag": 12345,
        "algorithm": "13",
        "digest": "ABC123DEF456",
        "digest_algorithm": "SHA256",
        "digest_type": "2",
        "ds": "12345 13 2 ABC123DEF456",
        "key_type": "ECDSAP256SHA256",
        "modified_on": "2024-01-15T10:30:00Z",
        "public_key": "mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ=="
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "flags": 257.0,
        "key_tag": 12345.0,
        "algorithm": "13",
        "digest": "ABC123DEF456",
        "digest_algorithm": "SHA256",
        "digest_type": "2",
        "ds": "12345 13 2 ABC123DEF456",
        "key_type": "ECDSAP256SHA256",
        "modified_on": "2024-01-15T10:30:00Z",
        "public_key": "mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ=="
      }
    }]
  }]
}`,
			},
			{
				Name: "State with pending status (should be converted to active)",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "pending",
        "algorithm": "13"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "algorithm": "13"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with pending-disabled status (should be converted to disabled)",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "pending-disabled",
        "algorithm": "13"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "disabled",
        "algorithm": "13"
      }
    }]
  }]
}`,
			},
			{
				Name: "State with modified_on in v4 format (should be converted to RFC3339)",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "modified_on": "Tue, 04 Nov 2025 21:52:44 +0000"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zone_dnssec",
    "name": "example",
    "instances": [{
      "attributes": {
        "zone_id": "abc123",
        "status": "active",
        "modified_on": "2025-11-04T21:52:44Z"
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
