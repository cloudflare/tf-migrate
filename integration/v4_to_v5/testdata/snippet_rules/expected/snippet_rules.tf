resource "cloudflare_snippet_rules" "basic" {
  zone_id = "test-zone-id"

  rules = [
    {
      enabled      = true
      expression   = "http.request.uri.path eq \"/api\""
      snippet_name = "basic_snippet"
      description  = "Basic test rule"
    }
  ]
}

resource "cloudflare_snippet_rules" "multiple_rules" {
  zone_id = "test-zone-id-2"


  rules = [
    {
      enabled      = true
      expression   = "http.request.uri.path eq \"/v1\""
      snippet_name = "v1_snippet"
    },
    {
      enabled      = false
      expression   = "http.request.uri.path eq \"/v2\""
      snippet_name = "v2_snippet"
      description  = "Version 2 API"
    }
  ]
}
