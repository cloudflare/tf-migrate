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
  account_id  = var.cloudflare_account_id
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  script_name = "cftftest-basic-worker"
  bindings = [
    {
      type = "secret_text"
      name = "API_KEY"
      text = "my-api-key-value"
    }
  ]
}


# ========================================
# Pattern 2: Multiple secrets for the same script
# ========================================

resource "cloudflare_workers_script" "multi_secret_worker" {
  account_id  = var.cloudflare_account_id
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Multi')); });"
  script_name = "cftftest-multi-secret-worker"
  bindings = [
    {
      type = "secret_text"
      name = "DB_PASSWORD"
      text = "super-secret-db-password"
      }, {
      type = "secret_text"
      name = "JWT_SECRET"
      text = "jwt-signing-key"
    }
  ]
}



# ========================================
# Pattern 3: Deprecated singular form (cloudflare_worker_secret)
# ========================================



# ========================================
# Pattern 4: Secret with script that has existing bindings
# ========================================

resource "cloudflare_workers_kv_namespace" "test_kv" {
  account_id = var.cloudflare_account_id
  title      = "cftftest-kv-for-secret-test"
}

resource "cloudflare_workers_script" "worker_with_bindings" {
  account_id = var.cloudflare_account_id
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Bindings')); });"


  script_name = "cftftest-worker-with-bindings"
  bindings = concat([
    {
      type         = "kv_namespace"
      name         = "MY_KV"
      namespace_id = cloudflare_workers_kv_namespace.test_kv.id
      }, {
      type = "plain_text"
      name = "ENV"
      text = "production"
    }
    ], [
    {
      type = "secret_text"
      name = "EXTRA_SECRET"
      text = "extra-secret-value"
    }
  ])
}


# ========================================
# Pattern 5: Secret referencing script via depends_on (literal name)
# ========================================

resource "cloudflare_workers_script" "literal_match_worker" {
  account_id  = var.cloudflare_account_id
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Literal')); });"
  script_name = "cftftest-literal-match"
  bindings = [
    {
      type = "secret_text"
      name = "LITERAL_SECRET"
      text = "literal-secret-value"
    }
  ]
}


removed {
  from = cloudflare_workers_secret.basic_secret
  lifecycle {
    destroy = false
  }
}

removed {
  from = cloudflare_workers_secret.db_password
  lifecycle {
    destroy = false
  }
}

removed {
  from = cloudflare_workers_secret.jwt_secret
  lifecycle {
    destroy = false
  }
}

resource "cloudflare_workers_script" "singular_worker" {
  account_id  = var.cloudflare_account_id
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Singular')); });"
  script_name = "cftftest-singular-worker"
  bindings = [
    {
      type = "secret_text"
      name = "SINGULAR_SECRET"
      text = "singular-secret-value"
    }
  ]
}

moved {
  from = cloudflare_worker_script.singular_worker
  to   = cloudflare_workers_script.singular_worker
}

removed {
  from = cloudflare_worker_secret.singular_secret
  lifecycle {
    destroy = false
  }
}

removed {
  from = cloudflare_workers_secret.binding_secret
  lifecycle {
    destroy = false
  }
}

removed {
  from = cloudflare_workers_secret.literal_match_secret
  lifecycle {
    destroy = false
  }
}
