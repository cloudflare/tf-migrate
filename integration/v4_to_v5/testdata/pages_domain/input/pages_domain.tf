# Basic pages_domain resource
resource "cloudflare_pages_domain" "basic" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "example.com"
}

# With project reference
resource "cloudflare_pages_project" "example" {
  account_id        = "f037e56e89293a057740de681ac9abbe"
  name              = "my-project"
  production_branch = "main"
}

resource "cloudflare_pages_domain" "with_reference" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = cloudflare_pages_project.example.name
  domain       = "prod.example.com"
}

# Multiple domains for same project
resource "cloudflare_pages_domain" "staging" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "staging.example.com"
}

resource "cloudflare_pages_domain" "dev" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "dev.example.com"
}

# With variables
variable "cloudflare_account_id" {
  type = string
}

variable "project_name" {
  type = string
}

variable "domain_name" {
  type = string
}

resource "cloudflare_pages_domain" "with_vars" {
  account_id   = var.cloudflare_account_id
  project_name = var.project_name
  domain       = var.domain_name
}

# Using for_each with map
variable "domains" {
  type = map(string)
  default = {
    prod    = "prod.example.com"
    staging = "staging.example.com"
    dev     = "dev.example.com"
  }
}

resource "cloudflare_pages_domain" "for_each_map" {
  for_each = var.domains

  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = each.value
}

# Using for_each with set
variable "domain_list" {
  type = set(string)
  default = [
    "api.example.com",
    "www.example.com",
  ]
}

resource "cloudflare_pages_domain" "for_each_set" {
  for_each = var.domain_list

  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = each.value
}

# Using count
resource "cloudflare_pages_domain" "with_count" {
  count = 2

  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "domain${count.index}.example.com"
}

# With conditional creation
variable "create_custom_domain" {
  type    = bool
  default = true
}

resource "cloudflare_pages_domain" "conditional" {
  count = var.create_custom_domain ? 1 : 0

  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "conditional.example.com"
}

# With locals
locals {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  base_domain  = "example.com"
}

resource "cloudflare_pages_domain" "with_locals" {
  account_id   = local.account_id
  project_name = local.project_name
  domain       = "app.${local.base_domain}"
}
