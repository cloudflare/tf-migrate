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
  }
}`,
			},
			{
				Name: "Handle kind instant-logs to empty string",
				Input: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = "instant-logs"
}`,
				Expected: `resource "cloudflare_logpush_job" "example" {
  dataset          = "http_requests"
  destination_conf = "s3://mybucket/logs?region=us-west-2"
  kind             = ""
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
  kind             = ""
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
  kind             = ""

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

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2"
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2"
  }
}`,
			},
			{
				Name: "State with logpull_options only (no output_options)",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "logpull_options": "fields=ClientIP,EdgeStartTimestamp&timestamps=unixnano"
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "logpull_options": "fields=ClientIP,EdgeStartTimestamp&timestamps=unixnano"
  }
}`,
			},
			{
				Name: "Transform output_options array to object",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": [
      {
        "batch_prefix": "{",
        "batch_suffix": "}",
        "field_names": ["ClientIP", "EdgeStartTimestamp"],
        "output_type": "ndjson"
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": {
      "batch_prefix": "{",
      "batch_suffix": "}",
      "field_names": ["ClientIP", "EdgeStartTimestamp"],
      "output_type": "ndjson",
      "field_delimiter": ",",
      "record_prefix": "{",
      "record_suffix": "}\n",
      "timestamp_format": "unixnano",
      "sample_rate": 1
    }
  }
}`,
			},
			{
				Name: "Rename cve20214428 in output_options array",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": [
      {
        "cve20214428": true,
        "output_type": "ndjson"
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": {
      "cve_2021_44228": true,
      "output_type": "ndjson",
      "field_delimiter": ",",
      "record_prefix": "{",
      "record_suffix": "}\n",
      "timestamp_format": "unixnano",
      "sample_rate": 1
    }
  }
}`,
			},
			{
				Name: "Rename cve20214428 in output_options object",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": {
      "cve20214428": true,
      "output_type": "ndjson"
    }
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": {
      "cve_2021_44228": true,
      "output_type": "ndjson",
      "field_delimiter": ",",
      "record_prefix": "{",
      "record_suffix": "}\n",
      "timestamp_format": "unixnano",
      "sample_rate": 1
    }
  }
}`,
			},
			{
				Name: "Convert numeric fields to float64",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "max_upload_bytes": 5000000,
    "max_upload_records": 1000,
    "max_upload_interval_seconds": 30
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "max_upload_bytes": 5000000.0,
    "max_upload_records": 1000.0,
    "max_upload_interval_seconds": 30.0
  }
}`,
			},
			{
				Name: "Remove numeric fields with value 0 (API defaults)",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "max_upload_bytes": 0,
    "max_upload_records": 0,
    "max_upload_interval_seconds": 0
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2"
  }
}`,
			},
			{
				Name: "Preserve v4 schema defaults in output_options state",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": [
      {
        "cve20214428": true,
        "output_type": "ndjson",
        "field_names": ["ClientIP", "EdgeStartTimestamp"],
        "field_delimiter": ",",
        "record_prefix": "{",
        "record_suffix": "}\n",
        "sample_rate": 1.0,
        "timestamp_format": "unixnano"
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": {
      "cve_2021_44228": true,
      "output_type": "ndjson",
      "field_names": ["ClientIP", "EdgeStartTimestamp"],
      "field_delimiter": ",",
      "record_prefix": "{",
      "record_suffix": "}\n",
      "sample_rate": 1.0,
      "timestamp_format": "unixnano"
    }
  }
}`,
			},
			{
				Name: "Preserve user-configured values over v4 defaults in state",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": [
      {
        "output_type": "csv",
        "field_delimiter": "|",
        "record_prefix": "[",
        "sample_rate": 0.5,
        "timestamp_format": "rfc3339"
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": {
      "output_type": "csv",
      "field_delimiter": "|",
      "record_prefix": "[",
      "sample_rate": 0.5,
      "timestamp_format": "rfc3339",
      "record_suffix": "}\n"
    }
  }
}`,
			},
			{
				Name: "Handle kind instant-logs to empty string",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "kind": "instant-logs"
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "kind": ""
  }
}`,
			},
			{
				Name: "Preserve kind edge value",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "kind": "edge"
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "kind": "edge"
  }
}`,
			},
			{
				Name: "Remove computed-only fields",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "error_message": "Some error",
    "last_complete": "2024-01-01T00:00:00Z",
    "last_error": "2024-01-01T00:00:00Z"
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2"
  }
}`,
			},
			{
				Name: "Empty output_options array removed",
				Input: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "output_options": []
  }
}`,
				Expected: `{
  "attributes": {
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2"
  }
}`,
			},
			{
				Name: "Full state transformation",
				Input: `{
  "attributes": {
    "account_id": "abc123",
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "kind": "instant-logs",
    "enabled": true,
    "max_upload_bytes": 5000000,
    "max_upload_records": 1000,
    "max_upload_interval_seconds": 30,
    "output_options": [
      {
        "cve20214428": true,
        "batch_prefix": "{",
        "batch_suffix": "}",
        "field_names": ["ClientIP", "EdgeStartTimestamp", "RayID"],
        "output_type": "ndjson",
        "sample_rate": 1.0,
        "timestamp_format": "unixnano"
      }
    ],
    "error_message": "Some error",
    "last_complete": "2024-01-01T00:00:00Z",
    "last_error": "2024-01-01T00:00:00Z"
  }
}`,
				Expected: `{
  "attributes": {
    "account_id": "abc123",
    "dataset": "http_requests",
    "destination_conf": "s3://mybucket/logs?region=us-west-2",
    "kind": "",
    "enabled": true,
    "max_upload_bytes": 5000000.0,
    "max_upload_records": 1000.0,
    "max_upload_interval_seconds": 30.0,
    "output_options": {
      "cve_2021_44228": true,
      "batch_prefix": "{",
      "batch_suffix": "}",
      "field_names": ["ClientIP", "EdgeStartTimestamp", "RayID"],
      "output_type": "ndjson",
      "sample_rate": 1.0,
      "timestamp_format": "unixnano",
      "field_delimiter": ",",
      "record_prefix": "{",
      "record_suffix": "}\n"
    }
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
