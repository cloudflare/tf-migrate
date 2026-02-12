package r2_bucket

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic r2 bucket with required fields",
				Input: `
resource "cloudflare_r2_bucket" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
}`,
				Expected: `
resource "cloudflare_r2_bucket" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
}`,
			},
			{
				Name: "r2 bucket with location",
				Input: `
resource "cloudflare_r2_bucket" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
  location   = "WNAM"
}`,
				Expected: `
resource "cloudflare_r2_bucket" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
  location   = "WNAM"
}`,
			},
			{
				Name: "r2 bucket with lowercase location",
				Input: `
resource "cloudflare_r2_bucket" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
  location   = "wnam"
}`,
				Expected: `
resource "cloudflare_r2_bucket" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
  location   = "wnam"
}`,
			},
			{
				Name: "multiple r2 buckets",
				Input: `
resource "cloudflare_r2_bucket" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "bucket-1"
}

resource "cloudflare_r2_bucket" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "bucket-2"
  location   = "EEUR"
}`,
				Expected: `
resource "cloudflare_r2_bucket" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "bucket-1"
}

resource "cloudflare_r2_bucket" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "bucket-2"
  location   = "EEUR"
}`,
			},
			{
				Name: "r2 bucket with variable reference",
				Input: `
variable "account_id" {
  type = string
}

resource "cloudflare_r2_bucket" "test" {
  account_id = var.account_id
  name       = "test-bucket"
}`,
				Expected: `
variable "account_id" {
  type = string
}

resource "cloudflare_r2_bucket" "test" {
  account_id = var.account_id
  name       = "test-bucket"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}
