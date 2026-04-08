# Test both v4 resource name variants

# Singular form (deprecated in v4)
resource "cloudflare_worker_secret" "singular_secret" {
  name        = "SINGULAR_SECRET"
  script_name = "my-worker"
  secret      = "singular-secret-value"
}

# Plural form (preferred in v4)
resource "cloudflare_workers_secret" "plural_secret" {
  name        = "PLURAL_SECRET"
  script_name = "my-worker"
  secret      = "plural-secret-value"
}

# With account_id (should not generate warning)
resource "cloudflare_worker_secret" "with_account" {
  account_id  = var.account_id
  name        = "WITH_ACCOUNT"
  script_name = "my-worker"
  secret      = "secret-with-account"
}

# With dispatch namespace
resource "cloudflare_workers_secret" "with_namespace" {
  name               = "NAMESPACE_SECRET"
  script_name        = "my-worker"
  secret             = "namespace-secret-value"
  dispatch_namespace = "my-namespace"
}
