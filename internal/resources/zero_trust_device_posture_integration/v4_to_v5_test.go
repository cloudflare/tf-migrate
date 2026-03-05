package zero_trust_device_posture_integration

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "minimal resource - deprecated name rename",
			Input: `
resource "cloudflare_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  interval   = "24h"
  config {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  interval   = "24h"
  config = {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}

moved {
  from = cloudflare_device_posture_integration.test
  to   = cloudflare_zero_trust_device_posture_integration.test
}
`,
		},
		{
			Name: "minimal resource - current name no rename",
			Input: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  interval   = "24h"
  config {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  interval   = "24h"
  config = {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}

moved {
  from = cloudflare_device_posture_integration.test
  to   = cloudflare_zero_trust_device_posture_integration.test
}
`,
		},
		{
			Name: "resource without interval - adds default",
			Input: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  config {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  interval   = "24h"
  config = {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}

moved {
  from = cloudflare_device_posture_integration.test
  to   = cloudflare_zero_trust_device_posture_integration.test
}
`,
		},
		{
			Name: "resource with identifier - removes it",
			Input: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  identifier = "legacy-id-123"
  interval   = "24h"
  config {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Test Integration"
  type       = "workspace_one"
  interval   = "24h"
  config = {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}

moved {
  from = cloudflare_device_posture_integration.test
  to   = cloudflare_zero_trust_device_posture_integration.test
}
`,
		},
		{
			Name: "full resource with all config fields",
			Input: `
resource "cloudflare_zero_trust_device_posture_integration" "full" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Full Integration"
  type       = "crowdstrike_s2s"
  interval   = "12h"
  config {
    api_url               = "https://api.crowdstrike.com"
    auth_url              = "https://auth.crowdstrike.com"
    client_id             = "crowdstrike-client"
    client_secret         = "crowdstrike-secret"
    customer_id           = "customer-acme"
    client_key            = "crowdstrike-key"
    access_client_id      = "cf-access-id"
    access_client_secret  = "cf-access-secret"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "full" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Full Integration"
  type       = "crowdstrike_s2s"
  interval   = "12h"
  config = {
    api_url               = "https://api.crowdstrike.com"
    auth_url              = "https://auth.crowdstrike.com"
    client_id             = "crowdstrike-client"
    client_secret         = "crowdstrike-secret"
    customer_id           = "customer-acme"
    client_key            = "crowdstrike-key"
    access_client_id      = "cf-access-id"
    access_client_secret  = "cf-access-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.full
  to   = cloudflare_zero_trust_device_posture_integration.full
}
`,
		},
		{
			Name: "multiple resources in one file",
			Input: `
resource "cloudflare_device_posture_integration" "ws1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Workspace One"
  type       = "workspace_one"
  interval   = "24h"
  config {
    api_url       = "https://as123.awmdm.com/api"
    auth_url      = "https://na.uemauth.vmwservices.com/connect/token"
    client_id     = "ws1-client"
    client_secret = "ws1-secret"
  }
}

resource "cloudflare_zero_trust_device_posture_integration" "crowdstrike" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "CrowdStrike"
  type       = "crowdstrike_s2s"
  config {
    client_id     = "cs-client"
    client_secret = "cs-secret"
    customer_id   = "cs-customer"
  }
}

resource "cloudflare_device_posture_integration" "uptycs" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Uptycs"
  type       = "uptycs"
  identifier = "should-be-removed"
  interval   = "1h"
  config {
    api_url    = "https://uptycs-api.example.com"
    client_id  = "uptycs-client"
    client_key = "uptycs-key"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "ws1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Workspace One"
  type       = "workspace_one"
  interval   = "24h"
  config = {
    api_url       = "https://as123.awmdm.com/api"
    auth_url      = "https://na.uemauth.vmwservices.com/connect/token"
    client_id     = "ws1-client"
    client_secret = "ws1-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.ws1
  to   = cloudflare_zero_trust_device_posture_integration.ws1
}

resource "cloudflare_zero_trust_device_posture_integration" "crowdstrike" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "CrowdStrike"
  type       = "crowdstrike_s2s"
  interval   = "24h"
  config = {
    client_id     = "cs-client"
    client_secret = "cs-secret"
    customer_id   = "cs-customer"
  }
}

moved {
  from = cloudflare_device_posture_integration.crowdstrike
  to   = cloudflare_zero_trust_device_posture_integration.crowdstrike
}

resource "cloudflare_zero_trust_device_posture_integration" "uptycs" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Uptycs"
  type       = "uptycs"
  interval   = "1h"
  config = {
    api_url    = "https://uptycs-api.example.com"
    client_id  = "uptycs-client"
    client_key = "uptycs-key"
  }
}

moved {
  from = cloudflare_device_posture_integration.uptycs
  to   = cloudflare_zero_trust_device_posture_integration.uptycs
}
`,
		},
		{
			Name: "different integration types",
			Input: `
resource "cloudflare_zero_trust_device_posture_integration" "intune" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Microsoft Intune"
  type       = "intune"
  interval   = "24h"
  config {
    client_id     = "intune-client"
    client_secret = "intune-secret"
  }
}

resource "cloudflare_zero_trust_device_posture_integration" "kolide" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Kolide"
  type       = "kolide"
  interval   = "24h"
  config {
    client_id             = "kolide-client"
    client_secret         = "kolide-secret"
    access_client_id      = "cf-access-id"
    access_client_secret  = "cf-access-secret"
  }
}

resource "cloudflare_zero_trust_device_posture_integration" "sentinelone" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SentinelOne"
  type       = "sentinelone_s2s"
  interval   = "12h"
  config {
    api_url    = "https://sentinelone.example.com"
    client_id  = "s1-client"
    client_key = "s1-key"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "intune" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Microsoft Intune"
  type       = "intune"
  interval   = "24h"
  config = {
    client_id     = "intune-client"
    client_secret = "intune-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.intune
  to   = cloudflare_zero_trust_device_posture_integration.intune
}

resource "cloudflare_zero_trust_device_posture_integration" "kolide" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Kolide"
  type       = "kolide"
  interval   = "24h"
  config = {
    client_id             = "kolide-client"
    client_secret         = "kolide-secret"
    access_client_id      = "cf-access-id"
    access_client_secret  = "cf-access-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.kolide
  to   = cloudflare_zero_trust_device_posture_integration.kolide
}

resource "cloudflare_zero_trust_device_posture_integration" "sentinelone" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SentinelOne"
  type       = "sentinelone_s2s"
  interval   = "12h"
  config = {
    api_url    = "https://sentinelone.example.com"
    client_id  = "s1-client"
    client_key = "s1-key"
  }
}

moved {
  from = cloudflare_device_posture_integration.sentinelone
  to   = cloudflare_zero_trust_device_posture_integration.sentinelone
}
`,
		},
		{
			Name: "combined edge case - deprecated name, no interval, with identifier",
			Input: `
resource "cloudflare_device_posture_integration" "edge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Edge Case Integration"
  type       = "custom_s2s"
  identifier = "remove-me"
  config {
    api_url               = "https://custom-api.example.com"
    access_client_id      = "custom-access-id"
    access_client_secret  = "custom-access-secret"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "edge" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Edge Case Integration"
  type       = "custom_s2s"
  interval   = "24h"
  config = {
    api_url               = "https://custom-api.example.com"
    access_client_id      = "custom-access-id"
    access_client_secret  = "custom-access-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.edge
  to   = cloudflare_zero_trust_device_posture_integration.edge
}
`,
		},
		{
			Name: "resource with comments preserved",
			Input: `
# Main workspace one integration
resource "cloudflare_zero_trust_device_posture_integration" "commented" {
  # Account configuration
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Commented Integration"
  type       = "workspace_one"
  interval   = "24h"

  # Integration configuration
  config {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}
`,
			Expected: `
# Main workspace one integration
resource "cloudflare_zero_trust_device_posture_integration" "commented" {
  # Account configuration
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Commented Integration"
  type       = "workspace_one"
  interval   = "24h"

  config = {
    api_url       = "https://api.example.com"
    auth_url      = "https://auth.example.com"
    client_id     = "client-123"
    client_secret = "secret-456"
  }
}

moved {
  from = cloudflare_device_posture_integration.commented
  to   = cloudflare_zero_trust_device_posture_integration.commented
}
`,
		},
		{
			Name: "resource with variable references",
			Input: `
resource "cloudflare_zero_trust_device_posture_integration" "vars" {
  account_id = var.cloudflare_account_id
  name       = var.integration_name
  type       = "workspace_one"
  interval   = var.check_interval
  config {
    api_url       = var.ws1_api_url
    auth_url      = var.ws1_auth_url
    client_id     = var.ws1_client_id
    client_secret = var.ws1_client_secret
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_integration" "vars" {
  account_id = var.cloudflare_account_id
  name       = var.integration_name
  type       = "workspace_one"
  interval   = var.check_interval
  config = {
    api_url       = var.ws1_api_url
    auth_url      = var.ws1_auth_url
    client_id     = var.ws1_client_id
    client_secret = var.ws1_client_secret
  }
}

moved {
  from = cloudflare_device_posture_integration.vars
  to   = cloudflare_zero_trust_device_posture_integration.vars
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestV4ToV5TransformationState_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}
