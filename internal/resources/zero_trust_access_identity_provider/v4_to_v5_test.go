package zero_trust_access_identity_provider

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
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
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformations(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "unwrap config array to object",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "GitHub OAuth",
    "type": "github",
    "config": [
      {
        "client_id": "github-client-id",
        "client_secret": "CONCEALED"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "GitHub OAuth",
    "type": "github",
    "config": {
      "client_id": "github-client-id",
      "client_secret": "CONCEALED"
    }
  }
}`,
		},
		{
			Name: "remove deprecated api_token from config",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "Test Provider",
    "type": "github",
    "config": [
      {
        "client_id": "client-id",
        "api_token": "deprecated-token"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "Test Provider",
    "type": "github",
    "config": {
      "client_id": "client-id"
    }
  }
}`,
		},
		{
			Name: "rename idp_public_cert to idp_public_certs and wrap in array",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "SAML Provider",
    "type": "saml",
    "config": [
      {
        "issuer_url": "https://saml.example.com",
        "sso_target_url": "https://saml.example.com/sso",
        "idp_public_cert": "MIIDpDCCAoygAwIBAgIGAV..."
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "SAML Provider",
    "type": "saml",
    "config": {
      "issuer_url": "https://saml.example.com",
      "sso_target_url": "https://saml.example.com/sso",
      "idp_public_certs": ["MIIDpDCCAoygAwIBAgIGAV..."]
    }
  }
}`,
		},
		{
			Name: "unwrap scim_config and remove deprecated fields",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "Azure AD",
    "type": "azureAD",
    "config": [
      {
        "client_id": "azure-client"
      }
    ],
    "scim_config": [
      {
        "enabled": true,
        "secret": "scim-secret-value",
        "user_deprovision": true,
        "group_member_deprovision": true
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "Azure AD",
    "type": "azureAD",
    "config": {
      "client_id": "azure-client"
    },
    "scim_config": {
      "enabled": true,
      "secret": "scim-secret-value",
      "user_deprovision": true
    }
  }
}`,
		},
		{
			Name: "missing config - add empty config object",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "One-Time PIN",
    "type": "onetimepin"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "One-Time PIN",
    "type": "onetimepin",
    "config": {}
  }
}`,
		},
		{
			Name: "null config - add empty config object",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "One-Time PIN",
    "type": "onetimepin",
    "config": null
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "One-Time PIN",
    "type": "onetimepin",
    "config": {}
  }
}`,
		},
		{
			Name: "empty config array - set to empty object",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "One-Time PIN",
    "type": "onetimepin",
    "config": []
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "One-Time PIN",
    "type": "onetimepin",
    "config": {}
  }
}`,
		},
		{
			Name: "complex state with all transformations",
			Input: `{
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "Azure AD with SCIM",
    "type": "azureAD",
    "config": [
      {
        "client_id": "azure-client-id",
        "client_secret": "CONCEALED",
        "directory_id": "dir-123",
        "conditional_access_enabled": true,
        "support_groups": true,
        "api_token": "deprecated-token"
      }
    ],
    "scim_config": [
      {
        "enabled": true,
        "secret": "scim-secret-value",
        "user_deprovision": true,
        "seat_deprovision": false,
        "group_member_deprovision": true,
        "identity_update_behavior": "automatic"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "test-id",
    "account_id": "account123",
    "name": "Azure AD with SCIM",
    "type": "azureAD",
    "config": {
      "client_id": "azure-client-id",
      "client_secret": "CONCEALED",
      "directory_id": "dir-123",
      "conditional_access_enabled": true,
      "support_groups": true
    },
    "scim_config": {
      "enabled": true,
      "secret": "scim-secret-value",
      "user_deprovision": true,
      "seat_deprovision": false,
      "identity_update_behavior": "automatic"
    }
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
