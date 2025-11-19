variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test Case 1: Minimal gateway policy
resource "cloudflare_zero_trust_gateway_policy" "minimal" {
  account_id  = var.cloudflare_account_id
  name        = "Minimal Policy"
  description = "Basic block policy"
  precedence  = 100
  action      = "block"
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"example.com\")"
}

# Test Case 2: Policy with rule_settings and field renames
resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
  account_id  = var.cloudflare_account_id
  name        = "Block Policy with Settings"
  description = "Policy with custom block page"
  precedence  = 200
  action      = "block"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] in {\"blocked.example.com\" \"malware.example.com\"})"

  rule_settings = {
    block_page_enabled = true
    override_ips       = ["1.1.1.1", "1.0.0.1"]
    ip_categories      = true
    block_reason       = "Access to this site is blocked by company policy"
  }
}

# Test Case 3: Policy with nested blocks requiring transformation
resource "cloudflare_zero_trust_gateway_policy" "with_nested_blocks" {
  account_id  = var.cloudflare_account_id
  name        = "L4 Override Policy"
  description = "Policy with L4 override and notification"
  precedence  = 300
  action      = "l4_override"
  enabled     = true
  filters     = ["l4"]
  traffic     = "net.dst.ip == 93.184.216.34"

  rule_settings = {
    l4override = {
      ip   = "192.168.1.100"
      port = 8080
    }
  }
}

# Test Case 4: Complex policy with multiple nested structures
resource "cloudflare_zero_trust_gateway_policy" "complex" {
  account_id  = var.cloudflare_account_id
  name        = "Complex Policy"
  description = "Policy with many nested settings"
  precedence  = 400
  action      = "allow"
  enabled     = true
  filters     = ["http"]
  traffic     = "http.request.uri matches \".*api.*\""

  rule_settings = {
    audit_ssh = {
      command_logging = true
    }
    biso_admin_controls = {
      version          = "v1"
      disable_printing = true
      disable_download = false
    }
    check_session = {
      enforce  = true
      duration = "24h"
    }
    payload_log = {
      enabled = true
    }
  }
}

# Test Case 5: Simple allow policy for testing rule_settings
# Note: Complex nested repeated blocks (ipv4/ipv6) require custom HCL handling
# State transformation handles this correctly (see unit tests)
resource "cloudflare_zero_trust_gateway_policy" "simple_resolver" {
  account_id  = var.cloudflare_account_id
  name        = "Simple Allow Policy"
  description = "Simple allow policy with settings"
  precedence  = 500
  action      = "allow"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"allowed.example.com\")"

  rule_settings = {
    block_page_enabled = false
  }
}
