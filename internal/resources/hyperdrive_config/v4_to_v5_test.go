package hyperdrive_config

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic hyperdrive config with required fields",
				Input: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}`,
				Expected: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}`,
			},
			{
				Name: "hyperdrive config with caching",
				Input: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  caching = {
    disabled               = false
    max_age                = 300
    stale_while_revalidate = 60
  }
}`,
				Expected: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  caching = {
    disabled               = false
    max_age                = 300
    stale_while_revalidate = 60
  }
}`,
			},
			{
				Name: "hyperdrive config with access origin",
				Input: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-hyperdrive-access"

  origin = {
    database             = "mydb"
    host                 = "db.internal.example.com"
    scheme               = "postgres"
    user                 = "admin"
    password             = "secret"
    access_client_id     = "client-id-abc.access"
    access_client_secret = "client-secret-xyz"
  }
}`,
				Expected: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-hyperdrive-access"

  origin = {
    database             = "mydb"
    host                 = "db.internal.example.com"
    scheme               = "postgres"
    user                 = "admin"
    password             = "secret"
    access_client_id     = "client-id-abc.access"
    access_client_secret = "client-secret-xyz"
  }
}`,
			},
			{
				Name: "hyperdrive config with variable references",
				Input: `
variable "account_id" {
  type = string
}

variable "db_password" {
  type      = string
  sensitive = true
}

resource "cloudflare_hyperdrive_config" "test" {
  account_id = var.account_id
  name       = "my-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = var.db_password
  }
}`,
				Expected: `
variable "account_id" {
  type = string
}

variable "db_password" {
  type      = string
  sensitive = true
}

resource "cloudflare_hyperdrive_config" "test" {
  account_id = var.account_id
  name       = "my-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = var.db_password
  }
}`,
			},
			{
				Name: "multiple hyperdrive configs",
				Input: `
resource "cloudflare_hyperdrive_config" "primary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "primary-hyperdrive"

  origin = {
    database = "primary_db"
    host     = "primary.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret1"
  }
}

resource "cloudflare_hyperdrive_config" "secondary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "secondary-hyperdrive"

  origin = {
    database = "secondary_db"
    host     = "secondary.example.com"
    port     = 3306
    scheme   = "mysql"
    user     = "root"
    password = "secret2"
  }
}`,
				Expected: `
resource "cloudflare_hyperdrive_config" "primary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "primary-hyperdrive"

  origin = {
    database = "primary_db"
    host     = "primary.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret1"
  }
}

resource "cloudflare_hyperdrive_config" "secondary" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "secondary-hyperdrive"

  origin = {
    database = "secondary_db"
    host     = "secondary.example.com"
    port     = 3306
    scheme   = "mysql"
    user     = "root"
    password = "secret2"
  }
}`,
			},
			{
				Name: "hyperdrive config with lifecycle",
				Input: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "lifecycle-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  lifecycle {
    prevent_destroy = true
  }
}`,
				Expected: `
resource "cloudflare_hyperdrive_config" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "lifecycle-hyperdrive"

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }

  lifecycle {
    prevent_destroy = true
  }
}`,
			},
			{
				Name: "hyperdrive config with count meta-argument",
				Input: `
variable "hyperdrive_names" {
  type = list(string)
}

resource "cloudflare_hyperdrive_config" "counted" {
  count      = length(var.hyperdrive_names)
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = var.hyperdrive_names[count.index]

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}`,
				Expected: `
variable "hyperdrive_names" {
  type = list(string)
}

resource "cloudflare_hyperdrive_config" "counted" {
  count      = length(var.hyperdrive_names)
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = var.hyperdrive_names[count.index]

  origin = {
    database = "mydb"
    host     = "db.example.com"
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}`,
			},
			{
				Name: "hyperdrive config with for_each",
				Input: `
variable "hyperdrives" {
  type = map(object({
    database = string
    host     = string
  }))
}

resource "cloudflare_hyperdrive_config" "each" {
  for_each   = var.hyperdrives
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = each.key

  origin = {
    database = each.value.database
    host     = each.value.host
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}`,
				Expected: `
variable "hyperdrives" {
  type = map(object({
    database = string
    host     = string
  }))
}

resource "cloudflare_hyperdrive_config" "each" {
  for_each   = var.hyperdrives
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = each.key

  origin = {
    database = each.value.database
    host     = each.value.host
    port     = 5432
    scheme   = "postgres"
    user     = "admin"
    password = "secret"
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
