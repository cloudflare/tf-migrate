resource "cloudflare_managed_transforms" "only_request" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    }
  ]
}

resource "cloudflare_managed_transforms" "only_response" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  managed_response_headers = [
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
}

resource "cloudflare_managed_transforms" "both" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  managed_request_headers = [
    {
      id      = "add_true_client_ip_headers"
      enabled = true
    }
  ]
  managed_response_headers = [
    {
      id      = "add_security_headers"
      enabled = true
    }
  ]
}

resource "cloudflare_managed_transforms" "neither" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}
