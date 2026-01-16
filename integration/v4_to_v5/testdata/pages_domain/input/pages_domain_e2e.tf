# ========================================
# Variables
# ========================================
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

# Locals for common values
locals {
  name_prefix = "tf-migrate-e2e-test"
}

# ============================================================================
# Scenario 1: Basic pages_domain with project reference
# ============================================================================

resource "cloudflare_pages_project" "basic" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-basic"
  production_branch = "main"
}

resource "cloudflare_pages_domain" "basic" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.basic.name
  domain       = "${local.name_prefix}-basic.${var.cloudflare_domain}"
}

# ============================================================================
# Scenario 2: Multiple domains for single project
# ============================================================================

resource "cloudflare_pages_project" "multi_domain" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-multi"
  production_branch = "main"
}

resource "cloudflare_pages_domain" "primary" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.multi_domain.name
  domain       = "${local.name_prefix}-primary.${var.cloudflare_domain}"
}

resource "cloudflare_pages_domain" "secondary" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.multi_domain.name
  domain       = "${local.name_prefix}-secondary.${var.cloudflare_domain}"
}

# ============================================================================
# Scenario 3: Using for_each to create multiple domains
# ============================================================================

resource "cloudflare_pages_project" "foreach_test" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-foreach"
  production_branch = "main"
}

variable "environment_domains" {
  type = map(string)
  default = {
    staging = "staging"
    preview = "preview"
  }
}

resource "cloudflare_pages_domain" "environments" {
  for_each = var.environment_domains

  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.foreach_test.name
  domain       = "${local.name_prefix}-${each.value}.${var.cloudflare_domain}"
}

# ============================================================================
# Scenario 4: Conditional creation with count
# ============================================================================

resource "cloudflare_pages_project" "conditional" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-conditional"
  production_branch = "main"
}

variable "enable_custom_domain" {
  type    = bool
  default = true
}

resource "cloudflare_pages_domain" "conditional" {
  count = var.enable_custom_domain ? 1 : 0

  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.conditional.name
  domain       = "${local.name_prefix}-conditional.${var.cloudflare_domain}"
}

# ============================================================================
# Scenario 5: Complex references with locals and variables
# ============================================================================

locals {
  projects = {
    app = "${local.name_prefix}-app"
    api = "${local.name_prefix}-api"
  }
}

resource "cloudflare_pages_project" "app" {
  account_id        = var.cloudflare_account_id
  name              = local.projects.app
  production_branch = "main"
}

resource "cloudflare_pages_project" "api" {
  account_id        = var.cloudflare_account_id
  name              = local.projects.api
  production_branch = "main"
}

resource "cloudflare_pages_domain" "app_domain" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.app.name
  domain       = "app.${var.cloudflare_domain}"
}

resource "cloudflare_pages_domain" "api_domain" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.api.name
  domain       = "api.${var.cloudflare_domain}"
}
