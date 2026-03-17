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

import {
  to = cloudflare_zone_setting.minimal_always_online
  id = "${var.cloudflare_zone_id}/always_online"
}

resource "cloudflare_zone_setting" "minimal_brotli" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "brotli"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.minimal_brotli
  id = "${var.cloudflare_zone_id}/brotli"
}

removed {
  from = cloudflare_zone_settings_override.minimal
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.minimal'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.minimal'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.minimal_<setting>
}

resource "cloudflare_zone_setting" "with_integers_browser_cache_ttl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "browser_cache_ttl"
  value      = 14400
}

import {
  to = cloudflare_zone_setting.with_integers_browser_cache_ttl
  id = "${var.cloudflare_zone_id}/browser_cache_ttl"
}

resource "cloudflare_zone_setting" "with_integers_challenge_ttl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "challenge_ttl"
  value      = 1800
}

import {
  to = cloudflare_zone_setting.with_integers_challenge_ttl
  id = "${var.cloudflare_zone_id}/challenge_ttl"
}

removed {
  from = cloudflare_zone_settings_override.with_integers
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_integers'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_integers'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_integers_<setting>
}

resource "cloudflare_zone_setting" "with_security_header_ssl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "ssl"
  value      = "flexible"
}

import {
  to = cloudflare_zone_setting.with_security_header_ssl
  id = "${var.cloudflare_zone_id}/ssl"
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

import {
  to = cloudflare_zone_setting.with_security_header_security_header
  id = "${var.cloudflare_zone_id}/security_header"
}

removed {
  from = cloudflare_zone_settings_override.with_security_header
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_security_header'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_security_header'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_security_header_<setting>
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

removed {
  from = cloudflare_zone_settings_override.conditional_enabled
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.conditional_enabled[0]'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.conditional_enabled[0]'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.conditional_enabled_<setting>
}

resource "cloudflare_zone_setting" "conditional_disabled_browser_check" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "browser_check"
  value      = "on"
  count      = local.enable_test_settings ? 1 : 0
}

removed {
  from = cloudflare_zone_settings_override.conditional_disabled
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.conditional_disabled[0]'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.conditional_disabled[0]'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.conditional_disabled_<setting>
}

resource "cloudflare_zone_setting" "with_functions_browser_cache_ttl" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "browser_cache_ttl"
  value      = lookup({ "default" = 14400, "custom" = 28800 }, "default")
}

import {
  to = cloudflare_zone_setting.with_functions_browser_cache_ttl
  id = "${var.cloudflare_zone_id}/browser_cache_ttl"
}

resource "cloudflare_zone_setting" "with_functions_cache_level" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "cache_level"
  value      = "aggressive"
}

import {
  to = cloudflare_zone_setting.with_functions_cache_level
  id = "${var.cloudflare_zone_id}/cache_level"
}

removed {
  from = cloudflare_zone_settings_override.with_functions
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_functions'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_functions'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_functions_<setting>
}

resource "cloudflare_zone_setting" "with_interpolation_automatic_https_rewrites" {
  zone_id    = local.primary_zone_id
  setting_id = "automatic_https_rewrites"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_interpolation_automatic_https_rewrites
  id = "${local.primary_zone_id}/automatic_https_rewrites"
}

resource "cloudflare_zone_setting" "with_interpolation_min_tls_version" {
  zone_id    = local.primary_zone_id
  setting_id = "min_tls_version"
  value      = "1.2"
}

import {
  to = cloudflare_zone_setting.with_interpolation_min_tls_version
  id = "${local.primary_zone_id}/min_tls_version"
}

removed {
  from = cloudflare_zone_settings_override.with_interpolation
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_interpolation'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_interpolation'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_interpolation_<setting>
}

resource "cloudflare_zone_setting" "with_lifecycle_always_online" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "always_online"
  value      = "on"
  lifecycle {
    create_before_destroy = true
  }
}

import {
  to = cloudflare_zone_setting.with_lifecycle_always_online
  id = "${var.cloudflare_zone_id}/always_online"
}

resource "cloudflare_zone_setting" "with_lifecycle_ipv6" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "ipv6"
  value      = "on"
  lifecycle {
    create_before_destroy = true
  }
}

import {
  to = cloudflare_zone_setting.with_lifecycle_ipv6
  id = "${var.cloudflare_zone_id}/ipv6"
}

removed {
  from = cloudflare_zone_settings_override.with_lifecycle
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_lifecycle'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_lifecycle'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_lifecycle_<setting>
}

resource "cloudflare_zone_setting" "with_ignore_changes_email_obfuscation" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "email_obfuscation"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_ignore_changes_email_obfuscation
  id = "${var.cloudflare_zone_id}/email_obfuscation"
}

resource "cloudflare_zone_setting" "with_ignore_changes_server_side_exclude" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "server_side_exclude"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_ignore_changes_server_side_exclude
  id = "${var.cloudflare_zone_id}/server_side_exclude"
}

removed {
  from = cloudflare_zone_settings_override.with_ignore_changes
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_ignore_changes'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_ignore_changes'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_ignore_changes_<setting>
}

resource "cloudflare_zone_setting" "with_name_mapping_http2" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "http2"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_name_mapping_http2
  id = "${var.cloudflare_zone_id}/http2"
}

resource "cloudflare_zone_setting" "with_name_mapping_http3" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "http3"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_name_mapping_http3
  id = "${var.cloudflare_zone_id}/http3"
}

removed {
  from = cloudflare_zone_settings_override.with_name_mapping
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_name_mapping'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_name_mapping'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_name_mapping_<setting>
}

resource "cloudflare_zone_setting" "with_deprecated_always_online" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "always_online"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_deprecated_always_online
  id = "${var.cloudflare_zone_id}/always_online"
}

resource "cloudflare_zone_setting" "with_deprecated_brotli" {
  zone_id    = var.cloudflare_zone_id
  setting_id = "brotli"
  value      = "on"
}

import {
  to = cloudflare_zone_setting.with_deprecated_brotli
  id = "${var.cloudflare_zone_id}/brotli"
}

removed {
  from = cloudflare_zone_settings_override.with_deprecated
  lifecycle {
    destroy = false
  }
  # MIGRATION NOTES:
  # 1. Before running terraform plan/apply, remove the old state entry:
  #      terraform state rm 'cloudflare_zone_settings_override.with_deprecated'
  #    If inside a module named "mymod":
  #      terraform state rm 'module.mymod.cloudflare_zone_settings_override.with_deprecated'
  # 2. The import blocks above are only valid in the root Terraform module.
  #    If this file is a child module, move the import blocks to your root
  #    module and prefix 'to' with the module path:
  #      to = module.mymod.cloudflare_zone_setting.with_deprecated_<setting>
}
