# Workers Secret Migration Guide (v4 → v5)

## Overview

The `cloudflare_worker_secret` and `cloudflare_workers_secret` resources from v4 have been consolidated into a single `cloudflare_workers_secret` resource in v5.

## Changes

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name (singular) | `cloudflare_worker_secret` | `cloudflare_workers_secret` | **Renamed** |
| Resource name (plural) | `cloudflare_workers_secret` | `cloudflare_workers_secret` | No change |
| `secret` attribute | `secret` | `secret_text` | **Renamed** |
| `account_id` | Optional | Required | **Now required** |
| `name` | Required | Required | No change |
| `script_name` | Required | Required | No change |
| `dispatch_namespace` | N/A | Optional | New in v5 |

## Migration Behavior

### Singular Form (`cloudflare_worker_secret`)

The resource type is renamed and a `moved` block is generated:

```hcl
# v4
resource "cloudflare_worker_secret" "my_secret" {
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}

# v5
resource "cloudflare_workers_secret" "my_secret" {
  name         = "MY_SECRET"
  script_name  = "my-worker"
  secret_text  = "super-secret-value"
  # MIGRATION REQUIRED: Add account_id attribute (required in v5)
}

moved {
  from = cloudflare_worker_secret.my_secret
  to   = cloudflare_workers_secret.my_secret
}
```

### Plural Form (`cloudflare_workers_secret`)

Only the attribute is renamed (no `moved` block needed):

```hcl
# v4
resource "cloudflare_workers_secret" "my_secret" {
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret      = "super-secret-value"
}

# v5
resource "cloudflare_workers_secret" "my_secret" {
  name         = "MY_SECRET"
  script_name  = "my-worker"
  secret_text  = "super-secret-value"
  # MIGRATION REQUIRED: Add account_id attribute (required in v5)
}
```

## Important: Required Account ID

The v5 provider requires `account_id` to be specified on the resource. If your v4 configuration doesn't have it, you'll need to add it manually after migration:

```hcl
resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "your-account-id"  # <-- Add this
  name        = "MY_SECRET"
  script_name = "my-worker"
  secret_text = "super-secret-value"
}
```

The migration tool will add a warning comment if `account_id` is missing.

## Testing

### Unit Tests

```bash
go test ./internal/resources/workers_secret -v
```

### Integration Tests

```bash
TEST_RESOURCE=workers_secret go test -v -run TestSingleResource ./integration/v4_to_v5/
```

### E2E Tests

```bash
./scripts/run-e2e-tests.sh --resources workers_secret --apply-exemptions
```
