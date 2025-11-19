variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Basic IP list with simple items array
resource "cloudflare_zero_trust_list" "ip_list" {
  account_id = var.cloudflare_account_id
  name       = "IP Allowlist"
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

# Domain list with items_with_description blocks
resource "cloudflare_zero_trust_list" "domain_list" {
  account_id  = var.cloudflare_account_id
  name        = "Allowed Domains"
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

# Mixed list with both items and items_with_description
resource "cloudflare_zero_trust_list" "email_list" {
  account_id = var.cloudflare_account_id
  name       = "VIP Emails"
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

# URL list with only items_with_description
resource "cloudflare_zero_trust_list" "url_list" {
  account_id = var.cloudflare_account_id
  name       = "Blocked URLs"
  type       = "URL"


  items = [{
    description = "Known phishing site"
    value       = "https://malicious.example.com/path"
    }, {
    description = "Spam website"
    value       = "https://spam.example.org/ads"
  }]
}

# Empty list - should be handled properly
resource "cloudflare_zero_trust_list" "empty_list" {
  account_id = var.cloudflare_account_id
  name       = "Empty Serial List"
  type       = "SERIAL"
}

# List with special characters and various formats
resource "cloudflare_zero_trust_list" "complex_ips" {
  account_id  = var.cloudflare_account_id
  name        = "Complex IP List"
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