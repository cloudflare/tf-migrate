package zero_trust_device_managed_networks

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
				Name: "Basic resource with all fields",
				Input: `
resource "cloudflare_device_managed_networks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-network"
  type       = "tls"

  config {
    tls_sockaddr = "example.com:443"
    sha256       = "abcd1234"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_device_managed_networks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-network"
  type       = "tls"
  config = {
    tls_sockaddr = "example.com:443"
    sha256       = "abcd1234"
  }
}`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `
resource "cloudflare_device_managed_networks" "network1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "network-1"
  type       = "tls"

  config {
    tls_sockaddr = "network1.example.com:443"
    sha256       = "abc123"
  }
}

resource "cloudflare_device_managed_networks" "network2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "network-2"
  type       = "tls"

  config {
    tls_sockaddr = "network2.example.com:443"
    sha256       = "def456"
  }
}`,
				Expected: `resource "cloudflare_zero_trust_device_managed_networks" "network1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "network-1"
  type       = "tls"
  config = {
    tls_sockaddr = "network1.example.com:443"
    sha256       = "abc123"
  }
}

resource "cloudflare_zero_trust_device_managed_networks" "network2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "network-2"
  type       = "tls"
  config = {
    tls_sockaddr = "network2.example.com:443"
    sha256       = "def456"
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
				Name: "Basic state with array config",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_device_managed_networks",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "abc123-network-id",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "example-network",
        "type": "tls",
        "config": [{
          "tls_sockaddr": "example.com:443",
          "sha256": "abcd1234"
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
    "type": "cloudflare_zero_trust_device_managed_networks",
    "name": "example",
    "instances": [{
      "attributes": {
        "id": "abc123-network-id",
        "network_id": "abc123-network-id",
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "name": "example-network",
        "type": "tls",
        "config": {
          "tls_sockaddr": "example.com:443",
          "sha256": "abcd1234"
        }
      },
      "schema_version": 0
    }]
  }]
}`,
			},
			{
				Name: "Multiple instances",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_device_managed_networks",
    "name": "network",
    "instances": [
      {
        "index_key": 0,
        "attributes": {
          "id": "network-id-1",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "network-1",
          "type": "tls",
          "config": [{
            "tls_sockaddr": "network1.example.com:443",
            "sha256": "abc123"
          }]
        },
        "schema_version": 0
      },
      {
        "index_key": 1,
        "attributes": {
          "id": "network-id-2",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "network-2",
          "type": "tls",
          "config": [{
            "tls_sockaddr": "network2.example.com:443",
            "sha256": "def456"
          }]
        },
        "schema_version": 0
      }
    ]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_device_managed_networks",
    "name": "network",
    "instances": [
      {
        "index_key": 0,
        "attributes": {
          "id": "network-id-1",
          "network_id": "network-id-1",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "network-1",
          "type": "tls",
          "config": {
            "tls_sockaddr": "network1.example.com:443",
            "sha256": "abc123"
          }
        },
        "schema_version": 0
      },
      {
        "index_key": 1,
        "attributes": {
          "id": "network-id-2",
          "network_id": "network-id-2",
          "account_id": "f037e56e89293a057740de681ac9abbe",
          "name": "network-2",
          "type": "tls",
          "config": {
            "tls_sockaddr": "network2.example.com:443",
            "sha256": "def456"
          }
        },
        "schema_version": 0
      }
    ]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
