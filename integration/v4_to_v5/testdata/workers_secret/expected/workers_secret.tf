# Test both v4 resource name variants

# Singular form (deprecated in v4)
resource "cloudflare_worker_secret" "singular_secret" {
  name        = "SINGULAR_SECRET"
  script_name = "my-worker"
  secret      = "singular-secret-value"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}

# Plural form (preferred in v4)
resource "cloudflare_workers_secret" "plural_secret" {
  name        = "PLURAL_SECRET"
  script_name = "my-worker"
  secret      = "plural-secret-value"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}

# With account_id (should not generate warning)
resource "cloudflare_worker_secret" "with_account" {
  account_id  = var.account_id
  name        = "WITH_ACCOUNT"
  script_name = "my-worker"
  secret      = "secret-with-account"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}

# With dispatch namespace
resource "cloudflare_workers_secret" "with_namespace" {
  name               = "NAMESPACE_SECRET"
  script_name        = "my-worker"
  secret             = "namespace-secret-value"
  dispatch_namespace = "my-namespace"
  # MIGRATION WARNING: cloudflare_workers_secret removed in v5. Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret
}
