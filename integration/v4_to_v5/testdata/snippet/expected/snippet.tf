# Comprehensive snippet integration test data

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

locals {
  name_prefix = "cftftest"
  zone_id     = "abc123def456"
}

variable "snippet_name" {
  type    = string
  default = "test_snippet"
}

variable "main_module_file" {
  type    = string
  default = "index.js"
}

# Test 1: Minimal snippet with single file
resource "cloudflare_snippet" "minimal" {
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_minimal"
  files = [{
    name    = "main.js"
    content = <<-EOT
      export default {
        async fetch(request, env, ctx) {
          const response = await fetch(request);
          const newHeaders = new Headers(response.headers);
          
          // Your custom logic here
          newHeaders.set("x-snippet-type", "inline-terraform-string");

          return new Response(response.body, {
            status: response.status,
            statusText: response.statusText,
            headers: newHeaders,
          });
        }
      };
    EOT
  }]
  metadata = {
    main_module = "main.js"
  }
}

# Test 2: Snippet with multiple files
resource "cloudflare_snippet" "multi_file" {
  zone_id = var.cloudflare_zone_id



  snippet_name = "${local.name_prefix}_multi"
  files = [{
    name    = "index.js"
    content = "export default { fetch() { return new Response('Hello'); } }"
    }, {
    name    = "utils.js"
    content = "export function helper() { return 42; }"
    }, {
    name    = "config.js"
    content = "export default {\"key\": \"value\"};"
  }]
  metadata = {
    main_module = "index.js"
  }
}

# Test 3: Snippet with special characters in content
resource "cloudflare_snippet" "special_chars" {
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_special"
  files = [{
    name    = "special.js"
    content = "export default { fetch() { const msg = \"Hello, \\\"World\\\"!\"; return new Response(msg); } }"
  }]
  metadata = {
    main_module = "special.js"
  }
}

# Test 4: Snippet using count
resource "cloudflare_snippet" "with_count" {
  count   = 3
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_count_${count.index}"
  files = [{
    name    = "main.js"
    content = "export default { fetch(r) { console.log('Index: ${count.index}'); return fetch(r); } }"
  }]
  metadata = {
    main_module = "main.js"
  }
}

# Test 5: Snippet using for_each with set
resource "cloudflare_snippet" "with_for_each_set" {
  for_each = toset(["api", "web", "worker"])

  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_${each.value}"
  files = [{
    name    = "${each.value}.js"
    content = "export default { fetch(r) { return fetch(r); } }; // Snippet for ${each.value}"
  }]
  metadata = {
    main_module = "${each.value}.js"
  }
}

# Test 6: Snippet using for_each with map
resource "cloudflare_snippet" "with_for_each_map" {
  for_each = {
    prod    = "production.js"
    staging = "staging.js"
    dev     = "development.js"
  }

  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_${each.key}"
  files = [{
    name    = each.value
    content = "export default { fetch(r) { return fetch(r); } }; // Environment: ${each.key}"
  }]
  metadata = {
    main_module = each.value
  }
}

# Test 7: Snippet with conditional (ternary operator)
resource "cloudflare_snippet" "conditional" {
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_conditional"
  files = [{
    name    = var.snippet_name != "" ? "custom.js" : "default.js"
    content = "export default { fetch(r) { return fetch(r); } }; ${var.snippet_name != "" ? "// Custom" : "// Default"}"
  }]
  metadata = {
    main_module = var.snippet_name != "" ? "custom.js" : "default.js"
  }
}

# Test 8: Snippet with multiline content
resource "cloudflare_snippet" "multiline" {
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_multiline"
  files = [{
    name    = "app.js"
    content = <<-EOT
      export default {
        async fetch(request) {
          return new Response('Hello World!');
        }
      }
    EOT
  }]
  metadata = {
    main_module = "app.js"
  }
}

# Test 9: Snippet with jsonencode function
resource "cloudflare_snippet" "with_jsonencode" {
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_json"
  files = [{
    name    = "index.js"
    content = "const config = ${jsonencode({ key = "value", debug = true })}; export default { fetch(r) { return new Response(JSON.stringify(config)); } }"
  }]
  metadata = {
    main_module = "index.js"
  }
}

# Test 10: Snippet with interpolation in all fields
resource "cloudflare_snippet" "interpolated" {
  zone_id = "${var.cloudflare_zone_id}"

  snippet_name = "${local.name_prefix}_interpolated"
  files = [{
    name    = "${var.main_module_file}"
    content = "export default { fetch(r) { return fetch(r); } }; // ${var.snippet_name}"
  }]
  metadata = {
    main_module = "${var.main_module_file}"
  }
}

# Test 11: Snippet with lifecycle meta-argument
resource "cloudflare_snippet" "with_lifecycle" {
  zone_id = var.cloudflare_zone_id


  lifecycle {
    create_before_destroy = true
    ignore_changes        = [files]
  }
  snippet_name = "${local.name_prefix}_lifecycle"
  files = [{
    name    = "main.js"
    content = "export default { fetch(r) { return fetch(r); } }; // Lifecycle test"
  }]
  metadata = {
    main_module = "main.js"
  }
}

# Test 12: Snippet with depends_on
resource "cloudflare_snippet" "with_depends_on" {
  depends_on = [cloudflare_snippet.minimal]

  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_depends"
  files = [{
    name    = "dependent.js"
    content = "export default { fetch(r) { return fetch(r); } }; // Depends on minimal snippet"
  }]
  metadata = {
    main_module = "dependent.js"
  }
}

# Test 13: Snippet with long content
resource "cloudflare_snippet" "long_content" {
  zone_id = var.cloudflare_zone_id

  snippet_name = "${local.name_prefix}_long"
  files = [{
    name    = "worker.js"
    content = <<-EOT
      // This is a comprehensive worker script
      export default {
        async fetch(request, env, ctx) {
          const url = new URL(request.url);

          // Handle different routes
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
  }]
  metadata = {
    main_module = "worker.js"
  }
}
