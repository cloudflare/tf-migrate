# Test Case 1: Basic webhook with minimal fields
resource "cloudflare_notification_policy_webhooks" "basic_webhook" {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "basic-webhook"
    url        = "https://example.com/webhook"
}

# Test Case 2: Full webhook with all fields
resource "cloudflare_notification_policy_webhooks" "full_webhook" {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "production-webhook"
    url        = "https://alerts.example.com/notify"
    secret     = "webhook-secret-token-12345"
}

# Test Case 3: Multiple webhooks
resource "cloudflare_notification_policy_webhooks" "primary" {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "primary-webhook"
    url        = "https://primary.example.com/webhook"
}

resource "cloudflare_notification_policy_webhooks" "backup" {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "backup-webhook"
    url        = "https://backup.example.com/webhook"
    secret     = "backup-secret"
}
