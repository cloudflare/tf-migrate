variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Minimal logpush job
resource "cloudflare_logpush_job" "minimal" {
  account_id       = var.cloudflare_account_id
  dataset          = "audit_logs"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"
}

# Job with logpull_options only (no output_options)
resource "cloudflare_logpush_job" "with_logpull_options" {
  account_id       = var.cloudflare_account_id
  dataset          = "audit_logs"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"
  logpull_options  = "fields=ClientIP,EdgeStartTimestamp&timestamps=unixnano"
}

# Job with output_options block
resource "cloudflare_logpush_job" "with_output_options" {
  account_id       = var.cloudflare_account_id
  dataset          = "audit_logs"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"

  output_options {
    batch_prefix = "{"
    batch_suffix = "}"
    field_names  = ["ClientIP", "EdgeStartTimestamp"]
    output_type  = "ndjson"
  }
}

# Job with cve20214428 field (should be renamed)
resource "cloudflare_logpush_job" "with_cve_field" {
  account_id       = var.cloudflare_account_id
  dataset          = "audit_logs"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"

  output_options {
    cve20214428 = true
    output_type = "ndjson"
  }
}

# Job with edge kind (should be preserved)
resource "cloudflare_logpush_job" "edge_logs" {
  zone_id          = var.cloudflare_zone_id
  dataset          = "http_requests"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"
  kind             = "edge"
}

# Job with "" kind (should be preserved)
resource "cloudflare_logpush_job" "empty_kind" {
  zone_id          = var.cloudflare_zone_id
  dataset          = "http_requests"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"
  kind             = ""
}

# Full featured job with all transformations
resource "cloudflare_logpush_job" "full" {
  zone_id          = var.cloudflare_zone_id
  dataset          = "audit_logs"
  destination_conf = "https://logpush-receiver.sd.cfplat.com"
  kind             = "edge"
  enabled          = true
  name             = "my-logpush-job"
  frequency        = "high"

  output_options {
    cve20214428      = true
    batch_prefix     = "{"
    batch_suffix     = "}"
    field_names      = ["ClientIP", "EdgeStartTimestamp", "RayID"]
    output_type      = "ndjson"
    sample_rate      = 1.0
    timestamp_format = "unixnano"
  }
}
