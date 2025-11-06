# Test Case 1: Basic Workers KV Namespace
resource "cloudflare_workers_kv_namespace" "basic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "test-namespace"
}

# Test Case 2: Namespace with special characters
resource "cloudflare_workers_kv_namespace" "special_chars" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "test-namespace-2024"
}

# Test Case 3: Namespace with spaces
resource "cloudflare_workers_kv_namespace" "with_spaces" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  title      = "My Workers KV Namespace"
}
