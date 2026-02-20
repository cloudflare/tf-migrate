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

# Test case 1: Basic GET operation
resource "cloudflare_api_shield_operation" "get_users" {
  zone_id  = var.cloudflare_zone_id
  method   = "GET"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users"
}

# Test case 2: POST operation with single parameter
resource "cloudflare_api_shield_operation" "create_user" {
  zone_id  = var.cloudflare_zone_id
  method   = "POST"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users"
}

# Test case 3: GET operation with single parameter
resource "cloudflare_api_shield_operation" "get_user_by_id" {
  zone_id  = var.cloudflare_zone_id
  method   = "GET"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/{var1}"
}

# Test case 4: PUT operation with parameter
resource "cloudflare_api_shield_operation" "update_user" {
  zone_id  = var.cloudflare_zone_id
  method   = "PUT"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/{var1}"
}

# Test case 5: DELETE operation with parameter
resource "cloudflare_api_shield_operation" "delete_user" {
  zone_id  = var.cloudflare_zone_id
  method   = "DELETE"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/{var1}"
}

# Test case 6: PATCH operation with multi-parameter endpoint
resource "cloudflare_api_shield_operation" "update_user_post" {
  zone_id  = var.cloudflare_zone_id
  method   = "PATCH"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/{var1}/posts/{var2}"
}

# Test case 7: GET operation with multi-parameter endpoint
resource "cloudflare_api_shield_operation" "get_user_comment" {
  zone_id  = var.cloudflare_zone_id
  method   = "GET"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/{var1}/posts/{var2}/comments/{var3}"
}

# Test case 8: POST operation with subdomain host
resource "cloudflare_api_shield_operation" "v2_create_resource" {
  zone_id  = var.cloudflare_zone_id
  method   = "POST"
  host     = "v2.api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/resources"
}

# Test case 9: GET operation with health check endpoint
resource "cloudflare_api_shield_operation" "staging_get_health" {
  zone_id  = var.cloudflare_zone_id
  method   = "GET"
  host     = "api.staging.${var.cloudflare_domain}"
  endpoint = "/cftftest/health"
}

# Test case 10: OPTIONS operation (for CORS preflight)
resource "cloudflare_api_shield_operation" "options_users" {
  zone_id  = var.cloudflare_zone_id
  method   = "OPTIONS"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users"
}

# Test case 11: HEAD operation
resource "cloudflare_api_shield_operation" "head_users" {
  zone_id  = var.cloudflare_zone_id
  method   = "HEAD"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users"
}

# Test case 12: POST with versioned API path
resource "cloudflare_api_shield_operation" "v1_create_order" {
  zone_id  = var.cloudflare_zone_id
  method   = "POST"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/v1/orders"
}

# Test case 13: GET with nested resource path
resource "cloudflare_api_shield_operation" "get_account_settings" {
  zone_id  = var.cloudflare_zone_id
  method   = "GET"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/v1/accounts/{var1}/settings"
}

# Test case 14: PATCH with query-like endpoint
resource "cloudflare_api_shield_operation" "search_users" {
  zone_id  = var.cloudflare_zone_id
  method   = "GET"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/search"
}

# Test case 15: DELETE with nested resource
resource "cloudflare_api_shield_operation" "delete_user_session" {
  zone_id  = var.cloudflare_zone_id
  method   = "DELETE"
  host     = "api.${var.cloudflare_domain}"
  endpoint = "/cftftest/api/users/{var1}/sessions/{var2}"
}
