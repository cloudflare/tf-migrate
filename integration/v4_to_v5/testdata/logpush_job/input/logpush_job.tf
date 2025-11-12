# Minimal logpush job
resource "cloudflare_logpush_job" "minimal" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  dataset          = "audit_logs"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
}

# Job with logpull_options only (no output_options)
resource "cloudflare_logpush_job" "with_logpull_options" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  dataset          = "audit_logs"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  logpull_options  = "fields=ClientIP,EdgeStartTimestamp&timestamps=unixnano"
}

# Job with output_options block
resource "cloudflare_logpush_job" "with_output_options" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  dataset          = "audit_logs"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options {
    batch_prefix = "{"
    batch_suffix = "}"
    field_names  = ["ClientIP", "EdgeStartTimestamp"]
    output_type  = "ndjson"
  }
}

# Job with cve20214428 field (should be renamed)
resource "cloudflare_logpush_job" "with_cve_field" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  dataset          = "audit_logs"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options {
    cve20214428 = true
    output_type = "ndjson"
  }
}

# Job with instant-logs kind (should become empty string)
resource "cloudflare_logpush_job" "instant_logs" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "instant-logs"
}

# Job with edge kind (should be preserved)
resource "cloudflare_logpush_job" "edge_logs" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "edge"
}

# Full featured job with all transformations
resource "cloudflare_logpush_job" "full" {
  zone_id       = "0da42c8d2132a9ddaf714f9e7c920711"
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "instant-logs"
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
