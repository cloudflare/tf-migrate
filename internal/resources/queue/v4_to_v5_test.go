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
}
