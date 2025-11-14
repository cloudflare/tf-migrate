# Test Case 1: Basic configuration with request headers
resource "cloudflare_managed_transforms" "example_request" {
  zone_id = "d56084adb405e0b7e32c52321bf07be6"


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
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"


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
  zone_id = "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"


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
  zone_id                  = "z9y8x7w6v5u4t3s2r1q0p9o8n7m6l5k4"
  managed_request_headers  = []
  managed_response_headers = []
}
