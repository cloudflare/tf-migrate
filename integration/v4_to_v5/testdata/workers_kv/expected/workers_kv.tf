variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Locals for reusable values
locals {
  common_account   = var.cloudflare_account_id
  namespace_prefix = "cftftest-kv"
  test_keys = [
    "config_key_1",
    "config_key_2",
    "config_key_3"
  ]
  enable_optional = true
  disable_test    = false
}

# Parent resource: KV Namespace for dependency testing
resource "cloudflare_workers_kv_namespace" "test_namespace" {
  account_id = local.common_account
  title      = "${local.namespace_prefix}-namespace"
}

# Pattern Group 1: Basic Resources
# Test Case 1: Minimal resource (only required fields)
resource "cloudflare_workers_kv" "minimal" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "minimal_value"
  key_name     = "minimal_key"
}

# Test Case 2: Basic resource with cross-resource reference
resource "cloudflare_workers_kv" "basic" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "config_value"
  key_name     = "config_key"
}

# Test Case 3: KV with special characters in key
resource "cloudflare_workers_kv" "special_chars" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "{\"api_key\": \"test123\", \"endpoint\": \"https://api.example.com\"}"
  key_name     = "api/token"
}

# Test Case 4: KV with empty value
resource "cloudflare_workers_kv" "empty_value" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = ""
  key_name     = "placeholder"
}

# Test Case 5: KV with special characters in value
resource "cloudflare_workers_kv" "special_value" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Line 1\nLine 2\tTabbed\r\nWindows line"
  key_name     = "special_value_key"
}

# Pattern Group 2: for_each with Maps (3-5 resources)
resource "cloudflare_workers_kv" "map_example" {
  for_each = {
    "config1" = {
      key   = "map_config_1"
      value = "Config value 1"
    }
    "config2" = {
      key   = "map_config_2"
      value = "Config value 2"
    }
    "config3" = {
      key   = "map_config_3"
      value = "Config value 3"
    }
    "config4" = {
      key   = "map_config_4"
      value = "Config value 4"
    }
  }

  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = each.value.value
  key_name     = each.value.key
}

# Pattern Group 3: for_each with Sets (3-5 items)
resource "cloudflare_workers_kv" "set_example" {
  for_each = toset([
    "alpha",
    "beta",
    "gamma",
    "delta"
  ])

  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Value for ${each.value}"
  key_name     = "set-${each.value}"
}

# Pattern Group 4: count-based Resources (at least 3)
resource "cloudflare_workers_kv" "counted" {
  count = 3

  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "This is count resource number ${count.index}"
  key_name     = "counted-${count.index}"
}

# Pattern Group 5: Conditional Creation
resource "cloudflare_workers_kv" "conditional_enabled" {
  count = local.enable_optional ? 1 : 0

  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "This resource is conditionally created"
  key_name     = "conditional_enabled_key"
}

resource "cloudflare_workers_kv" "conditional_disabled" {
  count = local.disable_test ? 1 : 0

  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "This resource should NOT be created"
  key_name     = "conditional_disabled_key"
}

# Pattern Group 6: Terraform Functions
resource "cloudflare_workers_kv" "with_functions" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id


  # String interpolation with account ID
  value    = "Resource for account ${var.cloudflare_account_id}"
  key_name = join("-", ["test", "function", "key"])
}

resource "cloudflare_workers_kv" "with_base64" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id

  # base64encode() function
  value    = base64encode("This is a base64 encoded value for testing")
  key_name = "encoded_key"
}

# Pattern Group 7: Lifecycle Meta-Arguments
resource "cloudflare_workers_kv" "with_lifecycle" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Testing lifecycle rules"

  lifecycle {
    create_before_destroy = true
  }
  key_name = "lifecycle_test_key"
}

resource "cloudflare_workers_kv" "with_ignore_changes" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Value that might change"

  lifecycle {
    ignore_changes = [value]
  }
  key_name = "ignore_changes_key"
}

# Pattern Group 8: Edge Cases
# Large value (approaching 25 MiB limit - using a smaller 1KB sample)
resource "cloudflare_workers_kv" "large_value" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt."
  key_name     = "large_value_key"
}

# Key with URL encoding special cases
#resource "cloudflare_workers_kv" "url_encoded_key" {
#  account_id   = var.cloudflare_account_id
#  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
#  key          = "path/to/resource?query=value&param=test"
#  value        = "Value for URL-like key"
#}

# Key with Unicode characters
resource "cloudflare_workers_kv" "unicode_key" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Unicode test value: 你好世界"
  key_name     = "unicode_测试_key"
}

# Value with JSON
resource "cloudflare_workers_kv" "json_value" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value = jsonencode({
    name    = "test"
    enabled = true
    count   = 42
    tags    = ["tag1", "tag2", "tag3"]
  })
  key_name = "json_data"
}

# Pattern Group 9: Multiple Resources with Dependencies
resource "cloudflare_workers_kv_namespace" "secondary_namespace" {
  account_id = var.cloudflare_account_id
  title      = "${local.namespace_prefix}-secondary"
}

resource "cloudflare_workers_kv" "secondary_ns_kv1" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.secondary_namespace.id
  value        = "Value in secondary namespace 1"
  key_name     = "secondary_key_1"
}

resource "cloudflare_workers_kv" "secondary_ns_kv2" {
  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.secondary_namespace.id
  value        = "Value in secondary namespace 2"
  key_name     = "secondary_key_2"
}

# Using local values in keys
resource "cloudflare_workers_kv" "with_locals" {
  count = length(local.test_keys)

  account_id   = var.cloudflare_account_id
  namespace_id = cloudflare_workers_kv_namespace.test_namespace.id
  value        = "Value for ${local.test_keys[count.index]}"
  key_name     = local.test_keys[count.index]
}
