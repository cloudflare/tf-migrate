variable "alert_emails" {
  type    = list(string)
  default = ["oncall@example.com", "platform@example.com"]
}

# A dynamic integration block alongside a static one of a different type.
resource "cloudflare_notification_policy" "dynamic_email" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Origin errors by code"
  description = "5xx rate from origin exceeds threshold"
  enabled     = true
  alert_type  = "http_alert_origin_error"


  mechanisms = {
    email = [for k, v in var.alert_emails : {
      id = v
    }]
    webhooks = [{
      id = "b14b4b4b4b4b4b4b4b4b4b4b4b4b4b4b"
    }]
  }
}
