package queue

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic queue with required fields",
				Input: `resource "cloudflare_queue" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-queue"
}`,
				Expected: `resource "cloudflare_queue" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  queue_name = "my-queue"
}`,
			},
			{
				Name: "Multiple queue resources",
				Input: `resource "cloudflare_queue" "queue1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "first-queue"
}

resource "cloudflare_queue" "queue2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "second-queue"
}`,
				Expected: `resource "cloudflare_queue" "queue1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  queue_name = "first-queue"
}

resource "cloudflare_queue" "queue2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  queue_name = "second-queue"
}`,
			},
			{
				Name: "Queue with variable reference",
				Input: `resource "cloudflare_queue" "example" {
  account_id = var.cloudflare_account_id
  name       = var.queue_name
}`,
				Expected: `resource "cloudflare_queue" "example" {
  account_id = var.cloudflare_account_id
  queue_name = var.queue_name
}`,
			},
			{
				Name: "Queue with string interpolation",
				Input: `resource "cloudflare_queue" "example" {
  account_id = var.cloudflare_account_id
  name       = "${var.environment}-queue"
}`,
				Expected: `resource "cloudflare_queue" "example" {
  account_id = var.cloudflare_account_id
  queue_name = "${var.environment}-queue"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic queue state transformation",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_queue",
    "name": "example",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "id": "queue-id-12345",
        "name": "my-queue"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_queue",
    "name": "example",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "id": "queue-id-12345",
        "queue_id": "queue-id-12345",
        "queue_name": "my-queue"
      }
    }]
  }]
}`,
			},
			{
				Name: "Queue with special characters in name",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_queue",
    "name": "example",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "id": "queue-id-67890",
        "name": "my-queue-test_123"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_queue",
    "name": "example",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "id": "queue-id-67890",
        "queue_id": "queue-id-67890",
        "queue_name": "my-queue-test_123"
      }
    }]
  }]
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
