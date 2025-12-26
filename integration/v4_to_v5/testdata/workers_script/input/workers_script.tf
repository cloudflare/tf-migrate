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
# Dependencies - KV Namespaces
# ========================================

resource "cloudflare_workers_kv_namespace" "test_kv" {
  account_id = var.cloudflare_account_id
  title      = "cftftest-kv-namespace-for-workers-script"
}

resource "cloudflare_workers_kv_namespace" "test_kv_multiple" {
  count      = 2
  account_id = var.cloudflare_account_id
  title      = "cftftest-kv-${count.index}"
}

# ========================================
# Dependencies - D1 Databases
# ========================================
# TODO: Uncomment when d1_database v4→v5 migration is created
# D1 bindings also require ES module format which adds complexity

# resource "cloudflare_d1_database" "test_d1" {
#   account_id = var.cloudflare_account_id
#   name       = "cftftest-d1-database"
# }

# ========================================
# Dependencies - Queues
# ========================================
# TODO: Uncomment when queue v4→v5 migration is created
# The v4 schema uses 'name', v5 uses 'queue_name'
# Need to create: internal/resources/queue/v4_to_v5.go

# resource "cloudflare_queue" "test_queue" {
#   account_id = var.cloudflare_account_id
#   name       = "cftftest-queue"
# }

# ========================================
# Dependencies - R2 Buckets
# ========================================

resource "cloudflare_r2_bucket" "test_r2" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-r2-bucket"
}

# ========================================
# Local Values
# ========================================

locals {
  worker_prefix = "cftftest"
  common_content = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"
  script_names = ["worker-1", "worker-2", "worker-3"]

  # Complex expression
  full_worker_name = "${local.worker_prefix}-main"
}

# ========================================
# Variables for Testing
# ========================================

variable "worker_content" {
  type    = string
  default = "addEventListener('fetch', event => { event.respondWith(new Response('Test')); });"
}

variable "enable_analytics" {
  type    = bool
  default = true
}

# ========================================
# Pattern 1: Simple worker with plain_text_binding
# ========================================

resource "cloudflare_workers_script" "plain_text" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-plain-text-worker"
  content    = local.common_content

  plain_text_binding {
    name = "MY_VAR"
    text = "my-value"
  }
}

# ========================================
# Pattern 2: Worker with secret_text_binding
# ========================================

resource "cloudflare_workers_script" "secret_text" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-secret-text-worker"
  content    = local.common_content

  secret_text_binding {
    name = "API_KEY"
    text = "secret-api-key-value"
  }
}

# ========================================
# Pattern 3: Worker with kv_namespace_binding
# ========================================

resource "cloudflare_workers_script" "kv_namespace" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-kv-worker"
  content    = local.common_content

  kv_namespace_binding {
    name         = "MY_KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv.id
  }
}

# ========================================
# Pattern 5: Worker with r2_bucket_binding
# ========================================

resource "cloudflare_workers_script" "r2_bucket" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-r2-worker"
  content    = local.common_content

  r2_bucket_binding {
    name        = "MY_BUCKET"
    bucket_name = cloudflare_r2_bucket.test_r2.name
  }
}

# ========================================
# Pattern 7: Worker with analytics_engine_binding
# ========================================

resource "cloudflare_workers_script" "analytics_engine" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-analytics-worker"
  content    = local.common_content

  analytics_engine_binding {
    name    = "MY_ANALYTICS"
    dataset = "analytics_dataset"
  }
}

# ========================================
# Pattern 8: Worker with queue_binding (binding → name, queue → queue_name)
# ========================================
# TODO: Uncomment when queue v4→v5 migration is created

# resource "cloudflare_workers_script" "queue" {
#   account_id = var.cloudflare_account_id
#   name       = "cftftest-queue-worker"
#   content    = local.common_content
#
#   queue_binding {
#     binding = "MY_QUEUE"
#     queue   = cloudflare_queue.test_queue.name
#   }
# }

# ========================================
# Pattern 9: Worker with multiple bindings (order preservation test)
# ========================================

resource "cloudflare_workers_script" "multiple_bindings" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-multiple-bindings-worker"
  content    = local.common_content

  plain_text_binding {
    name = "VAR_ONE"
    text = "value-one"
  }

  secret_text_binding {
    name = "SECRET_ONE"
    text = "secret-value-one"
  }

  kv_namespace_binding {
    name         = "KV_ONE"
    namespace_id = cloudflare_workers_kv_namespace.test_kv.id
  }
}

# ========================================
# Pattern 10: Worker with module = false (becomes body_part)
# ========================================

resource "cloudflare_workers_script" "module_false" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-module-false-worker"
  content    = local.common_content
  module     = false
}

# ========================================
# Pattern 15: Worker with placement block (becomes object)
# ========================================

resource "cloudflare_workers_script" "placement" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-placement-worker"
  content    = local.common_content

  placement {
    mode = "smart"
  }
}

# ========================================
# Pattern 16: Worker with tags (should be removed)
# ========================================

resource "cloudflare_workers_script" "tags" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-tags-worker"
  content    = local.common_content
  tags       = ["test", "migration"]

  plain_text_binding {
    name = "VAR"
    text = "value"
  }
}

# ========================================
# Pattern 17: Comprehensive worker with all transformations
# ========================================

resource "cloudflare_workers_script" "comprehensive" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-comprehensive-worker"
  content    = local.common_content
  module     = false
  tags       = ["comprehensive", "test"]

  plain_text_binding {
    name = "PLAIN_VAR"
    text = "plain-value"
  }

  secret_text_binding {
    name = "SECRET_VAR"
    text = "secret-value"
  }

  kv_namespace_binding {
    name         = "KV_NS"
    namespace_id = cloudflare_workers_kv_namespace.test_kv.id
  }

  placement {
    mode = "smart"
  }
}

# ========================================
# Pattern 18: for_each with map
# ========================================

variable "worker_configs" {
  type = map(object({
    content = string
    var_value = string
  }))
  default = {
    "worker-a" = {
      content   = "addEventListener('fetch', e => { e.respondWith(new Response('A')); });"
      var_value = "value-a"
    }
    "worker-b" = {
      content   = "addEventListener('fetch', e => { e.respondWith(new Response('B')); });"
      var_value = "value-b"
    }
    "worker-c" = {
      content   = "addEventListener('fetch', e => { e.respondWith(new Response('C')); });"
      var_value = "value-c"
    }
  }
}

resource "cloudflare_workers_script" "for_each_map" {
  for_each = var.worker_configs

  account_id = var.cloudflare_account_id
  name       = "cftftest-${each.key}"
  content    = each.value.content

  plain_text_binding {
    name = "CONFIG_VAR"
    text = each.value.var_value
  }
}

# ========================================
# Pattern 19: for_each with set
# ========================================

variable "simple_workers" {
  type = set(string)
  default = ["simple-1", "simple-2", "simple-3"]
}

resource "cloudflare_workers_script" "for_each_set" {
  for_each = var.simple_workers

  account_id = var.cloudflare_account_id
  name       = "cftftest-${each.value}"
  content    = "addEventListener('fetch', e => { e.respondWith(new Response('${each.value}')); });"

  plain_text_binding {
    name = "WORKER_NAME"
    text = each.value
  }
}

# ========================================
# Pattern 20: Count-based resources
# ========================================

variable "replica_count" {
  type    = number
  default = 3
}

resource "cloudflare_workers_script" "count_based" {
  count = var.replica_count

  account_id = var.cloudflare_account_id
  name       = "cftftest-replica-${count.index}"
  content    = local.common_content

  plain_text_binding {
    name = "REPLICA_ID"
    text = tostring(count.index)
  }

  kv_namespace_binding {
    name         = "KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv_multiple[count.index % 2].id
  }
}

# ========================================
# Pattern 21: Conditional resource creation
# ========================================

resource "cloudflare_workers_script" "conditional" {
  count = var.enable_analytics ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "cftftest-conditional-worker"
  content    = local.common_content

  analytics_engine_binding {
    name    = "ANALYTICS"
    dataset = "conditional_dataset"
  }
}

# ========================================
# Pattern 22: Resource with lifecycle meta-arguments
# ========================================

resource "cloudflare_workers_script" "lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-lifecycle-worker"
  content    = local.common_content

  plain_text_binding {
    name = "LIFECYCLE_VAR"
    text = "protected"
  }

  lifecycle {
    prevent_destroy       = false
    create_before_destroy = true
  }
}

# ========================================
# Pattern 23: Using terraform expressions in bindings
# ========================================

resource "cloudflare_workers_script" "conditional_binding" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-conditional-binding-worker"
  content    = var.enable_analytics ? local.common_content : var.worker_content

  plain_text_binding {
    name = "ENABLED"
    text = var.enable_analytics ? "true" : "false"
  }

  kv_namespace_binding {
    name         = "KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv.id
  }
}

# ========================================
# Pattern 24: Cross-resource references with dependencies
# ========================================

resource "cloudflare_workers_script" "dependent" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-dependent-worker"
  content    = local.common_content

  kv_namespace_binding {
    name         = "PRIMARY_KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv.id
  }

  kv_namespace_binding {
    name         = "SECONDARY_KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv_multiple[0].id
  }

  r2_bucket_binding {
    name        = "STORAGE"
    bucket_name = cloudflare_r2_bucket.test_r2.name
  }

  # queue_binding commented out - requires queue v4→v5 migration
  # queue_binding {
  #   binding = "JOBS"
  #   queue   = cloudflare_queue.test_queue.name
  # }
}

# ========================================
# Pattern 25: Worker referencing local values extensively
# ========================================

resource "cloudflare_workers_script" "local_refs" {
  account_id = var.cloudflare_account_id
  name       = local.full_worker_name
  content    = local.common_content

  plain_text_binding {
    name = "PREFIX"
    text = local.worker_prefix
  }
}

# ========================================
# Pattern 26: Singular resource name (cloudflare_worker_script)
# Tests resource rename: cloudflare_worker_script → cloudflare_workers_script
# ========================================

resource "cloudflare_worker_script" "singular_name" {
  account_id = var.cloudflare_account_id
  name       = "cftftest-singular-resource"
  content    = local.common_content

  plain_text_binding {
    name = "SINGULAR"
    text = "true"
  }
}

# ========================================
# Pattern 27: Complex for_each with list transformation
# ========================================

variable "environment_configs" {
  type = list(object({
    env       = string
    kv_index  = number
    use_queue = bool
  }))
  default = [
    {
      env       = "dev"
      kv_index  = 0
      use_queue = true
    },
    {
      env       = "staging"
      kv_index  = 1
      use_queue = true
    },
    {
      env       = "prod"
      kv_index  = 0
      use_queue = false
    }
  ]
}

resource "cloudflare_workers_script" "environment" {
  for_each = { for cfg in var.environment_configs : cfg.env => cfg }

  account_id = var.cloudflare_account_id
  name       = "cftftest-${each.key}-worker"
  content    = local.common_content

  plain_text_binding {
    name = "ENVIRONMENT"
    text = each.value.env
  }

  kv_namespace_binding {
    name         = "ENV_KV"
    namespace_id = cloudflare_workers_kv_namespace.test_kv_multiple[each.value.kv_index].id
  }

  # dynamic queue_binding commented out - requires queue v4→v5 migration
  # dynamic "queue_binding" {
  #   for_each = each.value.use_queue ? [1] : []
  #   content {
  #     binding = "ENV_QUEUE"
  #     queue   = cloudflare_queue.test_queue.name
  #   }
  # }
}
