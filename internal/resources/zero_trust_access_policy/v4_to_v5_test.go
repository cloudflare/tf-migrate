package zero_trust_access_policy

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
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
			Name: "policy with deprecated fields removed (zone_id, precedence, session_duration)",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id       = "account-123"
  zone_id          = "zone-456"
  precedence       = 1
  session_duration = "24h"
  name             = "Test Policy"
  decision         = "allow"

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

func TestConfigTransformation_ApplicationScopedPolicySkipped(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	input := `
resource "cloudflare_access_policy" "test" {
  account_id     = "account-123"
  application_id = "app-789"
  name           = "Test Policy"
  decision       = "allow"
  precedence     = 1

  include {
    everyone = true
  }
}`

	// Parse the input HCL
	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("Failed to parse input HCL: %v", diags)
	}

	ctx := &transform.Context{
		Content:     []byte(input),
		Filename:    "test.tf",
		CFGFile:     file,
		Diagnostics: hcl.Diagnostics{},
	}

	// Get the resource block
	body := file.Body()
	var resourceBlock *hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "resource" {
			resourceBlock = block
			break
		}
	}
	if resourceBlock == nil {
		t.Fatal("Resource block not found")
	}

	// Transform
	result, err := migrator.TransformConfig(ctx, resourceBlock)
	if err != nil {
		t.Fatalf("TransformConfig returned error: %v", err)
	}

	// Should return the original block unchanged (no transformation, no moved block)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Blocks) != 1 {
		t.Errorf("Expected 1 block (original only), got %d", len(result.Blocks))
	}
	if result.RemoveOriginal {
		t.Error("Expected RemoveOriginal to be false")
	}

	// Block should still be cloudflare_access_policy (not renamed)
	labels := result.Blocks[0].Labels()
	if len(labels) < 1 || labels[0] != "cloudflare_access_policy" {
		t.Errorf("Expected resource type to remain 'cloudflare_access_policy', got %v", labels)
	}

	// Should have a warning diagnostic
	if len(ctx.Diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(ctx.Diagnostics))
	}
	if ctx.Diagnostics[0].Severity != hcl.DiagWarning {
		t.Errorf("Expected DiagWarning severity, got %v", ctx.Diagnostics[0].Severity)
	}
	if ctx.Diagnostics[0].Summary != "Application-scoped access policy cannot be automatically migrated" {
		t.Errorf("Unexpected diagnostic summary: %s", ctx.Diagnostics[0].Summary)
	}
	if !strings.Contains(ctx.Diagnostics[0].Detail, "application_id") {
		t.Errorf("Expected diagnostic detail to mention 'application_id', got: %s", ctx.Diagnostics[0].Detail)
	}
	if !strings.Contains(ctx.Diagnostics[0].Detail, "cloudflare_access_policy.test") {
		t.Errorf("Expected diagnostic detail to mention resource name, got: %s", ctx.Diagnostics[0].Detail)
	}
}
