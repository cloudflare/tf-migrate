variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# Single comprehensive test covering all valid header types
resource "cloudflare_managed_transforms" "test" {
  zone_id = var.cloudflare_zone_id




  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
    }, {
    id      = "add_visitor_location_headers"
    enabled = false
  }]
  managed_response_headers = [{
    id      = "add_security_headers"
    enabled = true
    }, {
    id      = "remove_x-powered-by_header"
    enabled = false
  }]
}
