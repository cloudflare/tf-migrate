variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test Case 1: Basic Workers KV Namespace
resource "cloudflare_workers_kv_namespace" "basic" {
  account_id = var.cloudflare_account_id
  title      = "test-namespace"
}

# Test Case 2: Namespace with special characters
resource "cloudflare_workers_kv_namespace" "special_chars" {
  account_id = var.cloudflare_account_id
  title      = "test-namespace-2024"
}

# Test Case 3: Namespace with spaces
resource "cloudflare_workers_kv_namespace" "with_spaces" {
  account_id = var.cloudflare_account_id
  title      = "My Workers KV Namespace"
}
