package zero_trust_access_group

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic resource rename",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include {
    email = ["user@example.com"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include = [
    {
      email = {
        email = "user@example.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "multiple email selectors",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include {
    email = ["user1@example.com", "user2@example.com"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include = [
    {
      email = {
        email = "user1@example.com"
      }
    },
    {
      email = {
        email = "user2@example.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "multiple selector types",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include {
    email = ["user@example.com"]
    ip    = ["192.168.1.0/24"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include = [
    {
      email = {
        email = "user@example.com"
      }
    },
    {
      ip = {
        ip = "192.168.1.0/24"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "boolean selectors",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Boolean Group"

  include {
    everyone = true
  }

  exclude {
    certificate = true
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Boolean Group"

  include = [
    {
      everyone = {}
    },
  ]

  exclude = [
    {
      certificate = {}
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "email_domain with field rename",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Domain Group"

  include {
    email_domain = ["example.com", "test.com"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Domain Group"

  include = [
    {
      email_domain = {
        domain = "example.com"
      }
    },
    {
      email_domain = {
        domain = "test.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "geo with field rename",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Geo Group"

  include {
    geo = ["US", "CA", "GB"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Geo Group"

  include = [
    {
      geo = {
        country_code = "US"
      }
    },
    {
      geo = {
        country_code = "CA"
      }
    },
    {
      geo = {
        country_code = "GB"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "service_token with field rename",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Token Group"

  include {
    service_token          = ["token-1", "token-2"]
    any_valid_service_token = true
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Token Group"

  include = [
    {
      service_token = {
        token_id = "token-1"
      }
    },
    {
      service_token = {
        token_id = "token-2"
      }
    },
    {
      any_valid_service_token = {}
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "device_posture with field rename",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Posture Group"

  include {
    device_posture = ["posture-1", "posture-2"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Posture Group"

  include = [
    {
      device_posture = {
        integration_uid = "posture-1"
      }
    },
    {
      device_posture = {
        integration_uid = "posture-2"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "common_name scalar",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Common Name Group"

  include {
    common_name = "client.example.com"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Common Name Group"

  include = [
    {
      common_name = {
        common_name = "client.example.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "common_names overflow array",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Common Names Group"

  include {
    common_names = ["client1.example.com", "client2.example.com"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Common Names Group"

  include = [
    {
      common_name = {
        common_name = "client1.example.com"
      }
    },
    {
      common_name = {
        common_name = "client2.example.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "auth_method scalar",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Auth Method Group"

  include {
    auth_method = "email"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Auth Method Group"

  include = [
    {
      auth_method = {
        auth_method = "email"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "all three rule types",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "All Rules Group"

  include {
    email = ["user@example.com"]
  }

  exclude {
    ip = ["192.168.0.0/16"]
  }

  require {
    email_domain = ["example.com"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "All Rules Group"

  include = [
    {
      email = {
        email = "user@example.com"
      }
    },
  ]

  exclude = [
    {
      ip = {
        ip = "192.168.0.0/16"
      }
    },
  ]

  require = [
    {
      email_domain = {
        domain = "example.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "github teams explosion",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "GitHub Group"

  include {
    github {
      name                 = "my-org"
      identity_provider_id = "idp-123"
      teams                = ["team-1", "team-2"]
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "GitHub Group"

  include = [
    {
      github_organization = {
        name                 = "my-org"
        team                 = "team-1"
        identity_provider_id = "idp-123"
      }
    },
    {
      github_organization = {
        name                 = "my-org"
        team                 = "team-2"
        identity_provider_id = "idp-123"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "github teams explosion with resource reference identity_provider_id",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "GitHub Group"

  include {
    github {
      name                 = "my-org"
      identity_provider_id = cloudflare_access_identity_provider.my_idp.id
      teams                = ["team-1", "team-2"]
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "GitHub Group"

  include = [
    {
      github_organization = {
        name                 = "my-org"
        team                 = "team-1"
        identity_provider_id = cloudflare_access_identity_provider.my_idp.id
      }
    },
    {
      github_organization = {
        name                 = "my-org"
        team                 = "team-2"
        identity_provider_id = cloudflare_access_identity_provider.my_idp.id
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
			to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "external_evaluation block conversion",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "External Eval Group"

  include {
    external_evaluation {
      evaluate_url = "https://example.com/eval"
      keys_url     = "https://example.com/keys"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "External Eval Group"

  include = [
    {
      external_evaluation = {
        evaluate_url = "https://example.com/eval"
        keys_url     = "https://example.com/keys"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "auth_context block conversion",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Auth Context Group"

  include {
    auth_context {
      id                   = "ctx-id"
      ac_id                = "ac-id"
      identity_provider_id = "idp-123"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Auth Context Group"

  include = [
    {
      auth_context = {
        id                   = "ctx-id"
        ac_id                = "ac-id"
        identity_provider_id = "idp-123"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		// Note: gsuite, azure, okta, saml complex nested tests are skipped
		// as the v4 schema uses these as array fields, not nested blocks
		// Testing these would require actual v4 state format which has arrays
		{
			Name: "complex multi-selector",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Complex Group"

  include {
    email        = ["admin@example.com", "manager@example.com"]
    email_domain = ["example.com"]
    ip           = ["10.0.0.0/8"]
    everyone     = true
  }

  exclude {
    ip  = ["10.0.1.0/24"]
    geo = ["CN", "RU"]
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Complex Group"

  include = [
    {
      email = {
        email = "admin@example.com"
      }
    },
    {
      email = {
        email = "manager@example.com"
      }
    },
    {
      email_domain = {
        domain = "example.com"
      }
    },
    {
      ip = {
        ip = "10.0.0.0/8"
      }
    },
    {
      everyone = {}
    },
  ]

  exclude = [
    {
      ip = {
        ip = "10.0.1.0/24"
      }
    },
    {
      geo = {
        country_code = "CN"
      }
    },
    {
      geo = {
        country_code = "RU"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		// ============================================================================
		// cf-terraforming format tests (multiple blocks with nested object syntax)
		// These test the format generated by cf-terraforming which uses multiple
		// include/exclude/require blocks instead of a single block with arrays
		// ============================================================================
		{
			Name: "cf_terraforming_multiple_include_blocks",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    email = {
      email = "user1@example.com"
    }
  }
  include {
    email = {
      email = "user2@example.com"
    }
  }
  include {
    ip = {
      ip = "192.168.1.0/24"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      email = {
        email = "user1@example.com"
      }
    },
    {
      email = {
        email = "user2@example.com"
      }
    },
    {
      ip = {
        ip = "192.168.1.0/24"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_mixed_conditions",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    email = {
      email = "user@example.com"
    }
  }
  exclude {
    ip = {
      ip = "10.0.0.0/8"
    }
  }
  exclude {
    geo = {
      country_code = "CN"
    }
  }
  require {
    email_domain = {
      domain = "company.com"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      email = {
        email = "user@example.com"
      }
    },
  ]
  exclude = [
    {
      ip = {
        ip = "10.0.0.0/8"
      }
    },
    {
      geo = {
        country_code = "CN"
      }
    },
  ]
  require = [
    {
      email_domain = {
        domain = "company.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_boolean_selectors_empty_objects",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    everyone = {}
  }
  exclude {
    certificate = {}
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      everyone = {}
    },
  ]
  exclude = [
    {
      certificate = {}
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_email_domain_selector",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    email_domain = {
      domain = "company.com"
    }
  }
  include {
    email_domain = {
      domain = "partner.com"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      email_domain = {
        domain = "company.com"
      }
    },
    {
      email_domain = {
        domain = "partner.com"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_geo_selector",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    geo = {
      country_code = "US"
    }
  }
  include {
    geo = {
      country_code = "CA"
    }
  }
  include {
    geo = {
      country_code = "GB"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      geo = {
        country_code = "US"
      }
    },
    {
      geo = {
        country_code = "CA"
      }
    },
    {
      geo = {
        country_code = "GB"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_login_method_device_posture",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    login_method = {
      id = "method-id-1"
    }
  }
  include {
    login_method = {
      id = "method-id-2"
    }
  }
  include {
    device_posture = {
      integration_uid = "posture-id-1"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      login_method = {
        id = "method-id-1"
      }
    },
    {
      login_method = {
        id = "method-id-2"
      }
    },
    {
      device_posture = {
        integration_uid = "posture-id-1"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_any_valid_service_token",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    any_valid_service_token = {}
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      any_valid_service_token = {}
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_common_name_auth_method",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    common_name = {
      common_name = "client.example.com"
    }
  }
  include {
    auth_method = {
      auth_method = "email"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      common_name = {
        common_name = "client.example.com"
      }
    },
    {
      auth_method = {
        auth_method = "email"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_ip_list_email_list",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    ip_list = {
      id = "iplist-id-1"
    }
  }
  include {
    ip_list = {
      id = "iplist-id-2"
    }
  }
  include {
    email_list = {
      id = "emaillist-id-1"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      ip_list = {
        id = "iplist-id-1"
      }
    },
    {
      ip_list = {
        id = "iplist-id-2"
      }
    },
    {
      email_list = {
        id = "emaillist-id-1"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
		{
			Name: "cf_terraforming_group_service_token",
			Input: `
resource "cloudflare_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include {
    group = {
      id = "group-id-1"
    }
  }
  include {
    service_token = {
      token_id = "token-id-1"
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"
  include = [
    {
      group = {
        id = "group-id-1"
      }
    },
    {
      service_token = {
        token_id = "token-id-1"
      }
    },
  ]
}
moved {
  from = cloudflare_access_group.test
  to   = cloudflare_zero_trust_access_group.test
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "handles old name",
			resourceType: "cloudflare_access_group",
			expected:     true,
		},
		{
			name:         "handles new name",
			resourceType: "cloudflare_zero_trust_access_group",
			expected:     true,
		},
		{
			name:         "rejects other resources",
			resourceType: "cloudflare_other_resource",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%s) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}
