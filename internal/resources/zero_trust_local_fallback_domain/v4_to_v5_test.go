package zero_trust_local_fallback_domain

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic default profile - no policy_id",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "default profile with all domain fields",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix      = "example.com"
    description = "Example domain"
    dns_server  = ["1.1.1.1", "8.8.8.8"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "example.com"
      description = "Example domain"
      dns_server  = ["1.1.1.1", "8.8.8.8"]
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "default profile with multiple domains",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix      = "example.com"
    description = "Example domain"
  }

  domains {
    suffix     = "another.com"
    dns_server = ["1.1.1.1"]
  }

  domains {
    suffix = "third.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix      = "example.com"
      description = "Example domain"
    },
    {
      suffix     = "another.com"
      dns_server = ["1.1.1.1"]
    },
    {
      suffix = "third.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "deprecated resource name - default profile",
			Input: `
resource "cloudflare_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "default profile with policy_id = null",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = null

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "basic custom profile - with policy_id",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "custom profile with all domain fields",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix      = "corp.example.com"
    description = "Corporate domain"
    dns_server  = ["10.0.0.1"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix      = "corp.example.com"
      description = "Corporate domain"
      dns_server  = ["10.0.0.1"]
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "custom profile with multiple domains",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "corp1.example.com"
  }

  domains {
    suffix      = "corp2.example.com"
    description = "Secondary corporate domain"
    dns_server  = ["10.0.0.2", "10.0.0.3"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "corp1.example.com"
    },
    {
      suffix      = "corp2.example.com"
      description = "Secondary corporate domain"
      dns_server  = ["10.0.0.2", "10.0.0.3"]
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "deprecated resource name - custom profile",
			Input: `
resource "cloudflare_fallback_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "example.com"
    }
  ]
}

moved {
  from = cloudflare_fallback_domain.example
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.example
}`,
		},
		{
			Name: "multiple resources - mixed default and custom",
			Input: `
resource "cloudflare_zero_trust_local_fallback_domain" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains {
    suffix = "default.example.com"
  }
}

resource "cloudflare_zero_trust_local_fallback_domain" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains {
    suffix = "custom.example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_device_default_profile_local_domain_fallback" "default" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  domains = [
    {
      suffix = "default.example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.default
  to   = cloudflare_zero_trust_device_default_profile_local_domain_fallback.default
}

resource "cloudflare_zero_trust_device_custom_profile_local_domain_fallback" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "policy123"

  domains = [
    {
      suffix = "custom.example.com"
    }
  ]
}

moved {
  from = cloudflare_zero_trust_local_fallback_domain.custom
  to   = cloudflare_zero_trust_device_custom_profile_local_domain_fallback.custom
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformation_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "handles current v4 name",
			resourceType: "cloudflare_zero_trust_local_fallback_domain",
			expected:     true,
		},
		{
			name:         "handles deprecated v4 name",
			resourceType: "cloudflare_fallback_domain",
			expected:     true,
		},
		{
			name:         "does not handle unrelated resource",
			resourceType: "cloudflare_other_resource",
			expected:     false,
		},
		{
			name:         "does not handle v5 default profile name",
			resourceType: "cloudflare_zero_trust_device_default_profile_local_domain_fallback",
			expected:     false,
		},
		{
			name:         "does not handle v5 custom profile name",
			resourceType: "cloudflare_zero_trust_device_custom_profile_local_domain_fallback",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}
