# Comprehensive Integration Test for access_rule v4 → v5 Migration
# Target: 20-25 resource instances covering all patterns

# Pattern 1: Variables (no defaults - must be provided)
variable "cloudflare_account_id" {
  description = "Cloudflare account ID for testing"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# Pattern 2: Locals with resource-specific configuration
locals {
  blocked_countries = ["CN", "RU"]
  blocked_ips = [
    "198.51.100.1",
    "198.51.100.2",
    "198.51.100.3",
  ]
  ip_ranges = {
    suspicious_range_1 = "192.0.2.0/24"
    suspicious_range_2 = "198.51.100.0/24"
  }
  enable_ipv6_blocking = true
  enable_asn_blocking  = true
}

# Basic Examples - All Mode Types
resource "cloudflare_access_rule" "example_block" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block malicious IP"
  configuration = {
    target = "ip"
    value  = "198.51.100.10"
  }
}

resource "cloudflare_access_rule" "example_challenge" {
  account_id = var.cloudflare_account_id
  mode       = "challenge"
  notes      = "Challenge unknown country"
  configuration = {
    target = "country"
    value  = "XX"
  }
}

resource "cloudflare_access_rule" "example_whitelist" {
  zone_id = var.cloudflare_zone_id
  mode    = "whitelist"
  notes   = "Allow trusted IP"
  configuration = {
    target = "ip"
    value  = "203.0.113.100"
  }
}

resource "cloudflare_access_rule" "example_js_challenge" {
  account_id = var.cloudflare_account_id
  mode       = "js_challenge"
  notes      = "JS challenge for specific ASN"
  configuration = {
    target = "asn"
    value  = "AS13335"
  }
}

resource "cloudflare_access_rule" "example_managed_challenge" {
  zone_id = var.cloudflare_zone_id
  mode    = "managed_challenge"
  configuration = {
    target = "ip_range"
    value  = "198.51.100.0/24"
  }
}

# All Target Types Examples
resource "cloudflare_access_rule" "target_ip" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block specific IPv4"
  configuration = {
    target = "ip"
    value  = "192.0.2.1"
  }
}

resource "cloudflare_access_rule" "target_ip6" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block specific IPv6"
  configuration = {
    target = "ip6"
    value  = "2001:0db8:0000:0000:0000:0000:0000:0001"
  }
}

resource "cloudflare_access_rule" "target_ip_range" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block IP range"
  configuration = {
    target = "ip_range"
    value  = "203.0.113.0/24"
  }
}

resource "cloudflare_access_rule" "target_asn" {
  zone_id = var.cloudflare_zone_id
  mode    = "challenge"
  notes   = "Challenge Google ASN"
  configuration = {
    target = "asn"
    value  = "AS15169"
  }
}

resource "cloudflare_access_rule" "target_country" {
  zone_id = var.cloudflare_zone_id
  mode    = "block"
  notes   = "Block unknown country code"
  configuration = {
    target = "country"
    value  = "XX"
  }
}

# Pattern 3: for_each with map
resource "cloudflare_access_rule" "block_countries" {
  for_each = toset(local.blocked_countries)

  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block country: ${each.value}"
  configuration = {
    target = "country"
    value  = each.value
  }
}

# Pattern 4: for_each with set (converted from list)
resource "cloudflare_access_rule" "block_ips" {
  for_each = toset(local.blocked_ips)

  zone_id = var.cloudflare_zone_id
  mode    = "block"
  notes   = "Block IP from list: ${each.value}"
  configuration = {
    target = "ip"
    value  = each.value
  }
}

# Pattern 5: for_each with map (complex)
resource "cloudflare_access_rule" "block_ranges" {
  for_each = local.ip_ranges

  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block range: ${each.key} (${each.value})"
  configuration = {
    target = "ip_range"
    value  = each.value
  }
}

# Pattern 6: count-based resources
resource "cloudflare_access_rule" "backup_blocks" {
  count = 3

  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Backup block rule ${count.index + 1}"
  configuration = {
    target = "ip"
    value  = "198.51.100.${count.index + 20}"
  }
}

# Pattern 7: Conditional resource creation
resource "cloudflare_access_rule" "ipv6_conditional" {
  count = local.enable_ipv6_blocking ? 1 : 0

  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Conditional IPv6 block"
  configuration = {
    target = "ip6"
    value  = "2001:0db8:0000:0000:0000:0000:0000:0bad"
  }
}

resource "cloudflare_access_rule" "asn_conditional" {
  count = local.enable_asn_blocking ? 1 : 0

  zone_id = var.cloudflare_zone_id
  mode    = "challenge"
  notes   = "Conditional ASN challenge"
  configuration = {
    target = "asn"
    value  = "AS12345"
  }
}

# Pattern 8: Lifecycle meta-arguments
resource "cloudflare_access_rule" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Rule with lifecycle settings"

  lifecycle {
    create_before_destroy = true
  }
  configuration = {
    target = "ip"
    value  = "198.51.100.99"
  }
}

# Pattern 9: Terraform functions and string interpolation
resource "cloudflare_access_rule" "with_functions" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Created at: ${formatdate("YYYY-MM-DD", timestamp())}"
  configuration = {
    target = "ip"
    value  = "198.51.100.100"
  }
}

# Cross-reference pattern (referencing other resources)
resource "cloudflare_access_rule" "referenced" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Primary block rule"
  configuration = {
    target = "ip"
    value  = "198.51.100.200"
  }
}

resource "cloudflare_access_rule" "referencing" {
  zone_id = var.cloudflare_zone_id
  mode    = "challenge"
  notes   = "Related to ${cloudflare_access_rule.referenced.notes}"
  configuration = {
    target = "country"
    value  = "US"
  }
}

# Without notes field (optional field test)
resource "cloudflare_access_rule" "no_notes_1" {
  account_id = var.cloudflare_account_id
  mode       = "whitelist"
  configuration = {
    target = "ip"
    value  = "203.0.113.50"
  }
}

resource "cloudflare_access_rule" "no_notes_2" {
  zone_id = var.cloudflare_zone_id
  mode    = "challenge"
  configuration = {
    target = "ip"
    value  = "198.51.100.240"
  }
}

# Special characters in notes
resource "cloudflare_access_rule" "special_chars" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Block: \"suspicious\" IP with 'quotes' & special chars! #security"
  configuration = {
    target = "ip"
    value  = "198.51.100.250"
  }
}

# Multiple modes on similar configs (testing mode variations)
resource "cloudflare_access_rule" "mode_block" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "Mode: block"
  configuration = {
    target = "country"
    value  = "KP"
  }
}

resource "cloudflare_access_rule" "mode_challenge" {
  account_id = var.cloudflare_account_id
  mode       = "challenge"
  notes      = "Mode: challenge"
  configuration = {
    target = "country"
    value  = "IR"
  }
}

resource "cloudflare_access_rule" "mode_whitelist" {
  account_id = var.cloudflare_account_id
  mode       = "whitelist"
  notes      = "Mode: whitelist"
  configuration = {
    target = "country"
    value  = "US"
  }
}

# Edge case: Very long notes
resource "cloudflare_access_rule" "long_notes" {
  zone_id = var.cloudflare_zone_id
  mode    = "block"
  notes   = "This is a very long note field that tests the handling of extended text content. It includes multiple sentences and various punctuation marks! Does the migration handle this correctly? We'll find out."
  configuration = {
    target = "ip"
    value  = "198.51.100.254"
  }
}
