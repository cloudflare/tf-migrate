# Basic IP list with simple items array
resource "cloudflare_zero_trust_list" "ip_list" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "IP Allowlist"
  type       = "IP"
  items {
    value = "192.168.1.1"
  }
  items {
    value = "192.168.1.2"
  }
  items {
    value = "10.0.0.0/8"
  }
}

# Domain list with items_with_description blocks
resource "cloudflare_zero_trust_list" "domain_list" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Allowed Domains"
  type        = "DOMAIN"
  description = "Company approved domains"
  items {
    value       = "example.com"
    description = "Main company domain"
  }
  items {
    value       = "api.example.com"
    description = "API subdomain"
  }
  items {
    value       = "test.example.com"
    description = "Testing environment"
  }
}

# Mixed list with both items and items_with_description
resource "cloudflare_zero_trust_list" "email_list" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "VIP Emails"
  type       = "EMAIL"
  items {
    value = "admin@example.com"
  }
  items {
    value = "security@example.com"
  }
  items {
    value       = "ceo@example.com"
    description = "CEO email address"
  }
  items {
    value       = "cto@example.com"
    description = "CTO email address"
  }
}

# URL list with only items_with_description
resource "cloudflare_zero_trust_list" "url_list" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Blocked URLs"
  type       = "URL"
  items {
    value       = "https://malicious.example.com/path"
    description = "Known phishing site"
  }
  items {
    value       = "https://spam.example.org/ads"
    description = "Spam website"
  }
}

# Empty list - should be handled properly
resource "cloudflare_zero_trust_list" "empty_list" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Empty Serial List"
  type       = "SERIAL"
}

# List with special characters and various formats
resource "cloudflare_zero_trust_list" "complex_ips" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Complex IP List"
  type        = "IP"
  description = "Various IP formats"
  items {
    value = "172.16.0.0/12"
  }
  items {
    value = "192.168.0.0/16"
  }
  items {
    value = "203.0.113.0/24"
  }
  items {
    value       = "198.51.100.0/24"
    description = "Documentation range"
  }
}