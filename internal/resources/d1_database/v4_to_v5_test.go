package d1_database

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic d1 database with required fields",
				Input: `
resource "cloudflare_d1_database" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-database"
}`,
				Expected: `
resource "cloudflare_d1_database" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-database"
}`,
			},
			{
				Name: "d1 database with variable references",
				Input: `
variable "account_id" {
  type = string
}

resource "cloudflare_d1_database" "test" {
  account_id = var.account_id
  name       = "my-database"
}`,
				Expected: `
variable "account_id" {
  type = string
}

resource "cloudflare_d1_database" "test" {
  account_id = var.account_id
  name       = "my-database"
}`,
			},
			{
				Name: "multiple d1 databases",
				Input: `
resource "cloudflare_d1_database" "db1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "database-one"
}

resource "cloudflare_d1_database" "db2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "database-two"
}`,
				Expected: `
resource "cloudflare_d1_database" "db1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "database-one"
}

resource "cloudflare_d1_database" "db2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "database-two"
}`,
			},
			{
				Name: "d1 database with string interpolation",
				Input: `
resource "cloudflare_d1_database" "test" {
  account_id = var.account_id
  name       = "${var.environment}-database"
}`,
				Expected: `
resource "cloudflare_d1_database" "test" {
  account_id = var.account_id
  name       = "${var.environment}-database"
}`,
			},
			{
				Name: "d1 database with lifecycle",
				Input: `
resource "cloudflare_d1_database" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "lifecycle-database"

  lifecycle {
    prevent_destroy = true
  }
}`,
				Expected: `
resource "cloudflare_d1_database" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "lifecycle-database"

  lifecycle {
    prevent_destroy = true
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
