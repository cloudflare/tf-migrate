# Pages Project Migration Guide (v4 → v5)

This guide explains how `cloudflare_pages_project` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_pages_project` | `cloudflare_pages_project` | No change |
| `build_config` | Block | Attribute object | Syntax change |
| `source` | Block with nested `config` | Attribute with nested object | Syntax change |
| `deployment_configs` | Block with nested blocks | Attribute with nested objects | Syntax change |
| `environment_variables` + `secrets` | Two separate maps | Merged into `env_vars` | Field merging |
| `service_binding` | Array of objects | `services` map | Structure change |
| Map fields (kv, d1, r2, durable_objects) | Simple string values | Wrapped in objects | Value wrapping |
| New computed fields | - | `canonical_deployment`, `framework`, `latest_deployment`, etc. | Fields added |
| `production_deployment_enabled` | In source.config | `production_deployments_enabled` | Field renamed |


---

## Migration Examples

### Example 1: Basic Pages Project

**v4 Configuration:**
```hcl
resource "cloudflare_pages_project" "basic" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "my-pages-app"
  production_branch = "main"

  build_config {
    build_command   = "npm run build"
    destination_dir = "dist"
  }

  source {
    type = "github"
    config {
      owner                         = "myorg"
      repo_name                     = "my-repo"
      production_branch             = "main"
      pr_comments_enabled           = true
      deployments_enabled           = true
      production_deployment_enabled = true
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_pages_project" "basic" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "my-pages-app"
  production_branch = "main"

  build_config = {
    build_command   = "npm run build"
    destination_dir = "dist"
  }

  source = {
    type = "github"
    config = {
      owner                          = "myorg"
      repo_name                      = "my-repo"
      production_branch              = "main"
      pr_comments_enabled            = true
      deployments_enabled            = true
      production_deployments_enabled = true
    }
  }
}
```

**What Changed:**
- All blocks → attribute objects
- `production_deployment_enabled` → `production_deployments_enabled`

---

### Example 2: Environment Variables and Secrets

**v4 Configuration:**
```hcl
resource "cloudflare_pages_project" "with_env" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "env-app"
  production_branch = "main"

  build_config {
    build_command = "npm run build"
  }

  deployment_configs {
    production {
      environment_variables = {
        "NODE_ENV"    = "production"
        "API_URL"     = "https://api.example.com"
        "FEATURE_FLAG" = "true"
      }

      secrets = {
        "API_KEY"    = "secret-key-123"
        "DB_PASSWORD" = "secret-pass-456"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_pages_project" "with_env" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "env-app"
  production_branch = "main"

  build_config = {
    build_command = "npm run build"
  }

  deployment_configs = {
    production = {
      env_vars = {
        "NODE_ENV" = {
          type  = "plain_text"
          value = "production"
        }
        "API_URL" = {
          type  = "plain_text"
          value = "https://api.example.com"
        }
        "FEATURE_FLAG" = {
          type  = "plain_text"
          value = "true"
        }
        "API_KEY" = {
          type  = "secret_text"
          value = "secret-key-123"
        }
        "DB_PASSWORD" = {
          type  = "secret_text"
          value = "secret-pass-456"
        }
      }
    }
  }
}
```

**What Changed:**
- `environment_variables` and `secrets` → unified `env_vars`
- Simple string values → objects with `type` and `value`
- `environment_variables` → `type = "plain_text"`
- `secrets` → `type = "secret_text"`

---

### Example 3: Service Bindings

**v4 Configuration:**
```hcl
resource "cloudflare_pages_project" "with_services" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "service-app"
  production_branch = "main"

  deployment_configs {
    production {
      service_binding {
        name        = "AUTH_SERVICE"
        service     = "auth-worker"
        environment = "production"
      }

      service_binding {
        name        = "DATA_SERVICE"
        service     = "data-worker"
        environment = "production"
        entrypoint  = "DataHandler"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_pages_project" "with_services" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "service-app"
  production_branch = "main"

  deployment_configs = {
    production = {
      services = {
        "AUTH_SERVICE" = {
          service     = "auth-worker"
          environment = "production"
        }
        "DATA_SERVICE" = {
          service     = "data-worker"
          environment = "production"
          entrypoint  = "DataHandler"
        }
      }
    }
  }
}
```

**What Changed:**
- `service_binding` array → `services` map
- Binding `name` becomes map key
- Other fields become object values

---

### Example 4: Bindings (KV, D1, R2, Durable Objects)

**v4 Configuration:**
```hcl
resource "cloudflare_pages_project" "with_bindings" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "bindings-app"
  production_branch = "main"

  deployment_configs {
    production {
      kv_namespaces = {
        "KV_BINDING" = "kv-namespace-id-123"
      }

      d1_databases = {
        "DB_BINDING" = "d1-database-id-456"
      }

      r2_buckets = {
        "R2_BINDING" = "my-bucket"
      }

      durable_object_namespaces = {
        "DO_BINDING" = "do-namespace-id-789"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_pages_project" "with_bindings" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "bindings-app"
  production_branch = "main"

  deployment_configs = {
    production = {
      kv_namespaces = {
        "KV_BINDING" = {
          namespace_id = "kv-namespace-id-123"
        }
      }

      d1_databases = {
        "DB_BINDING" = {
          id = "d1-database-id-456"
        }
      }

      r2_buckets = {
        "R2_BINDING" = {
          name = "my-bucket"
        }
      }

      durable_object_namespaces = {
        "DO_BINDING" = {
          namespace_id = "do-namespace-id-789"
        }
      }
    }
  }
}
```

**What Changed:**
- Simple string values wrapped in objects
- `kv_namespaces`: string → `{namespace_id: string}`
- `d1_databases`: string → `{id: string}`
- `r2_buckets`: string → `{name: string}`
- `durable_object_namespaces`: string → `{namespace_id: string}`

---

### Example 5: Preview and Production Configs

**v4 Configuration:**
```hcl
resource "cloudflare_pages_project" "full_config" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "full-app"
  production_branch = "main"

  deployment_configs {
    preview {
      compatibility_date  = "2024-01-01"
      compatibility_flags = ["nodejs_compat"]
      usage_model        = "bundled"

      environment_variables = {
        "ENV" = "preview"
      }
    }

    production {
      compatibility_date  = "2024-01-01"
      compatibility_flags = ["nodejs_compat"]
      usage_model        = "bundled"

      environment_variables = {
        "ENV" = "production"
      }

      placement {
        mode = "smart"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_pages_project" "full_config" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "full-app"
  production_branch = "main"

  deployment_configs = {
    preview = {
      compatibility_date  = "2024-01-01"
      compatibility_flags = ["nodejs_compat"]
      usage_model         = "bundled"

      env_vars = {
        "ENV" = {
          type  = "plain_text"
          value = "preview"
        }
      }
    }

    production = {
      compatibility_date  = "2024-01-01"
      compatibility_flags = ["nodejs_compat"]
      usage_model         = "bundled"

      env_vars = {
        "ENV" = {
          type  = "plain_text"
          value = "production"
        }
      }

      placement = {
        mode = "smart"
      }
    }
  }
}
```

**What Changed:**
- All blocks → attributes
- `environment_variables` → `env_vars` with type/value structure
- Nested `placement` block → attribute

---

