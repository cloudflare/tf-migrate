package zero_trust_gateway_settings

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "minimal config with only required fields",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
		},
		// TODO: Add more test cases:
		// - Config with flat boolean flags
		// - Config with MaxItems:1 blocks
		// - Config with browser isolation
		// - Config with antivirus + notification settings
		// - Config with removed fields (logging, proxy, etc.)
		// - Config with all settings
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "minimal state",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "f037e56e89293a057740de681ac9abbe",
    "account_id": "f037e56e89293a057740de681ac9abbe"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "f037e56e89293a057740de681ac9abbe",
    "account_id": "f037e56e89293a057740de681ac9abbe"
  }
}`,
		},
		// TODO: Add more test cases:
		// - State with flat booleans
		// - State with MaxItems:1 arrays
		// - State with all settings
		// - State with removed fields
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}
