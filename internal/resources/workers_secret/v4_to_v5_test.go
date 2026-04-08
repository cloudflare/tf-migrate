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
				Expected: `resource "cloudflare_worker_secret" "my_secret" {
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
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
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
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
				Expected: `resource "cloudflare_worker_secret" "my_secret" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
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
				Expected: `resource "cloudflare_worker_secret" "my_secret" {
  name               = "MY_SECRET"
  script_name        = "my-worker"
  secret             = "super-secret-value"
  dispatch_namespace = "my-namespace"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
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
				// Note: Blocks are processed in order they appear in input
				Expected: `resource "cloudflare_worker_secret" "secret1" {
  name        = "SECRET_ONE"
  script_name = "worker-one"
  secret      = "first-secret"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}

resource "cloudflare_workers_secret" "secret2" {
  name        = "SECRET_TWO"
  script_name = "worker-two"
  secret      = "second-secret"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}`,
			},
			{
				Name: "plural_with_account_id",
				Input: `
resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}`,
				Expected: `resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
