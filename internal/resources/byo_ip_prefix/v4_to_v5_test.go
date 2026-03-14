package byo_ip_prefix

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic BYO IP prefix - removes prefix_id and advertisement",
				Input: `
resource "cloudflare_byo_ip_prefix" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  prefix_id     = "prefix-abc123"
  description   = "My BYO IP prefix"
  advertisement = "on"
}`,
				Expected: `resource "cloudflare_byo_ip_prefix" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  description = "My BYO IP prefix"
}`,
			},
			{
				Name: "minimal BYO IP prefix - only required fields",
				Input: `
resource "cloudflare_byo_ip_prefix" "minimal" {
  account_id = var.cloudflare_account_id
  prefix_id  = "prefix-xyz789"
}`,
				Expected: `resource "cloudflare_byo_ip_prefix" "minimal" {
  account_id = var.cloudflare_account_id
}`,
			},
			{
				Name: "BYO IP prefix with description only",
				Input: `
resource "cloudflare_byo_ip_prefix" "test" {
  account_id  = "test-account-id"
  prefix_id   = "prefix-test"
  description = "Test prefix"
}`,
				Expected: `resource "cloudflare_byo_ip_prefix" "test" {
  account_id  = "test-account-id"
  description = "Test prefix"
}`,
			},
			{
				Name: "multiple BYO IP prefixes in one file",
				Input: `
resource "cloudflare_byo_ip_prefix" "first" {
  account_id    = var.account_id
  prefix_id     = "prefix-first"
  description   = "First prefix"
  advertisement = "on"
}

resource "cloudflare_byo_ip_prefix" "second" {
  account_id    = var.account_id
  prefix_id     = "prefix-second"
  description   = "Second prefix"
  advertisement = "off"
}`,
				Expected: `resource "cloudflare_byo_ip_prefix" "first" {
  account_id  = var.account_id
  description = "First prefix"
}

resource "cloudflare_byo_ip_prefix" "second" {
  account_id  = var.account_id
  description = "Second prefix"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}
