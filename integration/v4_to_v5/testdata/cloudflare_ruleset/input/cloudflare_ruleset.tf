# Cloudflare ruleset example
# Note: Config transformation is extremely complex (815 lines)
# Users should use the provider's migrate tool for config transformation
# This test focuses on state transformation only

resource "cloudflare_ruleset" "example" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "Example Ruleset"
  description = "Example ruleset for testing"
  kind        = "zone"
  phase       = "http_request_firewall_custom"
}
