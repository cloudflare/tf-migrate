# Test both v4 resource name variants


# Plural form (preferred in v4)
resource "cloudflare_workers_secret" "plural_secret" {
  name        = "PLURAL_SECRET"
  script_name = "my-worker"
  secret_text = "plural-secret-value"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}


# With dispatch namespace
resource "cloudflare_workers_secret" "with_namespace" {
  name               = "NAMESPACE_SECRET"
  script_name        = "my-worker"
  dispatch_namespace = "my-namespace"
  secret_text        = "namespace-secret-value"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

# Singular form (deprecated in v4)
resource "cloudflare_workers_secret" "singular_secret" {
  name        = "SINGULAR_SECRET"
  script_name = "my-worker"
  secret_text = "singular-secret-value"
  # MIGRATION WARNING: MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

moved {
  from = cloudflare_worker_secret.singular_secret
  to   = cloudflare_workers_secret.singular_secret
}

# With account_id (should not generate warning)
resource "cloudflare_workers_secret" "with_account" {
  account_id  = var.account_id
  name        = "WITH_ACCOUNT"
  script_name = "my-worker"
  secret_text = "secret-with-account"
}

moved {
  from = cloudflare_worker_secret.with_account
  to   = cloudflare_workers_secret.with_account
}
