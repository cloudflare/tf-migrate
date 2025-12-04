variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Resource-specific variables with defaults
variable "policy_prefix" {
  type    = string
  default = "test"
}

variable "enable_security_policies" {
  type    = bool
  default = true
}

variable "policy_configs" {
  type = map(object({
    action      = string
    precedence  = number
    description = string
  }))
  default = {
    allow_internal = {
      action      = "allow"
      precedence  = 100
      description = "Allow internal traffic"
    }
    block_malware = {
      action      = "block"
      precedence  = 200
      description = "Block known malware domains"
    }
    audit_api = {
      action      = "allow"
      precedence  = 300
      description = "Audit API traffic"
    }
  }
}

# Locals with multiple values
locals {
  common_account_id         = var.cloudflare_account_id
  policy_name_prefix        = "${var.policy_prefix}-policy"
  dns_filter                = ["dns"]
  http_filter               = ["http"]
  l4_filter                 = ["l4"]
  common_traffic_expression = "any(dns.domains[*] == \"example.com\")"
  block_reason              = "Access blocked by company policy - ${var.policy_prefix}"
}

# ============================================================================
# Pattern Group 1: Basic Resources (Edge Cases)
# ============================================================================

# 1. Minimal gateway policy
resource "cloudflare_zero_trust_gateway_policy" "minimal" {
  account_id  = local.common_account_id
  name        = "cftftest Minimal Policy"
  description = "Basic block policy"
  precedence  = 100
  action      = "block"
  filters     = local.dns_filter
  traffic     = local.common_traffic_expression
}

# 2. Maximal policy with all settings
resource "cloudflare_zero_trust_gateway_policy" "maximal" {
  account_id  = var.cloudflare_account_id
  name        = "${local.policy_name_prefix}-maximal"
  description = "Policy with all possible settings"
  precedence  = 150
  action      = "block"
  enabled     = true
  filters     = local.dns_filter
  traffic     = "any(dns.domains[*] in {\"blocked.example.com\" \"malware.example.com\"})"

  rule_settings = {
    block_page_enabled = true
    override_ips       = ["1.1.1.1", "1.0.0.1"]
    ip_categories      = true
    block_reason       = local.block_reason
  }
}

# 3. Policy with rule_settings and field renames
resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Block Policy with Settings"
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

# 4. Policy with nested blocks requiring transformation
resource "cloudflare_zero_trust_gateway_policy" "with_nested_blocks" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest L4 Override Policy"
  description = "Policy with L4 override and notification"
  precedence  = 300
  action      = "l4_override"
  enabled     = true
  filters     = local.l4_filter
  traffic     = "net.dst.ip == 93.184.216.34"

  rule_settings = {
    l4override = {
      ip   = "192.168.1.100"
      port = 8080
    }
  }
}

# 5. Complex policy with multiple nested structures
resource "cloudflare_zero_trust_gateway_policy" "complex" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Complex Policy"
  description = "Policy with many nested settings"
  precedence  = 400
  action      = "allow"
  enabled     = true
  filters     = local.http_filter
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

# 6. Simple allow policy for testing rule_settings
resource "cloudflare_zero_trust_gateway_policy" "simple_resolver" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Simple Allow Policy"
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

# ============================================================================
# Pattern Group 2: for_each with Maps
# ============================================================================

# 7-9. Resources created with for_each over map
resource "cloudflare_zero_trust_gateway_policy" "policy_configs" {
  for_each = var.policy_configs

  account_id  = var.cloudflare_account_id
  name        = "${local.policy_name_prefix}-${each.key}"
  description = each.value.description
  precedence  = each.value.precedence + 5000 # Add offset to avoid conflicts
  action      = each.value.action
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"${each.key}.example.com\")"
}

# ============================================================================
# Pattern Group 3: for_each with Sets
# ============================================================================

# 10-13. Resources created with for_each over set
resource "cloudflare_zero_trust_gateway_policy" "environment_policies" {
  for_each = toset(["staging", "production", "development", "testing"])

  account_id  = var.cloudflare_account_id
  name        = "${local.policy_name_prefix}-${each.value}"
  description = "Policy for ${each.value} environment"
  precedence  = 600 + index(["staging", "production", "development", "testing"], each.value)
  action      = "allow"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"${each.value}.example.com\")"
}

# ============================================================================
# Pattern Group 4: count-based Resources
# ============================================================================

# 14-16. Resources created with count
resource "cloudflare_zero_trust_gateway_policy" "tiered_policies" {
  count = 3

  account_id  = var.cloudflare_account_id
  name        = "${local.policy_name_prefix}-tier-${count.index}"
  description = "Tier ${count.index} policy"
  precedence  = 700 + (count.index * 10)
  action      = count.index == 0 ? "block" : "allow"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"tier${count.index}.example.com\")"
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

# 17. Conditionally created policy
resource "cloudflare_zero_trust_gateway_policy" "conditional_enabled" {
  count = var.enable_security_policies ? 1 : 0

  account_id  = var.cloudflare_account_id
  name        = "cftftest Security Policy - Enabled"
  description = "This policy is conditionally created"
  precedence  = 800
  action      = "block"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] in {\"malware.example.com\" \"phishing.example.com\"})"

  rule_settings = {
    block_page_enabled = true
    block_reason       = "This domain is blocked for security reasons"
  }
}

# 18. Conditionally not created policy
resource "cloudflare_zero_trust_gateway_policy" "conditional_disabled" {
  count = var.enable_security_policies ? 0 : 1

  account_id  = var.cloudflare_account_id
  name        = "cftftest Security Policy - Disabled"
  description = "This policy is conditionally not created"
  precedence  = 900
  action      = "allow"
  enabled     = false
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"insecure.example.com\")"
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# ============================================================================

# 19. Policy using join() function
resource "cloudflare_zero_trust_gateway_policy" "with_join" {
  account_id  = var.cloudflare_account_id
  name        = join("-", [local.policy_name_prefix, "joined", "policy"])
  description = "Policy created with join function"
  precedence  = 1000
  action      = "allow"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"${join(".", ["subdomain", "example", "com"])}\")"
}

# 20. Policy using string interpolation
resource "cloudflare_zero_trust_gateway_policy" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  name        = "Policy for account ${var.cloudflare_account_id}"
  description = "Description with ${local.policy_name_prefix} interpolation"
  precedence  = 1100
  action      = "allow"
  enabled     = true
  filters     = ["http"]
  traffic     = "http.request.host == \"${var.policy_prefix}.example.com\""
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ============================================================================

# 21. Policy with lifecycle block
resource "cloudflare_zero_trust_gateway_policy" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Protected Policy"
  description = "Policy with lifecycle settings"
  precedence  = 1200
  action      = "block"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"protected.example.com\")"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}

# 22. Policy with prevent_destroy (set to false for testing)
resource "cloudflare_zero_trust_gateway_policy" "with_prevent_destroy" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Important Policy"
  description = "Critical policy"
  precedence  = 1300
  action      = "block"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"critical.example.com\")"

  lifecycle {
    prevent_destroy = false # Set to false for testing
  }

  rule_settings = {
    block_page_enabled = true
    block_reason       = "Critical domain blocked"
  }
}

# ============================================================================
# Pattern Group 8: Additional Edge Cases & Action Types
# ============================================================================

# 23. L4 Override policy with specific settings
resource "cloudflare_zero_trust_gateway_policy" "l4_override_detailed" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest L4 Override Detailed"
  description = "Detailed L4 override configuration"
  precedence  = 1400
  action      = "l4_override"
  enabled     = true
  filters     = ["l4"]
  traffic     = "net.dst.ip == 203.0.113.1"

  rule_settings = {
    l4override = {
      ip   = "10.0.0.1"
      port = 9090
    }
  }
}

# 24. Policy with audit SSH settings
resource "cloudflare_zero_trust_gateway_policy" "with_audit_ssh" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest SSH Audit Policy"
  description = "Policy with SSH auditing enabled"
  precedence  = 1500
  action      = "allow"
  enabled     = true
  filters     = ["l4"]
  traffic     = "net.dst.port == 22"

  rule_settings = {
    audit_ssh = {
      command_logging = true
    }
  }
}

# 25. Policy with check_session settings
resource "cloudflare_zero_trust_gateway_policy" "with_check_session" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Session Check Policy"
  description = "Policy with session validation"
  precedence  = 1600
  action      = "allow"
  enabled     = true
  filters     = ["http"]
  traffic     = "http.request.host == \"secure.example.com\""

  rule_settings = {
    check_session = {
      enforce  = true
      duration = "12h"
    }
  }
}

# 26. Policy with BISO admin controls
resource "cloudflare_zero_trust_gateway_policy" "with_biso" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest BISO Controls Policy"
  description = "Policy with browser isolation controls"
  precedence  = 1700
  action      = "isolate"
  enabled     = true
  filters     = ["http"]
  traffic     = "http.request.host == \"isolated.example.com\""

  rule_settings = {
    biso_admin_controls = {
      version          = "v2"
      disable_printing = true
      disable_download = true
    }
  }
}

# 27. Policy with payload logging
resource "cloudflare_zero_trust_gateway_policy" "with_payload_log" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Payload Logging Policy"
  description = "Policy with payload logging enabled"
  precedence  = 1800
  action      = "allow"
  enabled     = true
  filters     = ["http"]
  traffic     = "http.request.uri matches \".*/api/.*\""

  rule_settings = {
    payload_log = {
      enabled = true
    }
  }
}

# 28. Policy with override IPs
resource "cloudflare_zero_trust_gateway_policy" "with_override_ips" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest DNS Override Policy"
  description = "Policy with custom DNS servers"
  precedence  = 1900
  action      = "allow"
  enabled     = true
  filters     = ["dns"]
  traffic     = "any(dns.domains[*] == \"override.example.com\")"

  rule_settings = {
    override_ips  = ["1.1.1.1", "8.8.8.8", "9.9.9.9"]
    ip_categories = false
  }
}

# Total: 28 base resources + 3 from for_each map + 4 from for_each set + 3 from count = 38 instances
