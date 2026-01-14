# Workers Script Migration Guide (v4 → v5)

This guide explains how `cloudflare_worker_script` / `cloudflare_workers_script` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_worker_script` (singular) | `cloudflare_workers_script` (plural) | Renamed |
| Alt resource name | `cloudflare_workers_script` | `cloudflare_workers_script` | No change |
| `name` | `name = "..."` | `script_name = "..."` | Field renamed |
| `module` | Boolean (true/false) | `main_module` or `body_part` | Type change + split |
| `tags` | Array | Removed | Deprecated field |
| Bindings | 10 separate block types | Unified `bindings` array | Major consolidation |
| `placement` | Block | Attribute object | Syntax change |
| `dispatch_namespace` | Attribute | Binding type | Converted to binding |


---

## Migration Examples

### Example 1: Basic Worker Script

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "basic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = file("worker.js")
  module     = false
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "basic" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "my-worker"
  content     = file("worker.js")
  body_part   = "worker.js"
}
```

**What Changed:**
- `name` → `script_name`
- `module = false` → `body_part = "worker.js"`

---

### Example 2: Module Worker (ES Modules)

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "module_worker" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "module-worker"
  content    = file("worker.mjs")
  module     = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "module_worker" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "module-worker"
  content     = file("worker.mjs")
  main_module = "worker.js"
}
```

**What Changed:**
- `module = true` → `main_module = "worker.js"`

---

### Example 3: With Environment Variables

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "with_env" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "env-worker"
  content    = file("worker.js")
  module     = false

  plain_text_binding {
    name = "API_URL"
    text = "https://api.example.com"
  }

  plain_text_binding {
    name = "ENV"
    text = "production"
  }

  secret_text_binding {
    name = "API_KEY"
    text = var.api_key
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "with_env" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "env-worker"
  content     = file("worker.js")
  body_part   = "worker.js"

  bindings = [
    {
      type = "plain_text"
      name = "API_URL"
      text = "https://api.example.com"
    },
    {
      type = "plain_text"
      name = "ENV"
      text = "production"
    },
    {
      type = "secret_text"
      name = "API_KEY"
      text = var.api_key
    }
  ]
}
```

**What Changed:**
- Multiple `plain_text_binding` blocks → unified `bindings` array
- `secret_text_binding` block → binding with `type = "secret_text"`
- Binding order preserved

---

### Example 4: With KV, R2, and Service Bindings

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "comprehensive-worker"
  content    = file("worker.js")
  module     = false

  kv_namespace_binding {
    name         = "MY_KV"
    namespace_id = cloudflare_workers_kv_namespace.example.id
  }

  r2_bucket_binding {
    name        = "MY_BUCKET"
    bucket_name = "my-bucket"
  }

  service_binding {
    name        = "AUTH_SERVICE"
    service     = "auth-worker"
    environment = "production"
  }

  d1_database_binding {
    name        = "DB"
    database_id = cloudflare_d1_database.example.id
  }

  queue_binding {
    binding = "MY_QUEUE"
    queue   = "production-queue"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "comprehensive" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "comprehensive-worker"
  content     = file("worker.js")
  body_part   = "worker.js"

  bindings = [
    {
      type         = "kv_namespace"
      name         = "MY_KV"
      namespace_id = cloudflare_workers_kv_namespace.example.id
    },
    {
      type        = "r2_bucket"
      name        = "MY_BUCKET"
      bucket_name = "my-bucket"
    },
    {
      type        = "service"
      name        = "AUTH_SERVICE"
      service     = "auth-worker"
      environment = "production"
    },
    {
      type = "d1"
      name = "DB"
      id   = cloudflare_d1_database.example.id
    },
    {
      type       = "queue"
      name       = "MY_QUEUE"
      queue_name = "production-queue"
    }
  ]
}
```

**What Changed:**
- 5 different binding block types → single `bindings` array
- Each binding gets a `type` field
- `queue_binding.binding` → `name`
- `queue_binding.queue` → `queue_name`
- `d1_database_binding.database_id` → `id`

---

### Example 5: WebAssembly and Analytics Bindings

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "wasm" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "wasm-worker"
  content    = file("worker.js")
  module     = false

  webassembly_binding {
    name   = "WASM"
    module = cloudflare_workers_script.wasm_module.id
  }

  analytics_engine_binding {
    name    = "ANALYTICS"
    dataset = "my_analytics"
  }

  hyperdrive_config_binding {
    binding = "HYPERDRIVE"
    id      = cloudflare_hyperdrive_config.example.id
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "wasm" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "wasm-worker"
  content     = file("worker.js")
  body_part   = "worker.js"

  bindings = [
    {
      type = "wasm_module"
      name = "WASM"
      part = cloudflare_workers_script.wasm_module.id
    },
    {
      type    = "analytics_engine"
      name    = "ANALYTICS"
      dataset = "my_analytics"
    },
    {
      type = "hyperdrive"
      name = "HYPERDRIVE"
      id   = cloudflare_hyperdrive_config.example.id
    }
  ]
}
```

**What Changed:**
- `webassembly_binding.module` → binding with `part` field
- `hyperdrive_config_binding.binding` → `name`

---

### Example 6: With Placement and Dispatch Namespace

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "dispatched" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name               = "dispatch-worker"
  content            = file("worker.js")
  module             = false
  dispatch_namespace = "my-dispatch-namespace"

  placement {
    mode = "smart"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "dispatched" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "dispatch-worker"
  content     = file("worker.js")
  body_part   = "worker.js"

  bindings = [{
    type      = "dispatch_namespace"
    namespace = "my-dispatch-namespace"
  }]

  placement = {
    mode = "smart"
  }
}
```

**What Changed:**
- `dispatch_namespace` attribute → binding with `type = "dispatch_namespace"`
- `placement { }` block → `placement = { }` attribute

---

### Example 7: With Tags (Removed)

**v4 Configuration:**
```hcl
resource "cloudflare_workers_script" "tagged" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "tagged-worker"
  content    = file("worker.js")
  module     = false
  tags       = ["production", "api"]  # ⚠️ Removed in v5
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_workers_script" "tagged" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "tagged-worker"
  content     = file("worker.js")
  body_part   = "worker.js"
  # tags removed - no longer supported
}
```

**What Changed:**
- `tags` field completely removed

---

