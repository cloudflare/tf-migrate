variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# ========================================
# Pattern 1: Single secret with parent script in same file
# ========================================

resource "cloudflare_workers_script" "basic_worker" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-basic-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
}

resource "cloudflare_workers_secret" "basic_secret" {
  account_id  = var.cloudflare_account_id
  script_name = cloudflare_workers_script.basic_worker.name
  name        = "API_KEY"
  secret_text = "my-api-key-value"
}

# ========================================
# Pattern 2: Multiple secrets for the same script
# ========================================

resource "cloudflare_workers_script" "multi_secret_worker" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-multi-secret-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Multi')); });"
}

resource "cloudflare_workers_secret" "db_password" {
  account_id  = var.cloudflare_account_id
  script_name = cloudflare_workers_script.multi_secret_worker.name
  name        = "DB_PASSWORD"
  secret_text = "super-secret-db-password"
}

resource "cloudflare_workers_secret" "jwt_secret" {
  account_id  = var.cloudflare_account_id
  script_name = cloudflare_workers_script.multi_secret_worker.name
  name        = "JWT_SECRET"
  secret_text = "jwt-signing-key"
}

# ========================================
# Pattern 3: Deprecated singular form (cloudflare_worker_secret)
# ========================================

resource "cloudflare_worker_script" "singular_worker" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-singular-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Singular')); });"
}

resource "cloudflare_worker_secret" "singular_secret" {
  account_id  = var.cloudflare_account_id
  script_name = cloudflare_worker_script.singular_worker.name
  name        = "SINGULAR_SECRET"
  secret_text = "singular-secret-value"
}

# ========================================
# Pattern 4: Secret with script that has existing bindings
# ========================================

resource "cloudflare_workers_kv_namespace" "test_kv" {
  account_id = var.cloudflare_account_id
  title      = "cftftest-kv-for-secret-test"
}

resource "cloudflare_workers_script" "worker_with_bindings" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-worker-with-bindings"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Bindings')); });"

  kv_namespace_binding {
    name         = "MY_KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv.id
  }

  plain_text_binding {
    name = "ENV"
    text = "production"
  }
}

resource "cloudflare_workers_secret" "binding_secret" {
  account_id  = var.cloudflare_account_id
  script_name = cloudflare_workers_script.worker_with_bindings.name
  name        = "EXTRA_SECRET"
  secret_text = "extra-secret-value"
}

# ========================================
# Pattern 5: Secret referencing script via depends_on (literal name)
# ========================================

resource "cloudflare_workers_script" "literal_match_worker" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-literal-match"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Literal')); });"
}

resource "cloudflare_workers_secret" "literal_match_secret" {
  account_id  = var.cloudflare_account_id
  script_name = "cftftest-literal-match"
  name        = "LITERAL_SECRET"
  secret_text = "literal-secret-value"
  depends_on  = [cloudflare_workers_script.literal_match_worker]
}
