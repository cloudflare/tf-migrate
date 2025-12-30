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

# Test Case 1: Basic settings
resource "cloudflare_zone_settings_override" "minimal" {
  zone_id = var.cloudflare_zone_id

  settings {
    always_online = "on"
    brotli        = "on"
  }
}

# Test Case 2: Integer settings
resource "cloudflare_zone_settings_override" "with_integers" {
  zone_id = var.cloudflare_zone_id

  settings {
    browser_cache_ttl = 14400
    challenge_ttl     = 1800
  }
}

# Test Case 3: Security settings  
resource "cloudflare_zone_settings_override" "with_security_header" {
  zone_id = var.cloudflare_zone_id

  settings {
    ssl = "flexible"

    security_header {
      enabled            = true
      max_age            = 86400
      include_subdomains = true
      preload            = true
      nosniff            = true
    }
  }
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

resource "cloudflare_zone_settings_override" "conditional_enabled" {
  count = local.enable_advanced_settings ? 1 : 0

  zone_id = var.cloudflare_zone_id

  settings {
    rocket_loader = "on"
    websockets    = "on"
  }
}

resource "cloudflare_zone_settings_override" "conditional_disabled" {
  count = local.enable_test_settings ? 1 : 0

  zone_id = var.cloudflare_zone_id

  settings {
    browser_check = "on"
  }
}

# Test Case 8: Terraform functions
resource "cloudflare_zone_settings_override" "with_functions" {
  zone_id = var.cloudflare_zone_id

  settings {
    cache_level       = "aggressive"
    browser_cache_ttl = lookup({ "default" = 14400, "custom" = 28800 }, "default")
  }
}

# Test Case: Interpolation (simplified - removed tls_1_3 due to API bug)
resource "cloudflare_zone_settings_override" "with_interpolation" {
  zone_id = local.primary_zone_id

  settings {
    min_tls_version          = "1.2"
    automatic_https_rewrites = "on"
  }
}

# Test Case 9: Lifecycle meta-arguments
resource "cloudflare_zone_settings_override" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id

  settings {
    always_online = "on"
    ipv6          = "on"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_zone_settings_override" "with_ignore_changes" {
  zone_id = var.cloudflare_zone_id

  settings {
    server_side_exclude = "on"
    email_obfuscation   = "on"
  }

  lifecycle {
    ignore_changes = [settings[0].server_side_exclude]
  }
}

# Test Case 10: Name mapping (removed zero_rtt → 0rtt due to plan restrictions)
resource "cloudflare_zone_settings_override" "with_name_mapping" {
  zone_id = var.cloudflare_zone_id

  settings {
    http2    = "on"
    http3    = "on"
  }
}

# Test Case 11: Deprecated settings filter
resource "cloudflare_zone_settings_override" "with_deprecated" {
  zone_id = var.cloudflare_zone_id

  settings {
    always_online = "on"
    universal_ssl = "on"  # Should be filtered during migration
    brotli        = "on"
  }
}
