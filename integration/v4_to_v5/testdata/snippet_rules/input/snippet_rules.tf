locals {
  name_prefix = "cftftest"
}

resource "cloudflare_snippet_rules" "basic" {
  zone_id = "test-zone-id"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/api\""
    snippet_name = "${local.name_prefix}-basic-snippet"
    description  = "Basic test rule"
  }
}

resource "cloudflare_snippet_rules" "multiple_rules" {
  zone_id = "test-zone-id-2"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/v1\""
    snippet_name = "${local.name_prefix}-v1-snippet"
  }

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/v2\""
    snippet_name = "${local.name_prefix}-v2-snippet"
    description  = "Version 2 API"
  }
}
