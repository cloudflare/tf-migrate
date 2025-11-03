# Snippet rules example - no transformation needed
resource "cloudflare_snippet_rules" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  rules = [
    {
      enabled      = true
      expression   = "true"
      description  = "Apply snippet to all requests"
      snippet_name = "example_snippet"
    }
  ]
}
