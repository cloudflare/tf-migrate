resource "cloudflare_load_balancer_monitor" "test" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  type           = "https"
  description    = "Test HTTPS monitor"
  method         = "GET"
  path           = "/health"
  interval       = 30
  retries        = 3
  timeout        = 10
  expected_codes = "2xx"
  expected_body  = "healthy"
  allow_insecure = true


  header = {
    "Host"          = ["api.example.com"]
    "Authorization" = ["Bearer token123"]
  }
}

resource "cloudflare_load_balancer_monitor" "minimal" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}
