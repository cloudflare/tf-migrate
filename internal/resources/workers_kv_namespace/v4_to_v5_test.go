package workers_kv_namespace

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic workers KV namespace",
				Input: `
resource "cloudflare_workers_kv_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "test-namespace"
}`,
				Expected: `
resource "cloudflare_workers_kv_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "test-namespace"
}`,
			},
			{
				Name: "namespace with special characters in title",
				Input: `
resource "cloudflare_workers_kv_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "test-namespace-2024"
}`,
				Expected: `
resource "cloudflare_workers_kv_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "test-namespace-2024"
}`,
			},
			{
				Name: "multiple namespaces",
				Input: `
resource "cloudflare_workers_kv_namespace" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "namespace-1"
}

resource "cloudflare_workers_kv_namespace" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "namespace-2"
}`,
				Expected: `
resource "cloudflare_workers_kv_namespace" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "namespace-1"
}

resource "cloudflare_workers_kv_namespace" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "namespace-2"
}`,
			},
			{
				Name: "namespace with variable reference",
				Input: `
variable "account_id" {
  type = string
}

resource "cloudflare_workers_kv_namespace" "test" {
  account_id = var.account_id
  title      = "test-namespace"
}`,
				Expected: `
variable "account_id" {
  type = string
}

resource "cloudflare_workers_kv_namespace" "test" {
  account_id = var.account_id
  title      = "test-namespace"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
