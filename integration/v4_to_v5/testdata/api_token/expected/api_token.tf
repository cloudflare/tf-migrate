variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test Case 1: Basic API token with single policy
resource "cloudflare_api_token" "basic_token" {
  name = "cftftest Basic API Token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
      }, {
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }]
}

# Test Case 2: API token with multiple policies
resource "cloudflare_api_token" "multi_policy_token" {
  name = "cftftest Multi Policy Token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    }, {
    effect = "deny"
    resources = jsonencode({
      "com.cloudflare.api.account.zone.*" = "*"
    })
    permission_groups = [{
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }]
}

# Test Case 3: API token with condition block
resource "cloudflare_api_token" "conditional_token" {
  name = "cftftest Conditional Token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
  condition = {
    request_ip = {
      in = [
        "192.168.1.0/24",
        "10.0.0.0/8"
      ]
    }
  }
}

# Test Case 4: API token with condition including not_in
resource "cloudflare_api_token" "advanced_condition_token" {
  name = "cftftest Advanced Condition Token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*"      = "*"
      "com.cloudflare.api.account.zone.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
  condition = {
    request_ip = {
      in = [
        "192.168.0.0/16",
        "10.0.0.0/8",
        "172.16.0.0/12"
      ]
      not_in = [
        "192.168.1.100/32",
        "10.0.0.1/32"
      ]
    }
  }
}

# Test Case 5: API token with TTL fields
resource "cloudflare_api_token" "time_limited_token" {
  name = "cftftest Time Limited Token"
  expires_on = "2025-12-31T23:59:59Z"
  not_before = "2024-01-01T00:00:00Z"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}

# Test Case 6: API token with minimal permission groups
resource "cloudflare_api_token" "empty_perms_token" {
  name = "cftftest Minimal Permissions Token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}

# Test Case 7: Full example with all features
resource "cloudflare_api_token" "full_example" {
  name = "cftftest Full Example Token"
  expires_on = "2035-12-31T23:59:59Z"
  not_before = "2024-01-01T00:00:00Z"



  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*"      = "*"
      "com.cloudflare.api.account.zone.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    }, {
    effect = "deny"
    resources = jsonencode({
      "com.cloudflare.api.account.zone.*" = "*"
    })
    permission_groups = [{
      id = "e086da7e2179491d91ee5f35b3ca210a"
    }]
  }]
  condition = {
    request_ip = {
      in = [
        "192.168.0.0/16",
        "10.0.0.0/8",
        "172.16.0.0/12",
        "fd00::/8"
      ]
      not_in = [
        "192.168.1.1/32",
        "10.0.0.1/32"
      ]
    }
  }
}

# Test Case 8: Token with data reference and timestamps
resource "cloudflare_api_token" "api_token_create" {
  name = "cftftest api_token_create"


  not_before = "2024-01-01T00:00:00Z"
  expires_on = "2035-12-31T23:59:59Z"

  policies = [{
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    effect = "allow"
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
  condition = {
    request_ip = {
      in     = ["192.0.2.1/32"]
      not_in = ["198.51.100.1/32"]
    }
  }
}

# Test Case 9: Token with complex multi-key resources map
resource "cloudflare_api_token" "complex_resources_token" {
  name = "cftftest Complex Resources Token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*"      = "*"
      "com.cloudflare.api.account.zone.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}

# Test Case 10: Token with multiple permission groups (v4 string format)
resource "cloudflare_api_token" "multi_perms_token" {
  name = "cftftest Multiple Permissions Token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
      }, {
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }]
}

# Test Case 11: Token without effect (should default to allow)
resource "cloudflare_api_token" "no_effect_token" {
  name = "cftftest No Effect Token"

  policies = [{
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    effect = "allow"
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}

# Test Case 12: Token with variable references in resources
resource "cloudflare_api_token" "variable_resources_token" {
  name = "cftftest Variable Resources Token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.${var.cloudflare_account_id}"   = "*"
      "com.cloudflare.api.account.zone.${var.cloudflare_zone_id}" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}
