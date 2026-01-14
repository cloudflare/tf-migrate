# Logpush Job Migration Guide (v4 → v5)

This guide explains how `cloudflare_logpush_job` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_logpush_job` | `cloudflare_logpush_job` | No change |
| `output_options` | Block | Attribute object | Syntax change |
| `output_options.cve20214428` | `cve20214428 = false` | `cve_2021_44228 = false` | Field renamed |
| `kind = "instant-logs"` | Supported | Removed | Deprecated |
| Numeric fields | Int | Int64 (float64 in state) | Type conversion |
| Computed fields | `error_message`, `last_complete`, `last_error` | Removed | No longer returned |


---

## Migration Examples

### Example 1: Basic HTTP Logpush Job

**v4 Configuration:**
```hcl
resource "cloudflare_logpush_job" "example" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "example-job"
  dataset             = "http_requests"
  destination_conf    = "s3://my-bucket/logs?region=us-west-2"
  ownership_challenge = "00000000000000000000"
  enabled             = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpush_job" "example" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "example-job"
  dataset             = "http_requests"
  destination_conf    = "s3://my-bucket/logs?region=us-west-2"
  ownership_challenge = "00000000000000000000"
  enabled             = true
}
```

**What Changed:**
- Configuration unchanged for basic jobs
- State receives default values for output_options internally

---

### Example 2: Logpush Job with Output Options

**v4 Configuration:**
```hcl
resource "cloudflare_logpush_job" "with_options" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "custom-format-job"
  dataset             = "http_requests"
  destination_conf    = "s3://my-bucket/logs?region=us-west-2"
  ownership_challenge = "00000000000000000000"
  enabled             = true

  output_options {
    cve20214428        = false
    field_delimiter    = ","
    record_prefix      = "{"
    record_suffix      = "}\n"
    timestamp_format   = "rfc3339"
    sample_rate        = 0.5
    field_names        = ["ClientIP", "EdgeStartTimestamp", "RayID"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpush_job" "with_options" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "custom-format-job"
  dataset             = "http_requests"
  destination_conf    = "s3://my-bucket/logs?region=us-west-2"
  ownership_challenge = "00000000000000000000"
  enabled             = true

  output_options = {
    cve_2021_44228     = false
    field_delimiter    = ","
    record_prefix      = "{"
    record_suffix      = "}\n"
    timestamp_format   = "rfc3339"
    sample_rate        = 0.5
    field_names        = ["ClientIP", "EdgeStartTimestamp", "RayID"]
  }
}
```

**What Changed:**
- `output_options { }` block → `output_options = { }` attribute
- `cve20214428` → `cve_2021_44228`

---

### Example 3: Zone-scoped Logpush with Filter

**v4 Configuration:**
```hcl
resource "cloudflare_logpush_job" "zone_filtered" {
  zone_id             = "0da42c8d2132a9ddaf714f9e7c920711"
  name                = "zone-http-logs"
  dataset             = "http_requests"
  destination_conf    = "s3://my-bucket/zone-logs?region=us-west-2"
  ownership_challenge = "00000000000000000000"
  enabled             = true
  filter              = "ClientRequestPath contains '/api/'"
  frequency           = "high"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpush_job" "zone_filtered" {
  zone_id             = "0da42c8d2132a9ddaf714f9e7c920711"
  name                = "zone-http-logs"
  dataset             = "http_requests"
  destination_conf    = "s3://my-bucket/zone-logs?region=us-west-2"
  ownership_challenge = "00000000000000000000"
  enabled             = true
  filter              = "ClientRequestPath contains '/api/'"
  frequency           = "high"
}
```

**What Changed:**
- No configuration changes needed
- Filter field preserved

---

### Example 4: Job with Max Upload Settings

**v4 Configuration:**
```hcl
resource "cloudflare_logpush_job" "batched" {
  account_id             = "f037e56e89293a057740de681ac9abbe"
  name                   = "batched-job"
  dataset                = "http_requests"
  destination_conf       = "s3://my-bucket/batched?region=us-west-2"
  ownership_challenge    = "00000000000000000000"
  enabled                = true
  max_upload_bytes       = 5000000
  max_upload_records     = 1000
  max_upload_interval_seconds = 30
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpush_job" "batched" {
  account_id             = "f037e56e89293a057740de681ac9abbe"
  name                   = "batched-job"
  dataset                = "http_requests"
  destination_conf       = "s3://my-bucket/batched?region=us-west-2"
  ownership_challenge    = "00000000000000000000"
  enabled                = true
  max_upload_bytes       = 5000000
  max_upload_records     = 1000
  max_upload_interval_seconds = 30
}
```

**What Changed:**
- Configuration unchanged
- Numeric values converted to float64 in state

---

### Example 5: Instant Logs (Deprecated)

**v4 Configuration:**
```hcl
resource "cloudflare_logpush_job" "instant" {
  zone_id             = "0da42c8d2132a9ddaf714f9e7c920711"
  name                = "instant-logs"
  dataset             = "http_requests"
  destination_conf    = "gs://my-bucket/instant"
  ownership_challenge = "00000000000000000000"
  kind                = "instant-logs"  # ⚠️ Deprecated
  enabled             = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_logpush_job" "instant" {
  zone_id             = "0da42c8d2132a9ddaf714f9e7c920711"
  name                = "instant-logs"
  dataset             = "http_requests"
  destination_conf    = "gs://my-bucket/instant"
  ownership_challenge = "00000000000000000000"
  # kind attribute removed - "instant-logs" is no longer valid in v5
  enabled             = true
}
```

**What Changed:**
- `kind = "instant-logs"` completely removed (no longer supported)

---

