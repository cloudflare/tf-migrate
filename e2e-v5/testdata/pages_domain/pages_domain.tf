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

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

resource "cloudflare_pages_project" "domain_project" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-pages-domain-project"
  production_branch = "main"
  deployment_configs = {
    preview = {
      fail_open = false
    }
    production = {
      fail_open = false
    }
  }
}

# Primary domain mapping
resource "cloudflare_pages_domain" "primary" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.domain_project.name
  name         = "${local.name_prefix}-primary.${var.cloudflare_domain}"
}

# Additional environment domains using for_each to retain reference-pattern coverage
locals {
  environment_domains = {
    staging = "${local.name_prefix}-staging.${var.cloudflare_domain}"
    preview = "${local.name_prefix}-preview.${var.cloudflare_domain}"
  }
}

resource "cloudflare_pages_domain" "environments" {
  for_each = local.environment_domains

  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.domain_project.name
  name         = each.value
}
