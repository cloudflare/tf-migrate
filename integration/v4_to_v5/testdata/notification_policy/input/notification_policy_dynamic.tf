variable "alert_emails" {
  type    = list(string)
  default = ["oncall@example.com", "platform@example.com"]
}

# A dynamic integration block alongside a static one of a different type.
resource "cloudflare_notification_policy" "dynamic_email" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "cftftest-origin-errors-by-code"
  description = "5xx rate from origin exceeds threshold"
  enabled     = true
  alert_type  = "http_alert_origin_error"

  dynamic "email_integration" {
    for_each = var.alert_emails
    content {
      id = email_integration.value
    }
  }

  webhooks_integration {
    id = "b14b4b4b4b4b4b4b4b4b4b4b4b4b4b4b"
  }
}
