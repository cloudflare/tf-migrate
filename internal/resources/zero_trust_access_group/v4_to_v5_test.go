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
      ip = {
        ip = "10.0.0.0/8"
      }
    },
    {
      email_domain = {
        domain = "example.com"
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
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "basic email selector",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Test Group",
    "include": [
      {
        "email": ["user@example.com"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Test Group",
    "include": [
      {
        "email": {
          "email": "user@example.com"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "multiple email selectors",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Test Group",
    "include": [
      {
        "email": ["user1@example.com", "user2@example.com"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Test Group",
    "include": [
      {
        "email": {
          "email": "user1@example.com"
        }
      },
      {
        "email": {
          "email": "user2@example.com"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "multiple selector types",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Test Group",
    "include": [
      {
        "email": ["user@example.com"],
        "ip": ["192.168.1.0/24"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Test Group",
    "include": [
      {
        "email": {
          "email": "user@example.com"
        }
      },
      {
        "ip": {
          "ip": "192.168.1.0/24"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "boolean selectors",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Boolean Group",
    "include": [
      {
        "everyone": true
      }
    ],
    "exclude": [
      {
        "certificate": true
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Boolean Group",
    "include": [
      {
        "everyone": {}
      }
    ],
    "exclude": [
      {
        "certificate": {}
      }
    ]
  }
}`,
		},
		{
			Name: "email_domain with field rename",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Domain Group",
    "include": [
      {
        "email_domain": ["example.com", "test.com"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Domain Group",
    "include": [
      {
        "email_domain": {
          "domain": "example.com"
        }
      },
      {
        "email_domain": {
          "domain": "test.com"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "geo with field rename",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Geo Group",
    "include": [
      {
        "geo": ["US", "CA"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Geo Group",
    "include": [
      {
        "geo": {
          "country_code": "US"
        }
      },
      {
        "geo": {
          "country_code": "CA"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "service_token with field rename",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Token Group",
    "include": [
      {
        "service_token": ["token-1", "token-2"],
        "any_valid_service_token": true
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Token Group",
    "include": [
      {
        "service_token": {
          "token_id": "token-1"
        }
      },
      {
        "service_token": {
          "token_id": "token-2"
        }
      },
      {
        "any_valid_service_token": {}
      }
    ]
  }
}`,
		},
		{
			Name: "device_posture with field rename",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Posture Group",
    "include": [
      {
        "device_posture": ["posture-1"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Posture Group",
    "include": [
      {
        "device_posture": {
          "integration_uid": "posture-1"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "common_name scalar",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Common Name Group",
    "include": [
      {
        "common_name": "client.example.com"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Common Name Group",
    "include": [
      {
        "common_name": {
          "common_name": "client.example.com"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "auth_method scalar",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Auth Method Group",
    "include": [
      {
        "auth_method": "email"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Auth Method Group",
    "include": [
      {
        "auth_method": {
          "auth_method": "email"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "all three rule types",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "All Rules Group",
    "include": [
      {
        "email": ["user@example.com"]
      }
    ],
    "exclude": [
      {
        "ip": ["192.168.0.0/16"]
      }
    ],
    "require": [
      {
        "email_domain": ["example.com"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "All Rules Group",
    "include": [
      {
        "email": {
          "email": "user@example.com"
        }
      }
    ],
    "exclude": [
      {
        "ip": {
          "ip": "192.168.0.0/16"
        }
      }
    ],
    "require": [
      {
        "email_domain": {
          "domain": "example.com"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "github teams explosion",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "GitHub Group",
    "include": [
      {
        "github": [
          {
            "name": "my-org",
            "identity_provider_id": "idp-123",
            "teams": ["team-1", "team-2"]
          }
        ]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "GitHub Group",
    "include": [
      {
        "github_organization": {
          "name": "my-org",
          "identity_provider_id": "idp-123",
          "team": "team-1"
        }
      },
      {
        "github_organization": {
          "name": "my-org",
          "identity_provider_id": "idp-123",
          "team": "team-2"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "gsuite with take-first",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "GSuite Group",
    "include": [
      {
        "gsuite": [
          {
            "email": ["admin@example.com", "user@example.com"],
            "identity_provider_id": "idp-456"
          }
        ]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "GSuite Group",
    "include": [
      {
        "gsuite": {
          "email": "admin@example.com",
          "identity_provider_id": "idp-456"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "azure with take-first",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Azure Group",
    "include": [
      {
        "azure": [
          {
            "id": ["id-1", "id-2"],
            "identity_provider_id": "idp-789"
          }
        ]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Azure Group",
    "include": [
      {
        "azure_ad": {
          "id": "id-1",
          "identity_provider_id": "idp-789"
        }
      }
    ]
  }
}`,
		},
		{
			Name: "complex multi-selector",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Complex Group",
    "include": [
      {
        "email": ["admin@example.com", "manager@example.com"],
        "email_domain": ["example.com"],
        "ip": ["10.0.0.0/8"]
      }
    ],
    "exclude": [
      {
        "ip": ["10.0.1.0/24"],
        "geo": ["CN", "RU"]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "name": "Complex Group",
    "include": [
      {
        "email": {
          "email": "admin@example.com"
        }
      },
      {
        "email": {
          "email": "manager@example.com"
        }
      },
      {
        "ip": {
          "ip": "10.0.0.0/8"
        }
      },
      {
        "email_domain": {
          "domain": "example.com"
        }
      }
    ],
    "exclude": [
      {
        "ip": {
          "ip": "10.0.1.0/24"
        }
      },
      {
        "geo": {
          "country_code": "CN"
        }
      },
      {
        "geo": {
          "country_code": "RU"
        }
      }
    ]
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func TestResourceRename(t *testing.T) {
	migrator := NewV4ToV5Migrator().(*V4ToV5Migrator)

	oldName, newName := migrator.GetResourceRename()

	if oldName != "cloudflare_access_group" {
		t.Errorf("Expected old name to be 'cloudflare_access_group', got '%s'", oldName)
	}

	if newName != "cloudflare_zero_trust_access_group" {
		t.Errorf("Expected new name to be 'cloudflare_zero_trust_access_group', got '%s'", newName)
	}
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
