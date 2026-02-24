package workers_for_platforms_dispatch_namespace

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic workers for platforms dispatch namespace",
				Input: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-namespace"
}`,
				Expected: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-namespace"
}`,
			},
			{
				Name: "namespace without explicit name (optional in v5)",
				Input: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
				Expected: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			},
			{
				Name: "namespace with special characters in name",
				Input: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-namespace-2024"
}`,
				Expected: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-namespace-2024"
}`,
			},
			{
				Name: "multiple namespaces",
				Input: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-1"
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-2"
}`,
				Expected: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-1"
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-2"
}`,
			},
			{
				Name: "namespace with variable reference",
				Input: `
variable "cloudflare_account_id" {
  type = string
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = var.cloudflare_account_id
  name       = "test-namespace"
}`,
				Expected: `
variable "cloudflare_account_id" {
  type = string
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = var.cloudflare_account_id
  name       = "test-namespace"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("ConfigTransformation_DeprecatedName", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "deprecated resource name gets renamed with moved block",
				Input: `
resource "cloudflare_workers_for_platforms_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-namespace"
}`,
				Expected: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-namespace"
}

moved {
  from = cloudflare_workers_for_platforms_namespace.test
  to   = cloudflare_workers_for_platforms_dispatch_namespace.test
}`,
			},
			{
				Name: "multiple deprecated resources get renamed with moved blocks",
				Input: `
resource "cloudflare_workers_for_platforms_namespace" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-1"
}

resource "cloudflare_workers_for_platforms_namespace" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-2"
}`,
				Expected: `
resource "cloudflare_workers_for_platforms_dispatch_namespace" "test1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-1"
}

moved {
  from = cloudflare_workers_for_platforms_namespace.test1
  to   = cloudflare_workers_for_platforms_dispatch_namespace.test1
}

resource "cloudflare_workers_for_platforms_dispatch_namespace" "test2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "namespace-2"
}

moved {
  from = cloudflare_workers_for_platforms_namespace.test2
  to   = cloudflare_workers_for_platforms_dispatch_namespace.test2
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

}
