variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Use pre-existing webhook endpoint worker
# Worker URL: https://e2e-webhook-endpoint.terraform-testing-a09.workers.dev/
# This worker responds with 200 OK to all requests for webhook validation

# Test Case 1: Basic webhook with minimal fields
resource "cloudflare_notification_policy_webhooks" "basic_webhook" {
    account_id = var.cloudflare_account_id
    name       = "basic-webhook"
    url        = "https://e2e-webhook-endpoint.terraform-testing-a09.workers.dev/basic"
}

# Test Case 2: Full webhook with all fields
resource "cloudflare_notification_policy_webhooks" "full_webhook" {
    account_id = var.cloudflare_account_id
    name       = "production-webhook"
    url        = "https://e2e-webhook-endpoint.terraform-testing-a09.workers.dev/full"
    secret     = "webhook-secret-token-12345"
}

# Test Case 3: Multiple webhooks
resource "cloudflare_notification_policy_webhooks" "primary" {
    account_id = var.cloudflare_account_id
    name       = "primary-webhook"
    url        = "https://e2e-webhook-endpoint.terraform-testing-a09.workers.dev/primary"
}

resource "cloudflare_notification_policy_webhooks" "backup" {
    account_id = var.cloudflare_account_id
    name       = "backup-webhook"
    url        = "https://e2e-webhook-endpoint.terraform-testing-a09.workers.dev/backup"
    secret     = "backup-secret"
}
