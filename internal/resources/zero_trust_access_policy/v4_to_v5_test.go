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
			Name: "policy with deprecated fields removed (zone_id, precedence), session_duration preserved",
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
  account_id       = "account-123"
  name             = "Test Policy"
  decision         = "allow"
  session_duration = "24h"

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
			Name: "array expansion - device_posture email_list ip_list",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    device_posture = ["posture-1"]
    email_list     = ["email-list-1"]
    ip_list        = ["ip-list-1"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ device_posture = { integration_uid = "posture-1" } },
    { email_list = { id = "email-list-1" } },
  { ip_list = { id = "ip-list-1" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "nested block expansion - azure okta gsuite",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    azure {
      id                   = ["group-1", "group-2"]
      identity_provider_id = "idp-1"
    }
    okta {
      name                 = ["okta-1", "okta-2"]
      identity_provider_id = "idp-2"
    }
    gsuite {
      email                = ["team@example.com"]
      identity_provider_id = "idp-3"
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ azure_ad = { id = "group-1"
    identity_provider_id = "idp-1" } },
    { azure_ad = { id = "group-2"
      identity_provider_id = "idp-1" } },
    { okta = { name = "okta-1"
      identity_provider_id = "idp-2" } },
    { okta = { name = "okta-2"
      identity_provider_id = "idp-2" } },
  { gsuite = { email = "team@example.com"
  identity_provider_id = "idp-3" } }]
}

moved {
  from = cloudflare_access_policy.test
  to   = cloudflare_zero_trust_access_policy.test
}`,
		},
		{
			Name: "nested block expansion - saml oidc external_evaluation auth_context",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    saml {
      attribute_name       = "group"
      attribute_value      = "eng"
      identity_provider_id = "idp-1"
    }
    oidc {
      claim_name           = "roles"
      claim_value          = "admin"
      identity_provider_id = "idp-2"
    }
    external_evaluation {
      evaluate_url = "https://example.com/evaluate"
      keys_url     = "https://example.com/keys"
    }
    auth_context {
      id                   = "ctx-id"
      ac_id                = "ctx-ac-id"
      identity_provider_id = "idp-3"
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ saml = { attribute_name = "group"
    attribute_value = "eng"
    identity_provider_id = "idp-1" } },
    { oidc = { claim_name = "roles"
      claim_value = "admin"
      identity_provider_id = "idp-2" } },
    { external_evaluation = { evaluate_url = "https://example.com/evaluate"
      keys_url = "https://example.com/keys" } },
  { auth_context = { id = "ctx-id"
  ac_id = "ctx-ac-id"
  identity_provider_id = "idp-3" } }]
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
			Name: "common_names overflow array",
			Input: `
resource "cloudflare_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    common_names = ["device1.example.com", "device2.example.com"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "account-123"
  name       = "Test Policy"
  decision   = "allow"

  include = [{ common_name = { common_name = "device1.example.com" } },
  { common_name = { common_name = "device2.example.com" } }]
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

	// Should return a removed block and RemoveOriginal should be true
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Blocks) != 1 {
		t.Errorf("Expected 1 block (removed block), got %d", len(result.Blocks))
	}
	if !result.RemoveOriginal {
		t.Error("Expected RemoveOriginal to be true")
	}

	// Block should be a "removed" block
	if result.Blocks[0].Type() != "removed" {
		t.Errorf("Expected block type 'removed', got '%s'", result.Blocks[0].Type())
	}

	// The removed block should have the correct "from" attribute
	removedBody := result.Blocks[0].Body()
	fromAttr := removedBody.GetAttribute("from")
	if fromAttr == nil {
		t.Error("Expected removed block to have 'from' attribute")
	} else {
		fromExpr := string(fromAttr.Expr().BuildTokens(nil).Bytes())
		if !strings.Contains(fromExpr, "cloudflare_access_policy.test") {
			t.Errorf("Expected 'from' to reference cloudflare_access_policy.test, got: %s", fromExpr)
		}
	}

	// Should have a warning diagnostic
	if len(ctx.Diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(ctx.Diagnostics))
	}
	if ctx.Diagnostics[0].Severity != hcl.DiagWarning {
		t.Errorf("Expected DiagWarning severity, got %v", ctx.Diagnostics[0].Severity)
	}
	if ctx.Diagnostics[0].Summary != "Application-scoped access policy must be inlined" {
		t.Errorf("Unexpected diagnostic summary: %s", ctx.Diagnostics[0].Summary)
	}
	if !strings.Contains(ctx.Diagnostics[0].Detail, "application_id") {
		t.Errorf("Expected diagnostic detail to mention 'application_id', got: %s", ctx.Diagnostics[0].Detail)
	}
	if !strings.Contains(ctx.Diagnostics[0].Detail, "cloudflare_access_policy.test") {
		t.Errorf("Expected diagnostic detail to mention resource name, got: %s", ctx.Diagnostics[0].Detail)
	}
	// Check that inline policy example is included
	if !strings.Contains(ctx.Diagnostics[0].Detail, "Inline policy to add") {
		t.Errorf("Expected diagnostic detail to include inline policy example, got: %s", ctx.Diagnostics[0].Detail)
	}
}

// TestConfigTransformation_AlreadyV5Named tests the scenario from BUGS-2006:
// The user has already run tf-migrate once (or manually renamed resources),
// so the resource type is already "cloudflare_zero_trust_access_policy" (v5 name),
// but the nested include/exclude/require blocks are still in v4 block syntax.
// tf-migrate must still convert the blocks even when the resource name is already v5.
func TestConfigTransformation_AlreadyV5Named(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "v5-named resource with include block still in block syntax",
			Input: `
resource "cloudflare_zero_trust_access_policy" "sarav2_testing_sara_token" {
  account_id = "account-123"
  name       = "Sara Token Policy"
  decision   = "allow"

  include {
    email = ["sara@example.com"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "sarav2_testing_sara_token" {
  account_id = "account-123"
  name       = "Sara Token Policy"
  decision   = "allow"

  include = [{ email = { email = "sara@example.com" } }]
}`,
		},
		{
			Name: "v5-named resource with multiple condition blocks still in block syntax",
			Input: `
resource "cloudflare_zero_trust_access_policy" "assemblyline_valid_cloudflare_email" {
  account_id = "account-123"
  name       = "Assemblyline Policy"
  decision   = "allow"

  include {
    email_domain = ["cloudflare.com"]
  }

  require {
    certificate = true
  }

  exclude {
    geo = ["CN"]
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "assemblyline_valid_cloudflare_email" {
  account_id = "account-123"
  name       = "Assemblyline Policy"
  decision   = "allow"

  include = [{ email_domain = { domain = "cloudflare.com" } }]
  require = [{ certificate = {} }]
  exclude = [{ geo = { country_code = "CN" } }]
}`,
		},
		{
			Name: "v5-named resource with everyone include block",
			Input: `
resource "cloudflare_zero_trust_access_policy" "test_everyone" {
  account_id = "account-123"
  name       = "Everyone Policy"
  decision   = "allow"

  include {
    everyone = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test_everyone" {
  account_id = "account-123"
  name       = "Everyone Policy"
  decision   = "allow"

  include = [{ everyone = {} }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
