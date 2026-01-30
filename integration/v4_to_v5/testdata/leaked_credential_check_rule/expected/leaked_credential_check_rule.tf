# This file tests comprehensive migration scenarios with 20+ resource instances
# Variables for dynamic configuration
variable "zone_id" {
  type    = string
  default = "0da42c8d2132a9ddaf714f9e7c920711"
}

variable "enable_optional_rules" {
  type    = bool
  default = true
}

# Locals for reusable values
locals {
  name_prefix = "cftftest"
  zone_id     = var.zone_id

  # Common ruleset expressions
  username_expressions = {
    form_username = "http.request.body.form.username"
    json_username = "lookup_json_string(http.request.body.raw, \"username\")"
    header_user   = "http.request.headers[\"x-user\"][0]"
  }

  password_expressions = {
    form_password = "http.request.body.form.password"
    json_password = "lookup_json_string(http.request.body.raw, \"password\")"
    header_pass   = "http.request.headers[\"x-pass\"][0]"
  }

  detection_patterns = toset([
    "form",
    "json",
    "header",
  ])
}

# =============================================================================
# TEST CASE 1: Basic resource with all fields
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "basic" {
  zone_id  = local.zone_id
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}

# =============================================================================
# TEST CASE 2: Minimal resource (zone_id only - both optional fields omitted)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "minimal" {
  zone_id = local.zone_id
}

# =============================================================================
# TEST CASE 3: Username only (password optional)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "username_only" {
  zone_id  = local.zone_id
  username = "http.request.body.form.username"
}

# =============================================================================
# TEST CASE 4: Password only (username optional)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "password_only" {
  zone_id  = local.zone_id
  password = "http.request.body.form.password"
}

# =============================================================================
# TEST CASE 5-7: Multiple simple resources
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "form_detection" {
  zone_id  = local.zone_id
  username = "http.request.body.form.user"
  password = "http.request.body.form.pass"
}

resource "cloudflare_leaked_credential_check_rule" "json_detection" {
  zone_id  = local.zone_id
  username = "lookup_json_string(http.request.body.raw, \"credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"credentials.password\")"
}

resource "cloudflare_leaked_credential_check_rule" "header_detection" {
  zone_id  = local.zone_id
  username = "http.request.headers[\"x-username\"][0]"
  password = "http.request.headers[\"x-password\"][0]"
}

# =============================================================================
# TEST CASE 8-10: for_each with map - username expressions
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "foreach_usernames" {
  for_each = local.username_expressions

  zone_id  = local.zone_id
  username = each.value
  password = local.password_expressions.form_password
}

# =============================================================================
# TEST CASE 11-13: for_each with set - detection patterns
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "foreach_patterns" {
  for_each = local.detection_patterns

  zone_id  = local.zone_id
  username = "http.request.body.${each.key}.username"
  password = "http.request.body.${each.key}.password"
}

# =============================================================================
# TEST CASE 14-16: count-based resources
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "counted" {
  count = 3

  zone_id  = local.zone_id
  username = "http.request.body.form.username${count.index}"
  password = "http.request.body.form.password${count.index}"
}

# =============================================================================
# TEST CASE 17: Conditional creation (enabled)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "conditional_enabled" {
  count = var.enable_optional_rules ? 1 : 0

  zone_id  = local.zone_id
  username = "http.request.body.form.conditional_user"
  password = "http.request.body.form.conditional_pass"
}

# =============================================================================
# TEST CASE 18: Conditional creation (disabled via count = 0)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "conditional_disabled" {
  count = var.enable_optional_rules ? 0 : 1

  zone_id  = local.zone_id
  username = "http.request.body.form.disabled_user"
  password = "http.request.body.form.disabled_pass"
}

# =============================================================================
# TEST CASE 19: Variable references (direct variable usage)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "with_variables" {
  zone_id  = var.zone_id
  username = "http.request.body.form.${local.name_prefix}_user"
  password = "http.request.body.form.${local.name_prefix}_pass"
}

# =============================================================================
# TEST CASE 20: Complex string interpolation
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "interpolated" {
  zone_id  = local.zone_id
  username = "${local.username_expressions.form_username}"
  password = "${local.password_expressions.form_password}"
}

# =============================================================================
# TEST CASE 21: Lifecycle meta-arguments
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "with_lifecycle" {
  zone_id  = local.zone_id
  username = "http.request.body.form.lifecycle_user"
  password = "http.request.body.form.lifecycle_pass"

  lifecycle {
    ignore_changes = [username]
  }
}

# =============================================================================
# TEST CASE 22: Complex nested JSON lookups
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "complex_json" {
  zone_id  = local.zone_id
  username = "lookup_json_string(http.request.body.raw, \"data.authentication.credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"data.authentication.credentials.password\")"
}

# =============================================================================
# TEST CASE 23: URL encoded form fields
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "urlencoded" {
  zone_id  = local.zone_id
  username = "http.request.body.form.user%5Fname"
  password = "http.request.body.form.pass%5Fword"
}

# =============================================================================
# TEST CASE 24: Null optional fields (explicitly set to null)
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "explicit_nulls" {
  zone_id  = local.zone_id
  username = null
  password = null
}

# =============================================================================
# TEST CASE 25: Query string parameters
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "query_params" {
  zone_id  = local.zone_id
  username = "http.request.uri.args[\"username\"][0]"
  password = "http.request.uri.args[\"password\"][0]"
}

# =============================================================================
# TEST CASE 26: Cookie-based detection
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "cookie_detection" {
  zone_id  = local.zone_id
  username = "http.request.cookies[\"session_user\"][0]"
  password = "http.request.cookies[\"session_token\"][0]"
}

# =============================================================================
# TEST CASE 27: Multi-step JSON path
# =============================================================================
resource "cloudflare_leaked_credential_check_rule" "deep_json" {
  zone_id  = local.zone_id
  username = "lookup_json_string(lookup_json_string(http.request.body.raw, \"payload\"), \"username\")"
  password = "lookup_json_string(lookup_json_string(http.request.body.raw, \"payload\"), \"password\")"
}

# =============================================================================
# Output for verification
# =============================================================================
output "total_rules_created" {
  value       = 27
  description = "Total number of leaked credential check rules created for testing"
}

output "conditional_rules_enabled" {
  value       = var.enable_optional_rules
  description = "Whether conditional rules are enabled"
}
