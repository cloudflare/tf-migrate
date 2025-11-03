# Load balancer pool with dynamic origins
resource "cloudflare_load_balancer_pool" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-pool"

  dynamic "origins" {
    for_each = ["192.0.2.1", "192.0.2.2"]
    content {
      name    = "origin-${origins.key}"
      address = origins.value
      enabled = true
      weight  = 1.0
      header {
        header = "Host"
        values = ["example.com"]
      }
    }
  }

  check_regions    = ["WEU", "ENAM"]
  description      = "Example pool"
  enabled          = true
  minimum_origins  = 1
  notification_email = "ops@example.com"
}
