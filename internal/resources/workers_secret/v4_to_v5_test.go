package workers_secret

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "singular_v4_name",
				Input: `
resource "cloudflare_worker_secret" "my_secret" {
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}`,
				Expected: `resource "cloudflare_workers_secret" "my_secret" {
  name         = "MY_SECRET"
  script_name  = "my-worker"
  secret_text  = "super-secret-value"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

moved {
  from = cloudflare_worker_secret.my_secret
  to   = cloudflare_workers_secret.my_secret
}`,
			},
			{
				Name: "plural_v4_name",
				Input: `
resource "cloudflare_workers_secret" "my_secret" {
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}`,
				Expected: `resource "cloudflare_workers_secret" "my_secret" {
  name         = "MY_SECRET"
  script_name  = "my-worker"
  secret_text  = "super-secret-value"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}`,
			},
			{
				Name: "with_account_id",
				Input: `
resource "cloudflare_worker_secret" "my_secret" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}`,
				Expected: `resource "cloudflare_workers_secret" "my_secret" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  name         = "MY_SECRET"
  script_name  = "my-worker"
  secret_text  = "super-secret-value"
}

moved {
  from = cloudflare_worker_secret.my_secret
  to   = cloudflare_workers_secret.my_secret
}`,
			},
			{
				Name: "with_dispatch_namespace",
				Input: `
resource "cloudflare_worker_secret" "my_secret" {
  name               = "MY_SECRET"
  script_name        = "my-worker"
  secret             = "super-secret-value"
  dispatch_namespace = "my-namespace"
}`,
				Expected: `resource "cloudflare_workers_secret" "my_secret" {
  name                = "MY_SECRET"
  script_name         = "my-worker"
  dispatch_namespace  = "my-namespace"
  secret_text         = "super-secret-value"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

moved {
  from = cloudflare_worker_secret.my_secret
  to   = cloudflare_workers_secret.my_secret
}`,
			},
			{
				Name: "multiple_secrets",
				Input: `
resource "cloudflare_worker_secret" "secret1" {
  name        = "SECRET_ONE"
  script_name = "worker-one"
  secret      = "first-secret"
}

resource "cloudflare_workers_secret" "secret2" {
  name        = "SECRET_TWO"
  script_name = "worker-two"
  secret      = "second-secret"
}`,
				// Note: Block ordering is non-deterministic, test individually
				Expected: `resource "cloudflare_workers_secret" "secret2" {
  name         = "SECRET_TWO"
  script_name  = "worker-two"
  secret_text  = "second-secret"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

resource "cloudflare_workers_secret" "secret1" {
  name         = "SECRET_ONE"
  script_name  = "worker-one"
  secret_text  = "first-secret"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

moved {
  from = cloudflare_worker_secret.secret1
  to   = cloudflare_workers_secret.secret1
}`,
			},
			{
				Name: "plural_with_account_id_no_warning",
				Input: `
resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}`,
				Expected: `resource "cloudflare_workers_secret" "my_secret" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  name         = "MY_SECRET"
  script_name  = "my-worker"
  secret_text  = "super-secret-value"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
