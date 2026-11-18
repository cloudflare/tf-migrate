package zero_trust_access_group

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

// An empty selector list selects nothing in v4. It used to be kept as its own
// rule and rendered as `service_token = [ , ]`, which is not valid HCL.
func TestConfigTransformation_EmptySelectorLists(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "empty lists next to a non-empty selector are dropped",
			Input: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include {
    email         = ["a@example.com"]
    service_token = []
    group         = []
    ip            = []
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include = [
    {
      email = {
        email = "a@example.com"
      }
    },
  ]
}`,
		},
		{
			Name: "a rule with only empty lists contributes nothing",
			Input: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include {
    email = ["a@example.com"]
  }

  exclude {
    service_token = []
    email         = []
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_group" "test" {
  account_id = "abc123"
  name       = "Test Group"

  include = [
    {
      email = {
        email = "a@example.com"
      }
    },
  ]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
