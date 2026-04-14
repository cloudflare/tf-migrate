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
				Name: "minimum config with status accepted added",
				Input: `
resource "cloudflare_account_member" "am_test" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "user@example.com"
  role_ids = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
				Expected: `
resource "cloudflare_account_member" "am_test" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  status        = "accepted"
  email         = "user@example.com"
  roles         = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
			},
			{
				Name: "full config with existing status preserved",
				Input: `
resource "cloudflare_account_member" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "user@example.com"
  status        = "accepted"
  role_ids = [
    "68b329da9893e34099c7d8ad5cb9c940",
    "d784fa8b6d98d27699781bd9a7cf19f0"
  ]
}`,
				Expected: `
resource "cloudflare_account_member" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  status        = "accepted"
  email         = "user@example.com"
  roles         = [
    "68b329da9893e34099c7d8ad5cb9c940",
    "d784fa8b6d98d27699781bd9a7cf19f0"
  ]
}`,
			},
			{
				Name: "service account member with status added",
				Input: `
resource "cloudflare_account_member" "svc_terraform" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "svc-terraform@example.com"
  role_ids      = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
				Expected: `
resource "cloudflare_account_member" "svc_terraform" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  status        = "accepted"
  email         = "svc-terraform@example.com"
  roles         = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
			},
			{
				Name: "member with pending status preserved",
				Input: `
resource "cloudflare_account_member" "pending_member" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  email_address = "pending@example.com"
  status        = "pending"
  role_ids      = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
				Expected: `
resource "cloudflare_account_member" "pending_member" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  status        = "pending"
  email         = "pending@example.com"
  roles         = ["68b329da9893e34099c7d8ad5cb9c940"]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}
