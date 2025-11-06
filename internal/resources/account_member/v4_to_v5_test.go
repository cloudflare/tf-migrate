package account_member

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "minimum config",
				Input: `
resource "cloudflare_account_member" "am_test" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "user@example.com"
  role_ids = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
				Expected: `
resource "cloudflare_account_member" "am_test" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email = "user@example.com"
  roles = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
			},
			{
				Name: "full config",
				Input: `
resource "cloudflare_account_member" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "user@example.com"
  status = "accepted"
  role_ids = [
    "68b329da9893e34099c7d8ad5cb9c940",
    "d784fa8b6d98d27699781bd9a7cf19f0"
  ]
}`,
				Expected: `
resource "cloudflare_account_member" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  status = "accepted"
  email = "user@example.com"
  roles = [
    "68b329da9893e34099c7d8ad5cb9c940",
    "d784fa8b6d98d27699781bd9a7cf19f0"
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "minimum state",
				Input: `{
					"attributes": {
						"id": "test-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"email_address": "user@example.com",
						"role_ids": ["68b329da9893e34099c7d8ad5cb9c940"]
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"email": "user@example.com",
						"roles": ["68b329da9893e34099c7d8ad5cb9c940"]
					}
				}`,
			},
			{
				Name: "full state",
				Input: `{
					"attributes": {
						"id": "test-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"email_address": "user@example.com",
						"status": "accepted",
						"role_ids": ["68b329da9893e34099c7d8ad5cb9c940", "d784fa8b6d98d27699781bd9a7cf19f0"]
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"email": "user@example.com",
						"status": "accepted",
						"roles": ["68b329da9893e34099c7d8ad5cb9c940", "d784fa8b6d98d27699781bd9a7cf19f0"]
					}
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
