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

# Test Case 1: Minimal Pages Project
resource "cloudflare_pages_project" "minimal" {
  account_id        = var.cloudflare_account_id
  name = "cftftest-minimal-project"
  production_branch = "main"
}

# Test Case 2: Pages Project with build_config block
resource "cloudflare_pages_project" "with_build_config" {
  account_id        = var.cloudflare_account_id
  name = "cftftest-project-with-build"
  production_branch = "main"

  build_config {
    build_command       = "npm run build"
    destination_dir     = "public"
    root_dir            = "/"
    web_analytics_tag   = "abc123"
    web_analytics_token = "token123"
  }
}

# Test Case 3: Pages Project with source and config blocks
resource "cloudflare_pages_project" "with_source" {
  account_id        = var.cloudflare_account_id
  name = "cftftest-project-with-source"
  production_branch = "main"

  #source {
  #  type = "github"
#
#    config {
#      owner                         = "cloudflare"
#      repo_name                     = "test-repo"
#      production_branch             = "main"
#      pr_comments_enabled           = true
#      deployments_enabled           = true
#      production_deployment_enabled = true
#      preview_deployment_setting    = "custom"
#      preview_branch_includes       = ["dev", "staging"]
#      preview_branch_excludes       = ["temp"]
#    }
#  }
}

# Test Case 4: Pages Project with deployment_configs
resource "cloudflare_pages_project" "with_deployment_configs" {
  account_id        = var.cloudflare_account_id
  name = "cftftest-project-with-deployments"
  production_branch = "main"

  deployment_configs {
    preview {
      compatibility_date  = "2024-01-01"
      compatibility_flags = ["nodejs_compat"]
      usage_model         = "bundled"

      placement {
        mode = "smart"
      }
    }

    production {
      compatibility_date  = "2024-01-01"
      compatibility_flags = ["nodejs_compat", "streams_enable_constructors"]
      usage_model         = "bundled"

      placement {
        mode = "smart"
      }
    }
  }
}

# Test Case 5: Full Pages Project with all features
resource "cloudflare_pages_project" "full" {
  account_id        = var.cloudflare_account_id
  name = "cftftest-full-project"
  production_branch = "main"

  build_config {
    build_command   = "npm run build"
    destination_dir = "dist"
    root_dir        = "/app"
  }

  #source {
  #  type = "github"
#
#    config {
#      owner                         = "my-org"
#      repo_name                     = "my-app"
#      production_branch             = "main"
#      production_deployment_enabled = true
#      pr_comments_enabled           = true
#    }
#  }

  deployment_configs {
    preview {
      compatibility_date = "2024-01-01"

      placement {
        mode = "smart"
      }
    }

    production {
      compatibility_date = "2024-01-01"

      placement {
        mode = "smart"
      }
    }
  }
}

# Test Case 6: Project with only deployment configs (no build/source)
resource "cloudflare_pages_project" "deployment_only" {
  account_id        = var.cloudflare_account_id
  name = "cftftest-deployment-only"
  production_branch = "main"

  deployment_configs {
    production {
      compatibility_date = "2024-01-15"

      placement {
        mode = "smart"
      }
    }
    preview {
      compatibility_date = "2024-01-15"

      placement {
        mode = "smart"
      }
    }
  }
}
