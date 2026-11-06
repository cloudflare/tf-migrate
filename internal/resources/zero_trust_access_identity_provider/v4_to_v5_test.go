package zero_trust_access_identity_provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestConfigTransformations(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "resource rename and basic config block to attribute",
			Input: `
resource "cloudflare_access_identity_provider" "test" {
  account_id = "account123"
  name       = "GitHub OAuth"
  type       = "github"
  config {
    client_id     = "github-client-id"
    client_secret = "github-secret"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "GitHub OAuth"
  type       = "github"
  config = {
    client_id     = "github-client-id"
    client_secret = "github-secret"
  }
}

moved {
  from = cloudflare_access_identity_provider.test
  to   = cloudflare_zero_trust_access_identity_provider.test
}`,
		},
		{
			Name: "current v5 name - no rename, no moved block",
			Input: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "GitHub OAuth"
  type       = "github"
  config {
    client_id     = "github-client-id"
    client_secret = "github-secret"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "GitHub OAuth"
  type       = "github"
  config = {
    client_id     = "github-client-id"
    client_secret = "github-secret"
  }
}`,
		},
		{
			Name: "remove deprecated api_token from config",
			Input: `
resource "cloudflare_access_identity_provider" "test" {
  account_id = "account123"
  name       = "Test Provider"
  type       = "github"
  config {
    client_id = "client-id"
    api_token = "deprecated-token"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "Test Provider"
  type       = "github"
  config = {
    client_id = "client-id"
  }
}

moved {
  from = cloudflare_access_identity_provider.test
  to   = cloudflare_zero_trust_access_identity_provider.test
}`,
		},
		{
			Name: "rename idp_public_cert to idp_public_certs and wrap in array",
			Input: `
resource "cloudflare_access_identity_provider" "test" {
  account_id = "account123"
  name       = "SAML Provider"
  type       = "saml"
  config {
    issuer_url      = "https://saml.example.com"
    sso_target_url  = "https://saml.example.com/sso"
    idp_public_cert = "MIIDpDCCAoygAwIBAgIGAV..."
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "SAML Provider"
  type       = "saml"
  config = {
    issuer_url       = "https://saml.example.com"
    sso_target_url   = "https://saml.example.com/sso"
    idp_public_certs = ["MIIDpDCCAoygAwIBAgIGAV..."]
  }
}

moved {
  from = cloudflare_access_identity_provider.test
  to   = cloudflare_zero_trust_access_identity_provider.test
}`,
		},
		{
			Name: "scim_config block to attribute and remove deprecated fields",
			Input: `
resource "cloudflare_access_identity_provider" "test" {
  account_id = "account123"
  name       = "Azure AD"
  type       = "azureAD"
  config {
    client_id = "azure-client"
  }
  scim_config {
    enabled                  = true
    user_deprovision         = true
    group_member_deprovision = true
    secret                   = "user-provided-secret"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "Azure AD"
  type       = "azureAD"
  config = {
    client_id = "azure-client"
  }
  scim_config = {
    enabled          = true
    user_deprovision = true
  }
}

moved {
  from = cloudflare_access_identity_provider.test
  to   = cloudflare_zero_trust_access_identity_provider.test
}`,
		},
		{
			Name: "missing config block - add empty config",
			Input: `
resource "cloudflare_access_identity_provider" "test" {
  account_id = "account123"
  name       = "One-Time PIN"
  type       = "onetimepin"
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "One-Time PIN"
  type       = "onetimepin"
  config = {}
}

moved {
  from = cloudflare_access_identity_provider.test
  to   = cloudflare_zero_trust_access_identity_provider.test
}`,
		},
		{
			Name: "complex config with all transformations",
			Input: `
resource "cloudflare_access_identity_provider" "test" {
  account_id = "account123"
  name       = "Azure AD with SCIM"
  type       = "azureAD"
  config {
    client_id                  = "azure-client-id"
    client_secret              = "azure-secret"
    directory_id               = "dir-123"
    conditional_access_enabled = true
    support_groups             = true
    api_token                  = "deprecated-token"
  }
  scim_config {
    enabled                   = true
    secret                    = "user-secret"
    user_deprovision          = true
    seat_deprovision          = false
    group_member_deprovision  = true
    identity_update_behavior  = "automatic"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_identity_provider" "test" {
  account_id = "account123"
  name       = "Azure AD with SCIM"
  type       = "azureAD"
  config = {
    client_id                  = "azure-client-id"
    client_secret              = "azure-secret"
    directory_id               = "dir-123"
    conditional_access_enabled = true
    support_groups             = true
  }
  scim_config = {
    enabled                  = true
    user_deprovision         = true
    seat_deprovision         = false
    identity_update_behavior = "automatic"
  }
}

moved {
  from = cloudflare_access_identity_provider.test
  to   = cloudflare_zero_trust_access_identity_provider.test
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// v4 accepted repeated config blocks but the provider read config.0 only, so
// later blocks never reached the API. Keeping the last would activate settings
// that were previously inert and change how the IdP authenticates.
func TestMultipleConfigBlocksKeepsFirst(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	input := `resource "cloudflare_access_identity_provider" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Okta"
  type       = "okta"

  config {
    client_id     = "live-client"
    client_secret = "live-secret"
  }

  config {
    client_id     = "inert-client"
    client_secret = "inert-secret"
  }
}`

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)

	ctx := &transform.Context{
		Content:       []byte(input),
		Filename:      "test.tf",
		Diagnostics:   make(hcl.Diagnostics, 0),
		Metadata:      make(map[string]interface{}),
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	var block *hclwrite.Block
	for _, b := range file.Body().Blocks() {
		if b.Type() == "resource" {
			block = b
			break
		}
	}
	require.NotNil(t, block)

	result, err := migrator.TransformConfig(ctx, block)
	require.NoError(t, err)
	require.NotNil(t, result)

	var out string
	for _, b := range result.Blocks {
		out += string(hclwrite.Format(b.BuildTokens(nil).Bytes()))
	}

	assert.Contains(t, out, "live-client",
		"the first config block is the one v4 used and must be kept, got:\n%s", out)
	assert.NotContains(t, out, "inert-client",
		"the second config block was never sent to the API in v4 and must not be "+
			"promoted into the v5 config, got:\n%s", out)

	var warned bool
	for _, d := range ctx.Diagnostics {
		if d.Severity == hcl.DiagWarning && strings.Contains(d.Summary, "config block(s) dropped") {
			warned = true
		}
	}
	assert.True(t, warned, "dropping a config block must be reported, got: %v", ctx.Diagnostics)
}
