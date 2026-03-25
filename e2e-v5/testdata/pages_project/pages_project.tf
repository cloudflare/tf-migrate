locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

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

# Minimal baseline project
resource "cloudflare_pages_project" "minimal" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-pages-project-minimal"
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

# Feature-rich project covering build + deployment settings
resource "cloudflare_pages_project" "feature_rich" {
  account_id        = var.cloudflare_account_id
  name              = "${local.name_prefix}-pages-project-feature"
  production_branch = "main"

  build_config = {
    build_command   = "npm run build"
    destination_dir = "dist"
    root_dir        = "/app"
  }

  deployment_configs = {
    preview = {
      compatibility_date = "2024-01-01"
      usage_model        = "bundled"
      fail_open          = false
      placement = {
        mode = "smart"
      }
    }
    production = {
      compatibility_date = "2024-01-01"
      usage_model        = "bundled"
      fail_open          = false
      placement = {
        mode = "smart"
      }
    }
  }
}
