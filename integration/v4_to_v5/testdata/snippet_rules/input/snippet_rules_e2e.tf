# Comprehensive Integration Tests for snippet_rules Migration
# This file tests all Terraform patterns and edge cases

# Test 1: Basic resource with single rule
resource "cloudflare_snippet_rules" "basic" {
  zone_id = "zone-basic-123"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/api\""
    snippet_name = "basic_snippet"
    description  = "Basic test rule"
  }
}

# Test 2: Multiple rules in one resource
resource "cloudflare_snippet_rules" "multiple_rules" {
  zone_id = "zone-multi-456"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/v1\""
    snippet_name = "v1_snippet"
    description  = "Version 1 API"
  }

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/v2\""
    snippet_name = "v2_snippet"
    description  = "Version 2 API"
  }

  rules {
    expression   = "http.request.uri.path eq \"/v3\""
    snippet_name = "v3_snippet"
  }
}

# Test 3: Using variables
variable "zone_id" {
  type    = string
  default = "zone-var-789"
}

variable "snippet_name" {
  type    = string
  default = "var_snippet"
}

resource "cloudflare_snippet_rules" "with_variables" {
  zone_id = var.zone_id

  rules {
    enabled      = var.enable_snippet
    expression   = "http.request.uri.path eq \"/var\""
    snippet_name = var.snippet_name
    description  = "Rule with variables"
  }
}

# Test 4: for_each with map
resource "cloudflare_snippet_rules" "for_each_map" {
  for_each = {
    prod    = "zone-prod-001"
    staging = "zone-staging-002"
    dev     = "zone-dev-003"
  }

  zone_id = each.value

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/${each.key}\""
    snippet_name = "${each.key}_snippet"
    description  = "Rule for ${each.key} environment"
  }
}

# Test 5: for_each with set
resource "cloudflare_snippet_rules" "for_each_set" {
  for_each = toset(["alpha", "beta", "gamma"])

  zone_id = "zone-set-${each.key}"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/${each.key}\""
    snippet_name = "${each.key}_snippet"
  }
}

# Test 6: count-based resources
resource "cloudflare_snippet_rules" "count_based" {
  count = 3

  zone_id = "zone-count-${count.index}"

  rules {
    enabled      = count.index == 0
    expression   = "http.request.uri.path eq \"/count/${count.index}\""
    snippet_name = "count_${count.index}_snippet"
    description  = "Count-based rule ${count.index}"
  }
}

# Test 7: Conditional creation
resource "cloudflare_snippet_rules" "conditional" {
  count = var.create_conditional ? 1 : 0

  zone_id = "zone-conditional-999"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/conditional\""
    snippet_name = "conditional_snippet"
  }
}

# Test 8: Complex expressions
resource "cloudflare_snippet_rules" "complex_expressions" {
  zone_id = "zone-complex-111"

  rules {
    enabled      = true
    expression   = "(http.request.uri.path eq \"/api\") and (http.request.method eq \"POST\")"
    snippet_name = "complex_snippet_1"
    description  = "Complex expression with AND"
  }

  rules {
    enabled      = true
    expression   = "(http.host eq \"example.com\") or (http.host eq \"www.example.com\")"
    snippet_name = "complex_snippet_2"
    description  = "Complex expression with OR"
  }
}

# Test 9: Cross-resource reference
resource "cloudflare_snippet_rules" "referenced" {
  zone_id = cloudflare_snippet_rules.basic.zone_id

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/referenced\""
    snippet_name = "ref_snippet"
    description  = "References another snippet_rules resource"
  }
}

# Test 10: Dynamic rules using locals
locals {
  environments = ["dev", "test", "prod"]
  base_zone_id = "zone-local-base"
}

resource "cloudflare_snippet_rules" "with_locals" {
  zone_id = local.base_zone_id

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/local\""
    snippet_name = "local_snippet"
    description  = "Using locals: ${join(", ", local.environments)}"
  }
}

# Test 11: Rules with special characters in expressions
resource "cloudflare_snippet_rules" "special_chars" {
  zone_id = "zone-special-222"

  rules {
    enabled      = true
    expression   = "http.request.uri.path matches \"^/api/v[0-9]+/.*$\""
    snippet_name = "regex_snippet"
    description  = "Rule with regex pattern"
  }

  rules {
    enabled      = true
    expression   = "http.request.uri.query contains \"token=\""
    snippet_name = "query_snippet"
    description  = "Rule checking query parameters"
  }
}

# Test 12: Minimal rules (only required fields)
resource "cloudflare_snippet_rules" "minimal" {
  zone_id = "zone-minimal-333"

  rules {
    expression   = "http.request.uri.path eq \"/minimal\""
    snippet_name = "minimal_snippet"
  }
}

# Test 13: All optional fields populated
resource "cloudflare_snippet_rules" "all_fields" {
  zone_id = "zone-all-444"

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/all\""
    snippet_name = "all_fields_snippet"
    description  = "Rule with all optional fields set"
  }
}

# Test 14: Mixed enabled states
resource "cloudflare_snippet_rules" "mixed_enabled" {
  zone_id = "zone-mixed-555"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/enabled\""
    snippet_name = "enabled_snippet"
  }

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/disabled\""
    snippet_name = "disabled_snippet"
  }

  rules {
    # Implicit enabled (uses default)
    expression   = "http.request.uri.path eq \"/default\""
    snippet_name = "default_snippet"
  }
}

# Test 15: Resource with depends_on
resource "cloudflare_snippet_rules" "depends" {
  depends_on = [cloudflare_snippet_rules.basic]

  zone_id = "zone-depends-666"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/depends\""
    snippet_name = "depends_snippet"
  }
}

# Test 16: Using terraform functions
resource "cloudflare_snippet_rules" "with_functions" {
  zone_id = "zone-func-777"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/${lower("UPPERCASE")}\""
    snippet_name = "${format("func_%s_snippet", "test")}"
    description  = "${title("function-based description")}"
  }
}

# Test 17: Multiple rules with various patterns
resource "cloudflare_snippet_rules" "comprehensive" {
  zone_id = "zone-comprehensive-888"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/rule1\""
    snippet_name = "rule1_snippet"
    description  = "First rule"
  }

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/rule2\""
    snippet_name = "rule2_snippet"
    description  = "Second rule"
  }

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/rule3\""
    snippet_name = "rule3_snippet"
    description  = "Third rule"
  }

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/rule4\""
    snippet_name = "rule4_snippet"
  }

  rules {
    expression   = "http.request.uri.path eq \"/rule5\""
    snippet_name = "rule5_snippet"
    description  = "Fifth rule without explicit enabled"
  }
}
