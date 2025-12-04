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
resource "cloudflare_zero_trust_list" "minimal" {
  account_id = local.common_account_id
  name       = "cftftest Minimal List"
  type       = "IP"
  items = [{
    description = null
    value       = "192.168.1.1"
  }]
}

# 2. Maximal resource - all fields populated
resource "cloudflare_zero_trust_list" "maximal" {
  account_id  = local.common_account_id
  name        = "${local.list_name_prefix}-maximal"
  type        = "DOMAIN"
  description = local.common_description


  items = [{
    description = "Secure subdomain"
    value       = "secure.example.com"
    }, {
    description = "Admin portal"
    value       = "admin.example.com"
    }, {
    description = null
    value       = "example.com"
    }, {
    description = null
    value       = "test.example.com"
    }, {
    description = null
    value       = "api.example.com"
  }]
}

# 3. Empty list - edge case
resource "cloudflare_zero_trust_list" "empty" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Empty List"
  type        = "URL"
  description = "Empty list for testing"
}

# 4. Basic IP list with simple items array
resource "cloudflare_zero_trust_list" "ip_list" {
  account_id = var.cloudflare_account_id
  name       = "cftftest IP Allowlist"
  type       = "IP"
  items = [{
    description = null
    value       = "192.168.1.1"
    }, {
    description = null
    value       = "192.168.1.2"
    }, {
    description = null
    value       = "10.0.0.0/8"
  }]
}

# 5. Domain list with items_with_description blocks
resource "cloudflare_zero_trust_list" "domain_list" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Allowed Domains"
  type        = "DOMAIN"
  description = "Company approved domains"



  items = [{
    description = "Main company domain"
    value       = "example.com"
    }, {
    description = "API subdomain"
    value       = "api.example.com"
    }, {
    description = "Testing environment"
    value       = "test.example.com"
  }]
}

# 6. Mixed list with both items and items_with_description
resource "cloudflare_zero_trust_list" "email_list" {
  account_id = var.cloudflare_account_id
  name       = "cftftest VIP Emails"
  type       = "EMAIL"


  items = [{
    description = "CEO email address"
    value       = "ceo@example.com"
    }, {
    description = "CTO email address"
    value       = "cto@example.com"
    }, {
    description = null
    value       = "admin@example.com"
    }, {
    description = null
    value       = "security@example.com"
  }]
}

# ============================================================================
# Pattern Group 2: for_each with Maps
# ============================================================================

# 7-9. Resources created with for_each over map
resource "cloudflare_zero_trust_list" "security_domains" {
  for_each = var.security_domains

  account_id  = var.cloudflare_account_id
  name        = "${local.list_name_prefix}-${each.key}"
  type        = each.value.type
  description = each.value.description
  items = [
    {
      value       = "${each.key}.example.com"
      description = null
    },
    {
      value       = "www.${each.key}.example.com"
      description = null
    }
  ]
}

# ============================================================================
# Pattern Group 3: for_each with Sets
# ============================================================================

# 10-13. Resources created with for_each over set
resource "cloudflare_zero_trust_list" "list_types" {
  for_each = toset(["staging", "production", "development", "testing"])

  account_id  = var.cloudflare_account_id
  name        = "${local.list_name_prefix}-${each.value}"
  type        = "DOMAIN"
  description = "List for ${each.value} environment"
  items = [
    {
      value       = "${each.value}.example.com"
      description = null
    }
  ]
}

# ============================================================================
# Pattern Group 4: count-based Resources
# ============================================================================

# 14-16. Resources created with count
resource "cloudflare_zero_trust_list" "ip_ranges" {
  count = 3

  account_id  = var.cloudflare_account_id
  name        = "${local.list_name_prefix}-range-${count.index}"
  type        = "IP"
  description = "IP range ${count.index}"
  items = [
    {
      value       = local.ip_ranges[count.index]
      description = null
    }
  ]
}

# ============================================================================
# Pattern Group 5: Conditional Creation
# ============================================================================

# 17. Conditionally created resource
resource "cloudflare_zero_trust_list" "conditional_enabled" {
  count = var.enable_security_lists ? 1 : 0

  account_id  = var.cloudflare_account_id
  name        = "cftftest Security List - Enabled"
  type        = "DOMAIN"
  description = "This list is conditionally created"
  items = [{
    description = null
    value       = "secure.example.com"
    }, {
    description = null
    value       = "protected.example.com"
  }]
}

# 18. Conditionally not created resource
resource "cloudflare_zero_trust_list" "conditional_disabled" {
  count = var.enable_security_lists ? 0 : 1

  account_id  = var.cloudflare_account_id
  name        = "cftftest Security List - Disabled"
  type        = "DOMAIN"
  description = "This list is conditionally not created"
  items = [{
    description = null
    value       = "insecure.example.com"
  }]
}

# ============================================================================
# Pattern Group 6: Terraform Functions
# ============================================================================

# 19. Resource using join() function
resource "cloudflare_zero_trust_list" "with_join" {
  account_id  = var.cloudflare_account_id
  name        = join("-", [local.list_name_prefix, "joined", "name"])
  type        = "DOMAIN"
  description = "List created with join function"
  items = [
    {
      value       = "${join(".", ["subdomain", "example", "com"])}"
      description = null
    }
  ]
}

# 20. Resource using string interpolation
resource "cloudflare_zero_trust_list" "with_interpolation" {
  account_id  = var.cloudflare_account_id
  name        = "List for account ${var.cloudflare_account_id}"
  type        = "URL"
  description = "Description with ${local.list_name_prefix} interpolation"
  items = [
    {
      value       = "https://${var.list_prefix}.example.com/path"
      description = null
    }
  ]
}

# ============================================================================
# Pattern Group 7: Lifecycle Meta-Arguments
# ============================================================================

# 21. Resource with lifecycle block
resource "cloudflare_zero_trust_list" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Protected List"
  type        = "IP"
  description = "List with lifecycle settings"

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }
  items = [{
    description = null
    value       = "203.0.113.0/24"
  }]
}

# 22. Resource with prevent_destroy (set to false for testing)
resource "cloudflare_zero_trust_list" "with_prevent_destroy" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Important List"
  type        = "DOMAIN"
  description = "Critical list"

  lifecycle {
    prevent_destroy = false # Set to false for testing
  }
  items = [{
    description = null
    value       = "critical.example.com"
  }]
}

# ============================================================================
# Pattern Group 8: Additional Edge Cases & Type Coverage
# ============================================================================

# 23. URL list with only items_with_description
resource "cloudflare_zero_trust_list" "url_list" {
  account_id = var.cloudflare_account_id
  name       = "cftftest Blocked URLs"
  type       = "URL"


  items = [{
    description = "Known phishing site"
    value       = "https://malicious.example.com/path"
    }, {
    description = "Spam website"
    value       = "https://spam.example.org/ads"
  }]
}

# 24. SERIAL type list
resource "cloudflare_zero_trust_list" "serial_list" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Certificate Serials"
  type        = "SERIAL"
  description = "Revoked certificate serial numbers"

  items = [{
    description = "Compromised certificate"
    value       = "DEADBEEF12345678"
    }, {
    description = null
    value       = "1234567890ABCDEF"
    }, {
    description = null
    value       = "FEDCBA0987654321"
  }]
}

# 25. List with special characters and various IP formats
resource "cloudflare_zero_trust_list" "complex_ips" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Complex IP List"
  type        = "IP"
  description = "Various IP formats"

  items = [{
    description = "Documentation range"
    value       = "198.51.100.0/24"
    }, {
    description = null
    value       = "172.16.0.0/12"
    }, {
    description = null
    value       = "192.168.0.0/16"
    }, {
    description = null
    value       = "203.0.113.0/24"
  }]
}

# 26. EMAIL type with complex addresses
resource "cloudflare_zero_trust_list" "complex_emails" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Complex Email List"
  type        = "EMAIL"
  description = "Emails with various formats"
  items = [{
    description = null
    value       = "user+tag@example.com"
    }, {
    description = null
    value       = "user.name@subdomain.example.com"
    }, {
    description = null
    value       = "admin@test.example.com"
  }]
}

# Total: 26 base resources + 3 from for_each map + 4 from for_each set + 3 from count = 36 instances
