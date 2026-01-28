package logpush_ownership_challenge

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic resource with zone_id - config unchanged",
				Input: `resource "cloudflare_logpush_ownership_challenge" "test" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  destination_conf = "s3://my-bucket-path?region=us-west-2"
}`,
				Expected: `resource "cloudflare_logpush_ownership_challenge" "test" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  destination_conf = "s3://my-bucket-path?region=us-west-2"
}`,
			},
			{
				Name: "Basic resource with account_id - config unchanged",
				Input: `resource "cloudflare_logpush_ownership_challenge" "test" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  destination_conf = "s3://my-bucket-path?region=us-east-1"
}`,
				Expected: `resource "cloudflare_logpush_ownership_challenge" "test" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  destination_conf = "s3://my-bucket-path?region=us-east-1"
}`,
			},
			{
				Name: "Multiple resources - all configs unchanged",
				Input: `resource "cloudflare_logpush_ownership_challenge" "zone_challenge" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  destination_conf = "s3://zone-logs?region=us-west-2"
}

resource "cloudflare_logpush_ownership_challenge" "account_challenge" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  destination_conf = "s3://account-logs?region=us-east-1"
}`,
				Expected: `resource "cloudflare_logpush_ownership_challenge" "zone_challenge" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  destination_conf = "s3://zone-logs?region=us-west-2"
}

resource "cloudflare_logpush_ownership_challenge" "account_challenge" {
  account_id       = "f037e56e89293a057740de681ac9abbe"
  destination_conf = "s3://account-logs?region=us-east-1"
}`,
			},
			{
				Name: "Variable references preserved",
				Input: `resource "cloudflare_logpush_ownership_challenge" "test" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = var.s3_bucket_path
}`,
				Expected: `resource "cloudflare_logpush_ownership_challenge" "test" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = var.s3_bucket_path
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Remove ownership_challenge_filename - zone_id variant",
				Input: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "attributes": {
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "destination_conf": "s3://my-bucket",
    "ownership_challenge_filename": "logs/ownership-challenge.txt"
  }
}`,
				Expected: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "schema_version": 0,
  "attributes": {
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "destination_conf": "s3://my-bucket"
  }
}`,
			},
			{
				Name: "Remove ownership_challenge_filename - account_id variant",
				Input: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "destination_conf": "s3://my-bucket",
    "ownership_challenge_filename": "logs/ownership-challenge.txt"
  }
}`,
				Expected: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "destination_conf": "s3://my-bucket"
  }
}`,
			},
			{
				Name: "Handle missing ownership_challenge_filename gracefully",
				Input: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "attributes": {
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "destination_conf": "s3://my-bucket"
  }
}`,
				Expected: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "schema_version": 0,
  "attributes": {
    "zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
    "destination_conf": "s3://my-bucket"
  }
}`,
			},
			{
				Name: "Handle empty attributes - set schema_version",
				Input: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "attributes": {}
}`,
				Expected: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "schema_version": 0,
  "attributes": {}
}`,
			},
			{
				Name: "Handle missing attributes - set schema_version",
				Input: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test"
}`,
				Expected: `{
  "type": "cloudflare_logpush_ownership_challenge",
  "name": "test",
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
