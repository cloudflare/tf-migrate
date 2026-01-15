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

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state with name",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"id": "my-namespace",
						"name": "test-namespace"
					}
				}`,
				Expected: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"id": "my-namespace",
						"name": "test-namespace",
						"namespace_name": "my-namespace"
					}
				}`,
			},
			{
				Name: "state with id (already present)",
				Input: `{
					"attributes": {
						"id": "0f2ac2fd364ea7d3f44bdc5a556c527e",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "my-dispatch-namespace"
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "0f2ac2fd364ea7d3f44bdc5a556c527e",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "my-dispatch-namespace",
						"namespace_name": "0f2ac2fd364ea7d3f44bdc5a556c527e"
					}
				}`,
			},
			{
				Name: "state without name (optional in v5)",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"id": "auto-generated-id"
					}
				}`,
				Expected: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"id": "auto-generated-id",
						"namespace_name": "auto-generated-id"
					}
				}`,
			},
			{
				Name: "state with special characters",
				Input: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"id": "test-namespace",
						"name": "test-namespace-2024"
					}
				}`,
				Expected: `{
					"attributes": {
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"id": "test-namespace",
						"name": "test-namespace-2024",
						"namespace_name": "test-namespace"
					}
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
