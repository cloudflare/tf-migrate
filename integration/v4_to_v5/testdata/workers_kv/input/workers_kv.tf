# Test Case 1: Basic Workers KV resource
resource "cloudflare_workers_kv" "basic" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "config_key"
  value        = "config_value"
}

# Test Case 2: KV with special characters
resource "cloudflare_workers_kv" "special_chars" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "api%2Ftoken"
  value        = "{\"api_key\": \"test123\", \"endpoint\": \"https://api.example.com\"}"
}

# Test Case 3: KV with empty value
resource "cloudflare_workers_kv" "empty_value" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "placeholder"
  value        = ""
}
