package zero_trust_access_policy

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation_Simple(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic policy with simple fields",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    everyone = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "policy with deprecated fields removed",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id     = "account-123"
  zone_id        = "zone-456"
  application_id = "app-789"
  precedence     = 1
  name           = "Test Policy"
  decision       = "allow"

  include {
    everyone = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "policy with approval_group renamed",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    everyone = true
  }

  approval_group {
    approvals_needed = 2
    email_addresses  = ["admin@example.com"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  approval_groups = [{
    approvals_needed = 2
    email_addresses  = ["admin@example.com"]
  }]
  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "policy with connection_rules converted to attribute syntax",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "SSH Policy"
  decision   = "allow"

  include {
    everyone = true
  }

  connection_rules {
    ssh {
      usernames         = ["admin", "deploy"]
      allow_email_alias = true
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "SSH Policy"
  decision   = "allow"

  connection_rules = {
    ssh = {
      usernames         = ["admin", "deploy"]
      allow_email_alias = true
    }
  }
  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestConfigTransformation_Conditions(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "boolean condition - everyone true",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    everyone = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ everyone = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "boolean condition - any_valid_service_token true",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    any_valid_service_token = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ any_valid_service_token = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "array expansion - email",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    email = ["alice@example.com", "bob@example.com"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ email = { email = "alice@example.com" } },
  { email = { email = "bob@example.com" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "array expansion - group",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    group = ["group-id-1", "group-id-2"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ group = { id = "group-id-1" } },
  { group = { id = "group-id-2" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "mixed conditions - boolean and array",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    everyone = true
    email    = ["admin@example.com"]
    group    = ["admins"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ email = { email = "admin@example.com" } },
    { group = { id = "admins" } },
  { everyone = {} }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "github with teams expansion",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    github = [{
      name                 = "my-org"
      teams                = ["engineering", "devops"]
      identity_provider_id = "provider-123"
    }]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ github_organization = { name = "my-org"
    team = "engineering"
    identity_provider_id = "provider-123" } },
    { github_organization = { name = "my-org"
      team = "devops"
  identity_provider_id = "provider-123" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "exclude and require conditions",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    email = ["allowed@example.com"]
  }

  exclude {
    geo = ["CN", "RU"]
  }

  require {
    certificate = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ email = { email = "allowed@example.com"  }  }]
  exclude = [{ geo = { country_code = "CN"  }  },
  { geo = { country_code = "RU"  }  }]
  require = [{ certificate = {}  }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "ip and email_domain expansion",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    ip           = ["192.168.1.0/24", "10.0.0.0/8"]
    email_domain = ["example.com", "company.org"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ ip = { ip = "192.168.1.0/24" } },
    { ip = { ip = "10.0.0.0/8" } },
    { email_domain = { domain = "example.com" } },
  { email_domain = { domain = "company.org" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "common_name single string",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    common_name = "device1.example.com"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ common_name = { common_name = "device1.example.com" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "auth_method single string",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    auth_method = "swk"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ auth_method = { auth_method = "swk" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "login_method array",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    login_method = ["otp", "warp"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ login_method = { id = "otp" } },
  { login_method = { id = "warp" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
