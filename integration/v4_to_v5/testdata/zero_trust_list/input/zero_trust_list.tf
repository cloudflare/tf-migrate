variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Basic IP list with simple items array
resource "cloudflare_teams_list" "ip_list" {
  account_id = var.cloudflare_account_id
  name       = "IP Allowlist"
  type       = "IP"
  items      = ["192.168.1.1", "192.168.1.2", "10.0.0.0/8"]
}

# Domain list with items_with_description blocks
resource "cloudflare_teams_list" "domain_list" {
  account_id  = var.cloudflare_account_id
  name        = "Allowed Domains"
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

# Mixed list with both items and items_with_description
resource "cloudflare_teams_list" "email_list" {
  account_id = var.cloudflare_account_id
  name       = "VIP Emails"
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

# URL list with only items_with_description
resource "cloudflare_teams_list" "url_list" {
  account_id = var.cloudflare_account_id
  name       = "Blocked URLs"
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

# Empty list - should be handled properly
resource "cloudflare_teams_list" "empty_list" {
  account_id = var.cloudflare_account_id
  name       = "Empty Serial List"
  type       = "SERIAL"
  items      = []
}

# List with special characters and various formats
resource "cloudflare_teams_list" "complex_ips" {
  account_id  = var.cloudflare_account_id
  name        = "Complex IP List"
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