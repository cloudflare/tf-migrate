# Test Case 1: Account-level custom page
resource "cloudflare_custom_pages" "account_500" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  state      = "customized"
  url        = "https://example.workers.dev/500.html"
  identifier = "500_errors"
}

# Test Case 2: Zone-level custom page
resource "cloudflare_custom_pages" "zone_basic_challenge" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  state      = "customized"
  url        = "https://example.workers.dev/challenge.html"
  identifier = "basic_challenge"
}

# Test Case 3: WAF block page
resource "cloudflare_custom_pages" "waf_block" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  state      = "customized"
  url        = "https://example.workers.dev/waf.html"
  identifier = "waf_block"
}
