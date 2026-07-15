# workers_secret Migration (v4 -> v5)

## Overview

The `cloudflare_workers_secret` (and deprecated `cloudflare_worker_secret`)
resource has been **removed** in the v5 provider. Worker secrets are now
managed as `secret_text` bindings on the `cloudflare_workers_script` resource.

## v4 Configuration

```hcl
resource "cloudflare_workers_script" "my_worker" {
  account_id = "abc123"
  name       = "my-worker"
  content    = file("worker.js")
}

resource "cloudflare_workers_secret" "api_key" {
  account_id  = "abc123"
  script_name = cloudflare_workers_script.my_worker.name
  name        = "API_KEY"
  secret_text = "my-api-key"
}
```

## v5 Configuration (after migration)

```hcl
resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = file("worker.js")
  bindings = [
    {
      type = "secret_text"
      name = "API_KEY"
      text = "my-api-key"
    }
  ]
}

removed {
  from = cloudflare_workers_secret.api_key
  lifecycle {
    destroy = false
  }
}
```

## Attribute Mapping

| v4 (`cloudflare_workers_secret`) | v5 (`cloudflare_workers_script.bindings[]`) |
|---|---|
| `name` | `name` |
| `secret_text` | `text` |
| (implicit) | `type = "secret_text"` |
| `script_name` | used to find parent script |
| `account_id` | dropped (already on parent) |

## Migration Behavior

### Cross-Resource Merge

When the parent `cloudflare_workers_script` is in the same file, the migrator
automatically merges the secret into the script's `bindings` list:

- **No existing bindings**: creates a new `bindings = [...]` attribute
- **Existing bindings**: wraps in `concat(existing, [new_secret])` to preserve
  both the original bindings and the merged secret

### Parent Matching

The migrator matches secrets to their parent script by:

1. **Reference matching**: parses `script_name = cloudflare_workers_script.NAME.script_name`
   to extract the resource name (supports both v4 singular and v5 plural prefixes)
2. **Literal matching**: compares the literal `script_name` value against each
   script's `script_name` attribute

### Orphan Secrets

If the parent script is not in the same file, the migrator:

- Generates a `removed {}` block
- Emits a diagnostic warning with the binding snippet to add manually

### Both v4 Names Supported

Both `cloudflare_workers_secret` (preferred) and `cloudflare_worker_secret`
(deprecated singular) are handled identically.

## Architecture

This migrator follows the cross-resource merge pattern established by
`zero_trust_split_tunnel` (merged into device profiles):

1. **`TransformConfig`**: generates `removed` block + diagnostic for each secret
2. **`ProcessCrossResourceConfigMigration`**: called from the `workers_script`
   migrator after its own binding transformation completes; scans the file,
   matches secrets to scripts, and merges them

The cross-resource merge only processes scripts that have already been migrated
(identified by the presence of `script_name` instead of `name`). This ensures
correct ordering when the pipeline processes blocks sequentially.
