# Zone Settings Migration Test - Safe settings for all plans
# Covers migration patterns without plan-restricted features

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

locals {
  primary_zone_id = var.cloudflare_zone_id
  cache_ttls      = [14400, 28800, 43200]
}




# Test Cases 4-6: Removed - Invalid Pattern
# Multiple zone_settings_override resources for the same zone will conflict
# In v4, cloudflare_zone_settings_override manages ALL settings for a zone
# Having multiple such resources causes them to overwrite each other
# This pattern is not supported and should not be used

# Test Case 7: Conditional creation
locals {
  enable_advanced_settings = true
  enable_test_settings     = false
}








resource "cloudflare_zone_setting" "minimal_always_online" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "always_online"
  value      = "on"
}
resource "cloudflare_zone_setting" "minimal_brotli" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "brotli"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_integers_browser_cache_ttl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "browser_cache_ttl"
  value      = 14400
}
resource "cloudflare_zone_setting" "with_integers_challenge_ttl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "challenge_ttl"
  value      = 1800
}
resource "cloudflare_zone_setting" "with_security_header_ssl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "ssl"
  value      = "flexible"
}
resource "cloudflare_zone_setting" "with_security_header_security_header" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "security_header"
  value = {
    strict_transport_security = {
      enabled            = true
      include_subdomains = true
      max_age            = 86400
      nosniff            = true
      preload            = true
    }
  }
}
resource "cloudflare_zone_setting" "conditional_enabled_rocket_loader" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "rocket_loader"
  value      = "on"
  count      = local.enable_advanced_settings ? 1 : 0
}
resource "cloudflare_zone_setting" "conditional_enabled_websockets" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "websockets"
  value      = "on"
  count      = local.enable_advanced_settings ? 1 : 0
}
resource "cloudflare_zone_setting" "conditional_disabled_browser_check" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "browser_check"
  value      = "on"
  count      = local.enable_test_settings ? 1 : 0
}
resource "cloudflare_zone_setting" "with_functions_browser_cache_ttl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "browser_cache_ttl"
  value      = lookup({ "default" = 14400, "custom" = 28800 }, "default")
}
resource "cloudflare_zone_setting" "with_functions_cache_level" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "cache_level"
  value      = "aggressive"
}
resource "cloudflare_zone_setting" "with_interpolation_automatic_https_rewrites" {
  zone_id    = local.primary_zone_id
  setting_id = "automatic_https_rewrites"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_interpolation_min_tls_version" {
  zone_id    = local.primary_zone_id
  setting_id = "min_tls_version"
  value      = "1.2"
}
resource "cloudflare_zone_setting" "with_lifecycle_always_online" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "always_online"
  value      = "on"
  lifecycle {
    create_before_destroy = true
  }
}
resource "cloudflare_zone_setting" "with_lifecycle_ipv6" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "ipv6"
  value      = "on"
  lifecycle {
    create_before_destroy = true
  }
}
resource "cloudflare_zone_setting" "with_ignore_changes_email_obfuscation" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "email_obfuscation"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_ignore_changes_server_side_exclude" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "server_side_exclude"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_name_mapping_http2" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "http2"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_name_mapping_http3" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "http3"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_deprecated_always_online" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "always_online"
  value      = "on"
}
resource "cloudflare_zone_setting" "with_deprecated_brotli" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "brotli"
  value      = "on"
}
