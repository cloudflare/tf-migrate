# E2E test for snippet_rules migration
#
# cloudflare_snippet_rules is a zone-level singleton (one resource per zone),
# so we use a single resource with multiple rules for e2e testing.
# for_each, count, and multi-resource patterns are tested in the integration
# tests (snippet_rules.tf) which only validates config transformation.
#
# This file includes cloudflare_snippet resources inline so the module is
# self-contained. Run with: --resources snippet_rules

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "enable_snippet" {
  type    = bool
  default = true
}

variable "snippet_name" {
  type    = string
  default = "cftftest_sr_special"
}

variable "create_conditional" {
  type    = bool
  default = true
}

variable "main_module_file" {
  type    = string
  default = "index.js"
}

locals {
  name_prefix  = "cftftest"
  environments = ["dev", "test", "prod"]
  base_zone_id = var.cloudflare_zone_id
}

# ========================================
# Snippet resources (dependencies for rules)
# ========================================

resource "cloudflare_snippet" "minimal" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_minimal"
  main_module = "main.js"

  files {
    name    = "main.js"
    content = <<-EOT
      export default {
        async fetch(request, env, ctx) {
          const response = await fetch(request);
          const newHeaders = new Headers(response.headers);
          newHeaders.set("x-snippet-type", "inline-terraform-string");
          return new Response(response.body, {
            status: response.status,
            statusText: response.statusText,
            headers: newHeaders,
          });
        }
      };
    EOT
  }
}

resource "cloudflare_snippet" "multi_file" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_multi"
  main_module = "index.js"

  files {
    name    = "index.js"
    content = "export default { fetch() { return new Response('Hello'); } }"
  }

  files {
    name    = "utils.js"
    content = "export function helper() { return 42; }"
  }

  files {
    name    = "config.js"
    content = "export default {\"key\": \"value\"};"
  }
}

resource "cloudflare_snippet" "special_chars" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_special"
  main_module = "special.js"

  files {
    name    = "special.js"
  content = "export default { fetch() { const msg = \"Hello, \\\"World\\\"!\"; return new Response(msg); } }"
  }
}

resource "cloudflare_snippet" "with_count" {
  count       = 3
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_count_${count.index}"
  main_module = "main.js"

  files {
    name    = "main.js"
    content = "export default { fetch(r) { console.log('Index: ${count.index}'); return fetch(r); } }"
  }
}

resource "cloudflare_snippet" "with_for_each_set" {
  for_each = toset(["api", "web", "worker"])

  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_${each.value}"
  main_module = "${each.value}.js"

  files {
    name    = "${each.value}.js"
    content = "export default { fetch(r) { return fetch(r); } }; // Snippet for ${each.value}"
  }
}

resource "cloudflare_snippet" "with_for_each_map" {
  for_each = {
    prod    = "production.js"
    staging = "staging.js"
    dev     = "development.js"
  }

  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_${each.key}"
  main_module = each.value

  files {
    name    = each.value
    content = "export default { fetch(r) { return fetch(r); } }; // Environment: ${each.key}"
  }
}

resource "cloudflare_snippet" "conditional" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_conditional"
  main_module = var.snippet_name != "" ? "custom.js" : "default.js"

  files {
    name    = var.snippet_name != "" ? "custom.js" : "default.js"
    content = "export default { fetch(r) { return fetch(r); } }; ${var.snippet_name != "" ? "// Custom" : "// Default"}"
  }
}

resource "cloudflare_snippet" "multiline" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_multiline"
  main_module = "app.js"

  files {
    name = "app.js"
    content = <<-EOT
      export default {
        async fetch(request) {
          return new Response('Hello World!');
        }
      }
    EOT
  }
}

resource "cloudflare_snippet" "with_jsonencode" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_json"
  main_module = "index.js"

  files {
    name    = "index.js"
    content = "const config = ${jsonencode({ key = "value", debug = true })}; export default { fetch(r) { return new Response(JSON.stringify(config)); } }"
  }
}

resource "cloudflare_snippet" "interpolated" {
  zone_id     = "${var.cloudflare_zone_id}"
  name        = "${local.name_prefix}_sr_interpolated"
  main_module = "${var.main_module_file}"

  files {
    name    = "${var.main_module_file}"
    content = "export default { fetch(r) { return fetch(r); } }; // ${var.snippet_name}"
  }
}

resource "cloudflare_snippet" "with_lifecycle" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_lifecycle"
  main_module = "main.js"

  files {
    name    = "main.js"
    content = "export default { fetch(r) { return fetch(r); } }; // Lifecycle test"
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [files]
  }
}

resource "cloudflare_snippet" "with_depends_on" {
  depends_on = [cloudflare_snippet.minimal]

  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_depends"
  main_module = "dependent.js"

  files {
    name    = "dependent.js"
    content = "export default { fetch(r) { return fetch(r); } }; // Depends on minimal snippet"
  }
}

resource "cloudflare_snippet" "long_content" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}_sr_long"
  main_module = "worker.js"

  files {
    name = "worker.js"
    content = <<-EOT
      // This is a comprehensive worker script
      export default {
        async fetch(request, env, ctx) {
          const url = new URL(request.url);
          switch (url.pathname) {
            case '/api':
              return handleAPI(request);
            case '/health':
              return new Response('OK', { status: 200 });
            default:
              return new Response('Not Found', { status: 404 });
          }
        }
      }
      async function handleAPI(request) {
        const data = await request.json();
        return new Response(JSON.stringify(data), {
          headers: { 'Content-Type': 'application/json' }
        });
      }
    EOT
  }
}

# ========================================
# Snippet rules (zone-level singleton)
# ========================================

resource "cloudflare_snippet_rules" "basic" {
  depends_on = [
    cloudflare_snippet.minimal,
    cloudflare_snippet.multi_file,
    cloudflare_snippet.special_chars,
    cloudflare_snippet.with_count,
    cloudflare_snippet.with_for_each_set,
    cloudflare_snippet.with_for_each_map,
    cloudflare_snippet.conditional,
    cloudflare_snippet.multiline,
    cloudflare_snippet.with_jsonencode,
    cloudflare_snippet.interpolated,
    cloudflare_snippet.with_lifecycle,
    cloudflare_snippet.with_depends_on,
    cloudflare_snippet.long_content,
  ]

  zone_id = local.base_zone_id

  # Rule 1: Basic rule with literal snippet name
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/api\""
    snippet_name = "${local.name_prefix}_sr_minimal"
    description  = "Basic test rule"
  }

  # Rule 2: Using variable for enabled
  rules {
    enabled      = var.enable_snippet
    expression   = "http.request.uri.path eq \"/v1\""
    snippet_name = "${local.name_prefix}_sr_multi"
    description  = "Rule with variable-controlled enabled"
  }

  # Rule 3: Disabled rule with var.snippet_name
  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/v2\""
    snippet_name = var.snippet_name
    description  = "Rule using var.snippet_name"
  }

  # Rule 4: Rule without optional enabled field
  rules {
    expression   = "http.request.uri.path eq \"/v3\""
    snippet_name = "${local.name_prefix}_sr_count_0"
  }

  # Rule 5: Complex expression with AND
  rules {
    enabled      = true
    expression   = "(http.request.uri.path eq \"/complex\") and (http.request.method eq \"POST\")"
    snippet_name = "${local.name_prefix}_sr_count_1"
    description  = "Complex expression with AND"
  }

  # Rule 6: Complex expression with OR
  rules {
    enabled      = true
    expression   = "(http.host eq \"example.com\") or (http.host eq \"www.example.com\")"
    snippet_name = "${local.name_prefix}_sr_count_2"
    description  = "Complex expression with OR"
  }

  # Rule 7: Regex pattern in expression
  rules {
    enabled      = true
    expression   = "http.request.uri.path matches \"^/api/v[0-9]+/.*$\""
    snippet_name = "${local.name_prefix}_sr_api"
    description  = "Rule with regex pattern"
  }

  # Rule 8: Query string matching
  rules {
    enabled      = true
    expression   = "http.request.uri.query contains \"token=\""
    snippet_name = "${local.name_prefix}_sr_web"
    description  = "Rule checking query parameters"
  }

  # Rule 9: Using terraform functions in snippet_name and description
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/${lower("UPPERCASE")}\""
    snippet_name = "${local.name_prefix}_${format("%s", "sr_interpolated")}"
    description  = "${title("function-based description")}"
  }

  # Rule 10: Using locals in description
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/local\""
    snippet_name = "${local.name_prefix}_sr_lifecycle"
    description  = "Using locals: ${join(", ", local.environments)}"
  }

  # Rule 11: Minimal rule (only required fields)
  rules {
    expression   = "http.request.uri.path eq \"/minimal\""
    snippet_name = "${local.name_prefix}_sr_long"
  }

  # Rule 12: All optional fields populated
  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/all\""
    snippet_name = "${local.name_prefix}_sr_conditional"
    description  = "Rule with all optional fields set"
  }

  # Rule 13: Prod environment
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/prod\""
    snippet_name = "${local.name_prefix}_sr_prod"
    description  = "Rule for prod environment"
  }

  # Rule 14: Staging environment
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/staging\""
    snippet_name = "${local.name_prefix}_sr_staging"
    description  = "Rule for staging environment"
  }

  # Rule 15: Dev environment
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/dev\""
    snippet_name = "${local.name_prefix}_sr_dev"
    description  = "Rule for dev environment"
  }

  # Rule 16: Worker snippet
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/worker\""
    snippet_name = "${local.name_prefix}_sr_worker"
  }

  # Rule 17: Multiline snippet
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/multiline\""
    snippet_name = "${local.name_prefix}_sr_multiline"
    description  = "Rule referencing multiline snippet"
  }

  # Rule 18: Conditional enabled via ternary
  rules {
    enabled      = var.create_conditional ? true : false
    expression   = "http.request.uri.path eq \"/json\""
    snippet_name = "${local.name_prefix}_sr_json"
    description  = "Conditional enabled via ternary"
  }

  # Rule 19: Depends snippet
  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/depends\""
    snippet_name = "${local.name_prefix}_sr_depends"
  }
}
