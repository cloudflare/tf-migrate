package zero_trust_access_policy

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

// An empty selector list selects nothing in v4. It used to be kept as its own
// rule, { service_token = [] }, which v5 rejects with "Incorrect attribute
// value type".
func TestConfigTransformation_EmptySelectorLists(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "empty lists are dropped, a rule with only empty lists contributes nothing",
			Input: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "abc123"
  name       = "Test Policy"
  decision   = "allow"

  include {
    email         = ["a@example.com"]
    service_token = []
    group         = []
  }

  exclude {
    ip = []
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_policy" "test" {
  account_id = "abc123"
  name       = "Test Policy"
  decision   = "allow"
  include    = [{ email = { email = "a@example.com" } }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
