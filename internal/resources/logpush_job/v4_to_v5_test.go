package logpush_job

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
}`,
			},
			{
				Name: "Resource with logpull_options only (no output_options)",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  logpull_options  = "fields=ClientIP,EdgeStartTimestamp&timestamps=unixnano"
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  logpull_options  = "fields=ClientIP,EdgeStartTimestamp&timestamps=unixnano"
}`,
			},
			{
				Name: "Convert output_options block to attribute and add v4 defaults",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options {
    batch_prefix = "{"
    batch_suffix = "}"
    field_names  = ["ClientIP", "EdgeStartTimestamp"]
    output_type  = "ndjson"
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options = {
    batch_prefix      = "{"
    batch_suffix      = "}"
    field_names       = ["ClientIP", "EdgeStartTimestamp"]
    output_type       = "ndjson"
    cve_2021_44228    = false
    field_delimiter   = ","
    record_prefix     = "{"
    record_suffix     = "}\n"
    timestamp_format  = "unixnano"
    sample_rate       = 1
  }
}`,
			},
			{
				Name: "Rename cve20214428 to cve_2021_44228 and add v4 defaults",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options {
    cve20214428 = true
    output_type = "ndjson"
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options = {
    output_type       = "ndjson"
    cve_2021_44228    = true
    field_delimiter   = ","
    record_prefix     = "{"
    record_suffix     = "}\n"
    timestamp_format  = "unixnano"
    sample_rate       = 1
  }
}`,
			},
			{
				Name: "Add v4 defaults when some fields are user-configured",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options {
    output_type      = "ndjson"
    field_delimiter  = "|"
    sample_rate      = 0.5
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options = {
    output_type       = "ndjson"
    field_delimiter   = "|"
    sample_rate       = 0.5
    cve_2021_44228    = false
    record_prefix     = "{"
    record_suffix     = "}\n"
    timestamp_format  = "unixnano"
  }
}`,
			},
			{
				Name: "Preserve all user-configured values over defaults",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options {
    output_type      = "csv"
    field_delimiter  = "|"
    record_prefix    = "["
    record_suffix    = "]"
    timestamp_format = "rfc3339"
    sample_rate      = 0.1
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"

  output_options = {
    output_type       = "csv"
    field_delimiter   = "|"
    record_prefix     = "["
    record_suffix     = "]"
    timestamp_format  = "rfc3339"
    sample_rate       = 0.1
    cve_2021_44228    = false
  }
}`,
			},
			{
				Name: "Handle kind instant-logs by removing attribute",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "instant-logs"
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
}`,
			},
			{
				Name: "Preserve kind edge value",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "edge"
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "edge"
}`,
			},
			{
				Name: "Preserve kind empty string",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = ""
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = ""
}`,
			},
			{
				Name: "Full transformation with all changes",
				Input: `resource "cloudflare_logpush_job" "example" {
  account_id       = "abc123"
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "instant-logs"
  enabled          = true

  output_options {
    cve20214428      = true
    batch_prefix     = "{"
    batch_suffix     = "}"
    field_names      = ["ClientIP", "EdgeStartTimestamp", "RayID"]
    output_type      = "ndjson"
    sample_rate      = 1.0
    timestamp_format = "unixnano"
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  account_id       = "abc123"
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  enabled          = true

  output_options = {
    batch_prefix     = "{"
    batch_suffix     = "}"
    field_names      = ["ClientIP", "EdgeStartTimestamp", "RayID"]
    output_type      = "ndjson"
    sample_rate      = 1.0
    timestamp_format = "unixnano"
    cve_2021_44228   = true
    field_delimiter  = ","
    record_prefix    = "{"
    record_suffix    = "}\n"
  }
}`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `resource "cloudflare_logpush_job" "job1" {
  dataset          = "http_requests"
  destination_conf = "s3://bucket1/logs?region=us-west-2"
  kind             = "instant-logs"

  output_options {
    cve20214428 = true
    output_type = "ndjson"
  }
}

resource "cloudflare_logpush_job" "job2" {
  dataset          = "firewall_events"
  destination_conf = "s3://bucket2/logs?region=us-east-1"

  output_options {
    output_type = "csv"
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "job1" {
  dataset          = "http_requests"
  destination_conf = "s3://bucket1/logs?region=us-west-2"

  output_options = {
    output_type       = "ndjson"
    cve_2021_44228    = true
    field_delimiter   = ","
    record_prefix     = "{"
    record_suffix     = "}\n"
    timestamp_format  = "unixnano"
    sample_rate       = 1
  }
}

resource "cloudflare_logpush_job" "job2" {
  dataset          = "firewall_events"
  destination_conf = "s3://bucket2/logs?region=us-east-1"

  output_options = {
    output_type       = "csv"
    cve_2021_44228    = false
    field_delimiter   = ","
    record_prefix     = "{"
    record_suffix     = "}\n"
    timestamp_format  = "unixnano"
    sample_rate       = 1
  }
}`,
			},
			{
				Name: "With variable references",
				Input: `resource "cloudflare_logpush_job" "example" {
  account_id       = var.account_id
  dataset          = var.dataset
  destination_conf = var.destination_conf

  output_options {
    field_names = var.field_names
    output_type = "ndjson"
  }
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  account_id       = var.account_id
  dataset          = var.dataset
  destination_conf = var.destination_conf

  output_options = {
    field_names       = var.field_names
    output_type       = "ndjson"
    cve_2021_44228    = false
    field_delimiter   = ","
    record_prefix     = "{"
    record_suffix     = "}\n"
    timestamp_format  = "unixnano"
    sample_rate       = 1
  }
}`,
			},
			{
				Name: "With deprecated frequency field",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  frequency        = "high"
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  frequency        = "high"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
