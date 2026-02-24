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
}

moved {
  from = cloudflare_device_managed_networks.example
  to   = cloudflare_zero_trust_device_managed_networks.example
}
`,
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

moved {
  from = cloudflare_device_managed_networks.network1
  to   = cloudflare_zero_trust_device_managed_networks.network1
}

resource "cloudflare_zero_trust_device_managed_networks" "network2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "network-2"
  type       = "tls"
  config = {
    tls_sockaddr = "network2.example.com:443"
    sha256       = "def456"
  }
}

moved {
  from = cloudflare_device_managed_networks.network2
  to   = cloudflare_zero_trust_device_managed_networks.network2
}
`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// State transformation tests removed - state migration is now handled by provider StateUpgraders
	// tf-migrate only transforms configs and generates moved blocks
	// The provider's MoveState and UpgradeState handlers automatically transform state when Terraform runs
}
