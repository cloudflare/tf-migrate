# Test Case 1: Minimal gateway policy
resource "cloudflare_teams_rule" "minimal" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Minimal Policy"
  description = "Basic block policy"
  precedence  = 100
  action      = "block"
}

# Test Case 2: Policy with rule_settings and field renames
resource "cloudflare_teams_rule" "with_settings" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Block Policy with Settings"
  description = "Policy with custom block page"
  precedence = 200
  action     = "block"
  enabled    = true
  filters    = ["dns"]

  rule_settings {
    block_page_enabled = true
    block_page_reason  = "Access to this site is blocked by company policy"
    override_ips       = ["1.1.1.1", "1.0.0.1"]
    ip_categories      = true
  }
}

# Test Case 3: Policy with nested blocks requiring transformation
resource "cloudflare_teams_rule" "with_nested_blocks" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "L4 Override Policy"
  description = "Policy with L4 override and notification"
  precedence = 300
  action     = "l4_override"
  enabled    = true
  filters    = ["l4"]

  rule_settings {
    l4override {
      ip   = "192.168.1.100"
      port = 8080
    }

    notification_settings {
      enabled = true
      message = "This connection has been redirected"
      support_url = "https://support.example.com"
    }
  }
}

# Test Case 4: Complex policy with multiple nested structures
resource "cloudflare_teams_rule" "complex" {
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "Complex Policy"
  description    = "Policy with many nested settings"
  precedence     = 400
  action         = "allow"
  enabled        = true
  filters        = ["http"]
  traffic        = "http.request.uri matches \".*api.*\""
  identity       = "any(identity.groups[*] in {\"developers\"})"
  device_posture = ""

  rule_settings {
    audit_ssh {
      command_logging = true
    }

    check_session {
      enforce  = true
      duration = "24h"
    }

    biso_admin_controls {
      version          = "v1"
      disable_printing = true
      disable_download = false
    }

    payload_log {
      enabled = true
    }

    untrusted_cert {
      action = "pass_through"
    }
  }
}

# Test Case 5: Simple dns_resolvers test
# Note: Complex nested repeated blocks (ipv4/ipv6) require custom HCL handling
# State transformation handles this correctly (see unit tests)
resource "cloudflare_teams_rule" "simple_resolver" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Resolve Policy"
  description = "Simple resolve policy"
  precedence  = 500
  action      = "resolve"
  enabled     = true
  filters     = ["dns"]
}
