variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Reference the namespace created in workers_kv_namespace module
# This creates a dependency and tests cross-resource references

# First create the namespace (from workers_kv_namespace testdata)
resource "cloudflare_workers_kv_namespace" "test_namespace" {
  account_id = var.cloudflare_account_id
  title      = "test-kv-namespace"
}

# Test Case 1: Basic Workers KV resource
resource "cloudflare_workers_kv" "basic" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  key          = "config_key"
  value        = "config_value"
}

# Test Case 2: KV with special characters
resource "cloudflare_workers_kv" "special_chars" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  key          = "api/token"
  value        = "{\"api_key\": \"test123\", \"endpoint\": \"https://api.example.com\"}"
}

# Test Case 3: KV with empty value
resource "cloudflare_workers_kv" "empty_value" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  key          = "placeholder"
  value        = ""
}
