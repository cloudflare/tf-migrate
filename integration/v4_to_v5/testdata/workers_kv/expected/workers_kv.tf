# Test Case 1: Basic Workers KV resource
resource "cloudflare_workers_kv" "basic" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = "config_value"
  key_name     = "config_key"
}

# Test Case 2: KV with special characters
resource "cloudflare_workers_kv" "special_chars" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = "{\"api_key\": \"test123\", \"endpoint\": \"https://api.example.com\"}"
  key_name     = "api%2Ftoken"
}

# Test Case 3: KV with empty value
resource "cloudflare_workers_kv" "empty_value" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = ""
  key_name     = "placeholder"
}
