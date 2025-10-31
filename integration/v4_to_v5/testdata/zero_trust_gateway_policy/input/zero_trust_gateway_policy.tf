# Test input for cloudflare_teams_rule → cloudflare_zero_trust_gateway_policy migration

# Minimal policy
resource "cloudflare_teams_rule" "minimal" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Minimal Gateway Policy"
  description = "A minimal policy for testing"
  precedence  = 100
  action      = "allow"
}

# Policy with rule settings and field renames
resource "cloudflare_teams_rule" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Policy with Settings"
  description = "Testing field renames in rule_settings"
  precedence  = 200
  action      = "block"
  enabled     = true
  
  rule_settings {
    block_page_enabled = true
    block_page_reason  = "This website is blocked by company policy"
    
    notification_settings {
      enabled     = true
      message     = "You have violated the acceptable use policy"
      support_url = "https://support.example.com/blocked"
    }
  }
}

# Policy with BISO admin controls
resource "cloudflare_teams_rule" "biso_controls" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "BISO Admin Controls"
  description = "Testing BISO admin control field renames"
  precedence  = 300
  action      = "isolate"
  
  rule_settings {
    biso_admin_controls {
      disable_printing               = true
      disable_copy_paste            = true
      disable_download              = false
      disable_keyboard              = false
      disable_upload                = true
      disable_clipboard_redirection = true
    }
    
    check_session {
      enforce  = true
      duration = 300
    }
  }
}

# Complex policy with DNS resolvers and filters
resource "cloudflare_teams_rule" "complex" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "Complex Gateway Policy"
  description    = "Testing all features"
  precedence     = 400
  action         = "allow"
  enabled        = true
  
  filters        = ["dns", "http", "l4"]
  traffic        = "any(http.request.uri.path contains \"/admin\")"
  identity       = "any(identity.groups.name == \"admins\")"
  device_posture = "any(device_posture.checks.passed == true)"
  
  rule_settings {
    block_page_enabled = false
    
    dns_resolvers {
      ipv4 {
        ip   = "1.1.1.1"
        port = 53
      }
      ipv4 {
        ip   = "1.0.0.1"
        port = 53
      }
      ipv6 {
        ip   = "2606:4700:4700::1111"
        port = 53
      }
      ipv6 {
        ip   = "2606:4700:4700::1001"
        port = 5053
      }
    }
    
    egress {
      ipv4          = "198.51.100.1"
      ipv6          = "2001:db8::1"
      ipv4_fallback = "203.0.113.1"
    }
    
    l4override {
      ip   = "10.0.0.1"
      port = 8080
    }
    
    payload_log {
      enabled = true
    }
    
    untrusted_cert {
      action = "error"
    }
  }
}

# Policy with empty description (edge case)
resource "cloudflare_teams_rule" "empty_desc" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Empty Description Policy"
  description = ""
  precedence  = 0
  action      = "allow"
}