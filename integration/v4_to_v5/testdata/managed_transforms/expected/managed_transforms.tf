variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# ========================================
# Test Case 1: Basic Configurations
# ========================================

# Test 1.1: Minimal configuration with no headers
resource "cloudflare_managed_transforms" "minimal" {
  zone_id                  = var.cloudflare_zone_id
  managed_request_headers  = []
  managed_response_headers = []
}

# Test 1.2: Request headers only
resource "cloudflare_managed_transforms" "request_only" {
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

# Test 1.3: Response headers only
resource "cloudflare_managed_transforms" "response_only" {
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

# Test 1.4: Both request and response headers
resource "cloudflare_managed_transforms" "both_headers" {
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

# Test 1.5: Multiple headers of each type
resource "cloudflare_managed_transforms" "multiple_headers" {
  zone_id = var.cloudflare_zone_id






  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
    }, {
    id      = "add_visitor_location_headers"
    enabled = true
    }, {
    id      = "add_bot_protection_headers"
    enabled = false
  }]
  managed_response_headers = [{
    id      = "add_security_headers"
    enabled = true
    }, {
    id      = "remove_x-powered-by_header"
    enabled = true
    }, {
    id      = "remove_server_header"
    enabled = false
  }]
}

# ========================================
# Test Case 2: Variable References
# ========================================

variable "enable_security_headers" {
  type    = bool
  default = true
}

variable "header_configs" {
  type = map(object({
    request_headers  = list(string)
    response_headers = list(string)
  }))
  default = {
    "production" = {
      request_headers  = ["add_true_client_ip_headers"]
      response_headers = ["add_security_headers", "remove_x-powered-by_header"]
    }
    "staging" = {
      request_headers  = ["add_visitor_location_headers"]
      response_headers = ["remove_server_header"]
    }
  }
}

# Test 2.1: Conditional resource creation with count
resource "cloudflare_managed_transforms" "conditional_headers" {
  count = var.enable_security_headers ? 1 : 0

  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = [{
    id      = "add_security_headers"
    enabled = true
  }]
}

# ========================================
# Test Case 3: for_each with Maps
# ========================================

# Test 3.1: for_each over map of configurations
resource "cloudflare_managed_transforms" "env_specific" {
  for_each = var.header_configs

  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = each.value.request_headers[0]
    enabled = true
  }]
  managed_response_headers = [{
    id      = each.value.response_headers[0]
    enabled = true
  }]
}

# ========================================
# Test Case 4: Locals and Expressions
# ========================================

locals {
  zones = {
    "zone1" = var.cloudflare_zone_id
    "zone2" = var.cloudflare_zone_id
  }

  header_enabled = true
  common_headers = ["add_true_client_ip_headers", "add_visitor_location_headers"]
}

# Test 4.1: Using locals
resource "cloudflare_managed_transforms" "with_locals" {
  zone_id = local.zones["zone1"]


  managed_request_headers = [{
    id      = local.common_headers[0]
    enabled = local.header_enabled
    }, {
    id      = local.common_headers[1]
    enabled = !local.header_enabled
  }]
  managed_response_headers = []
}

# Test 4.2: for_each over local values
resource "cloudflare_managed_transforms" "multi_zone" {
  for_each = local.zones

  zone_id = each.value

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = []
}

# ========================================
# Test Case 5: Boolean Edge Cases
# ========================================

# Test 5.1: All headers enabled
resource "cloudflare_managed_transforms" "all_enabled" {
  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = "add_visitor_location_headers"
    enabled = true
  }]
  managed_response_headers = [{
    id      = "remove_server_header"
    enabled = true
  }]
}

# Test 5.2: All headers disabled
resource "cloudflare_managed_transforms" "all_disabled" {
  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = "add_visitor_location_headers"
    enabled = false
  }]
  managed_response_headers = [{
    id      = "remove_server_header"
    enabled = false
  }]
}

# Test 5.3: Mixed enabled/disabled states
resource "cloudflare_managed_transforms" "mixed_states" {
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

# ========================================
# Test Case 6: Header ID Variations
# ========================================

# Test 6.1: Header IDs with underscores
resource "cloudflare_managed_transforms" "underscores" {
  zone_id = var.cloudflare_zone_id


  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = [{
    id      = "remove_x-powered-by_header"
    enabled = true
  }]
}

# Test 6.2: Header IDs with dashes
resource "cloudflare_managed_transforms" "dashes" {
  zone_id = var.cloudflare_zone_id

  managed_request_headers = [{
    id      = "add-custom-header"
    enabled = true
  }]
  managed_response_headers = []
}

# ========================================
# Test Case 7: Lifecycle Meta-Arguments
# ========================================

# Test 7.1: Resource with lifecycle
resource "cloudflare_managed_transforms" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id


  lifecycle {
    create_before_destroy = true
  }
  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = []
}

# Test 7.2: Resource with prevent_destroy
resource "cloudflare_managed_transforms" "prevent_destroy" {
  zone_id = var.cloudflare_zone_id


  lifecycle {
    prevent_destroy = false
  }
  managed_request_headers = []
  managed_response_headers = [{
    id      = "add_security_headers"
    enabled = true
  }]
}

# ========================================
# Test Case 8: Comments Preservation
# ========================================

# Test 8.1: Resource with inline comments
resource "cloudflare_managed_transforms" "with_comments" {
  zone_id = var.cloudflare_zone_id # Primary zone


  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = [{
    id      = "add_security_headers"
    enabled = true
  }]
}

# ========================================
# Test Case 9: Count with Index
# ========================================

variable "header_count" {
  type    = number
  default = 2
}

# Test 9.1: Resource with count
resource "cloudflare_managed_transforms" "with_count" {
  count = var.header_count

  zone_id = var.cloudflare_zone_id

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = count.index == 0
  }]
  managed_response_headers = []
}

# ========================================
# Test Case 10: String Interpolation
# ========================================

variable "environment" {
  type    = string
  default = "test"
}

# Test 10.1: Using string interpolation (for resource naming only)
resource "cloudflare_managed_transforms" "interpolated" {
  zone_id = var.cloudflare_zone_id

  managed_request_headers = [{
    id      = "add_visitor_location_headers"
    enabled = var.environment == "production"
  }]
  managed_response_headers = []
}

# ========================================
# Test Case 11: Ternary Operators
# ========================================

# Test 11.1: Ternary in enabled field
resource "cloudflare_managed_transforms" "ternary" {
  zone_id = var.cloudflare_zone_id

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = var.enable_security_headers ? true : false
  }]
  managed_response_headers = []
}

# ========================================
# Test Case 12: for_each with toset
# ========================================

variable "zones_list" {
  type    = list(string)
  default = ["zone_a", "zone_b"]
}

# Test 12.1: for_each with toset conversion
resource "cloudflare_managed_transforms" "from_list" {
  for_each = toset(var.zones_list)

  zone_id = var.cloudflare_zone_id

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = []
}

# ========================================
# Summary: 20+ resource instances total
# ========================================
# - Basic configurations: 5 resources
# - Conditional (count): 1 resource (potentially 0-1 instances)
# - for_each with maps: 2 resources (2 instances each = 4 instances)
# - Locals: 3 resources (2 for multi_zone = 4 instances)
# - Boolean edge cases: 3 resources
# - Header ID variations: 2 resources
# - Lifecycle: 2 resources
# - Comments: 1 resource
# - Count: 1 resource (2 instances)
# - Interpolation: 1 resource
# - Ternary: 1 resource
# - toset: 1 resource (2 instances)
# Total: ~25+ instances covering all Terraform patterns
