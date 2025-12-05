variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Resource-specific variables with defaults
variable "list_prefix" {
  type    = string
  default = "test"
}

variable "enable_security_lists" {
  type    = bool
  default = true
}

variable "security_domains" {
  type = map(object({
    type        = string
    description = string
  }))
  default = {
    malware = {
      type        = "DOMAIN"
      description = "Known malware domains"
    }
    phishing = {
      type        = "DOMAIN"
      description = "Phishing sites"
    }
    spam = {
      type        = "DOMAIN"
      description = "Spam domains"
    }
  }
}

# Locals with multiple values
locals {
  name_prefix       = "cftftest"
  common_account_id = var.cloudflare_account_id
  list_name_prefix  = "${var.list_prefix}-list"
  ip_ranges = [
    "192.168.1.0/24",
    "10.0.0.0/8",
    "172.16.0.0/12"
  ]
  common_description = "Managed by Terraform - ${var.list_prefix}"
}

# ============================================================================
# Pattern Group 1: Basic Resources (Edge Cases)
# ============================================================================

# 1. Minimal resource - only required fields
resource "cloudflare_teams_list" "minimal" {
  account_id = local.common_account_id
  name = "${local.name_prefix} Minimal List"
  type       = "IP"
  items      = ["192.168.1.1"]
}

# 2. Maximal resource - all fields populated
resource "cloudflare_teams_list" "maximal" {
  account_id  = local.common_account_id
  name        = "${local.list_name_prefix}-maximal"
  type        = "DOMAIN"
  description = local.common_description
  items       = ["example.com", "test.example.com", "api.example.com"]

  items_with_description {
    value       = "secure.example.com"
    description = "Secure subdomain"
  }

  items_with_description {
    value       = "admin.example.com"
    description = "Admin portal"
  }
}

# 3. Empty list - edge case
resource "cloudflare_teams_list" "empty" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Empty List"
  type        = "URL"
  description = "Empty list for testing"
  items       = []
}

# 4. Basic IP list with simple items array
resource "cloudflare_teams_list" "ip_list" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix} IP Allowlist"
  type       = "IP"
  items      = ["192.168.1.1", "192.168.1.2", "10.0.0.0/8"]
}

# 5. Domain list with items_with_description blocks
resource "cloudflare_teams_list" "domain_list" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Allowed Domains"
  type        = "DOMAIN"
  description = "Company approved domains"

  items_with_description {
    value       = "example.com"
    description = "Main company domain"
  }

  items_with_description {
    value       = "api.example.com"
    description = "API subdomain"
  }

  items_with_description {
    value       = "test.example.com"
    description = "Testing environment"
  }
}

# 6. Mixed list with both items and items_with_description
resource "cloudflare_teams_list" "email_list" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix} VIP Emails"
  type       = "EMAIL"
  items      = ["admin@example.com", "security@example.com"]

  items_with_description {
    value       = "ceo@example.com"
    description = "CEO email address"
  }

  items_with_description {
    value       = "cto@example.com"
    description = "CTO email address"
  }
}

# ============================================================================
# Pattern Group 2: for_each with Maps
# ============================================================================

# 7-9. Resources created with for_each over map
resource "cloudflare_teams_list" "security_domains" {
  for_each = var.security_domains

  account_id  = var.cloudflare_account_id
  name        = "${local.list_name_prefix}-${each.key}"
  type        = each.value.type
  description = each.value.description
  items       = ["${each.key}.example.com", "www.${each.key}.example.com"]
}

# ============================================================================
# Pattern Group 3: for_each with Sets
# ============================================================================

# 10-13. Resources created with for_each over set
resource "cloudflare_teams_list" "list_types" {
  for_each = toset(["staging", "production", "development", "testing"])

  account_id  = var.cloudflare_account_id
  name        = "${local.list_name_prefix}-${each.value}"
  type        = "DOMAIN"
  description = "List for ${each.value} environment"
  items       = ["${each.value}.example.com"]
}

# ============================================================================
# Pattern Group 4: count-based Resources
# ============================================================================

# 14-16. Resources created with count
resource "cloudflare_teams_list" "ip_ranges" {
  count = 3

  account_id  = var.cloudflare_account_id
  name        = "${local.list_name_prefix}-range-${count.index}"
  type        = "IP"
  description = "IP range ${count.index}"
  items       = [local.ip_ranges[count.index]]
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

# 17. Conditionally created resource
resource "cloudflare_teams_list" "conditional_enabled" {
  count = var.enable_security_lists ? 1 : 0

  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Security List - Enabled"
  type        = "DOMAIN"
  description = "This list is conditionally created"
  items       = ["secure.example.com", "protected.example.com"]
}

# 18. Conditionally not created resource
resource "cloudflare_teams_list" "conditional_disabled" {
  count = var.enable_security_lists ? 0 : 1

  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Security List - Disabled"
  type        = "DOMAIN"
  description = "This list is conditionally not created"
  items       = ["insecure.example.com"]
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# ============================================================================

# 19. Resource using join() function
resource "cloudflare_teams_list" "with_join" {
  account_id  = var.cloudflare_account_id
  name        = join("-", [local.list_name_prefix, "joined", "name"])
  type        = "DOMAIN"
  description = "List created with join function"
  items       = ["${join(".", ["subdomain", "example", "com"])}"]
}

# 20. Resource using string interpolation
resource "cloudflare_teams_list" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  name        = "List for account ${var.cloudflare_account_id}"
  type        = "URL"
  description = "Description with ${local.list_name_prefix} interpolation"
  items       = ["https://${var.list_prefix}.example.com/path"]
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ============================================================================

# 21. Resource with lifecycle block
resource "cloudflare_teams_list" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Protected List"
  type        = "IP"
  description = "List with lifecycle settings"
  items       = ["203.0.113.0/24"]

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
}

# 22. Resource with prevent_destroy (set to false for testing)
resource "cloudflare_teams_list" "with_prevent_destroy" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Important List"
  type        = "DOMAIN"
  description = "Critical list"
  items       = ["critical.example.com"]

  lifecycle {
    prevent_destroy = false  # Set to false for testing
  }
}

# ============================================================================
# Pattern Group 8: Additional Edge Cases & Type Coverage
# ============================================================================

# 23. URL list with only items_with_description
resource "cloudflare_teams_list" "url_list" {
  account_id = var.cloudflare_account_id
  name = "${local.name_prefix} Blocked URLs"
  type       = "URL"

  items_with_description {
    value       = "https://malicious.example.com/path"
    description = "Known phishing site"
  }

  items_with_description {
    value       = "https://spam.example.org/ads"
    description = "Spam website"
  }
}

# 24. SERIAL type list
resource "cloudflare_teams_list" "serial_list" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Certificate Serials"
  type        = "SERIAL"
  description = "Revoked certificate serial numbers"
  items       = ["1234567890ABCDEF", "FEDCBA0987654321"]

  items_with_description {
    value       = "DEADBEEF12345678"
    description = "Compromised certificate"
  }
}

# 25. List with special characters and various IP formats
resource "cloudflare_teams_list" "complex_ips" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Complex IP List"
  type        = "IP"
  description = "Various IP formats"
  items       = [
    "172.16.0.0/12",
    "192.168.0.0/16",
    "203.0.113.0/24"
  ]

  items_with_description {
    value       = "198.51.100.0/24"
    description = "Documentation range"
  }
}

# 26. EMAIL type with complex addresses
resource "cloudflare_teams_list" "complex_emails" {
  account_id  = var.cloudflare_account_id
  name = "${local.name_prefix} Complex Email List"
  type        = "EMAIL"
  description = "Emails with various formats"
  items       = [
    "user+tag@example.com",
    "user.name@subdomain.example.com",
    "admin@test.example.com"
  ]
}

# Total: 26 base resources + 3 from for_each map + 4 from for_each set + 3 from count = 36 instances
