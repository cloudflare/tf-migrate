# Test Case 1: Basic configuration with request headers
resource "cloudflare_managed_transforms" "example_request" {
  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
    }, {
    id      = "add_visitor_location_headers"
    enabled = false
  }]
  managed_response_headers = []
}

# Test Case 2: Basic configuration with response headers
resource "cloudflare_managed_transforms" "example_response" {
  zone_id = var.cloudflare_zone_id


  managed_request_headers = []
  managed_response_headers = [{
    id      = "remove_x-powered-by_header"
    enabled = true
    }, {
    id      = "add_security_headers"
    enabled = false
  }]
}

# Test Case 3: Configuration with both request and response headers
resource "cloudflare_managed_transforms" "example_both" {
  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = "add_bot_protection_headers"
    enabled = true
  }]
  managed_response_headers = [{
    id      = "remove_server_header"
    enabled = true
  }]
}

# Test Case 4: Minimal configuration with no headers
resource "cloudflare_managed_transforms" "example_minimal" {
  zone_id                  = var.cloudflare_zone_id
  managed_request_headers  = []
  managed_response_headers = []
}
