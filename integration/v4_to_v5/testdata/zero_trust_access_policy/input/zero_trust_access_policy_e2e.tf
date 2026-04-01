# E2E test for zero_trust_access_policy migration
# Excludes resources that can't be created in the e2e environment:
# - service_token_policy, multi_service_token_policy, combined_research_team_policy:
#   require real cloudflare_access_service_token resources
# - app_scoped_policy: requires a real domain/application setup (TKT-003)
# - test_token, test_token_2: excluded since dependent policies are excluded
#
# The TKT-004 service_token fix and TKT-003 application_id handling
# are validated by integration tests only.

# Variables
variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare Domain"
  type        = string
}

locals {
  name_prefix    = "cftftest"
  common_account = var.cloudflare_account_id
  policy_names   = ["dev", "staging", "prod"]
  enable_test    = true
  enable_demo    = false
}

# NOTE: In v5, cloudflare_zero_trust_access_policy are account-level policies.
# The migration removes zone_id and application_id fields from v4 policies.
# Application-specific policies (with application_id in v4) cannot be migrated
# as they use different API endpoints and are fundamentally different resources.

# Basic test cases
resource "cloudflare_access_policy" "example" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-example-policy"
  decision         = "allow"
  session_duration = "24h"

  approval_group {
    approvals_needed = 1
    email_addresses  = ["approver@example.com"]
  }

  include {
    everyone = true
  }
}

resource "cloudflare_access_policy" "complex" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-complex-policy"
  decision         = "allow"
  session_duration = "24h"

  include {
    email        = ["user@example.com", "admin@example.com"]
    email_domain = ["example.com"]
  }

  exclude {
    ip = ["192.168.1.1", "10.0.0.0/8"]
  }

  require {
    email = ["required@example.com"]
  }
}

resource "cloudflare_access_policy" "common_names_overflow" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-common-names-overflow"
  decision         = "allow"
  session_duration = "24h"

  include {
    common_names = ["device2.example.com", "device3.example.com"]
  }
}

# Minimal resource (only required fields)
resource "cloudflare_access_policy" "minimal" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-minimal"
  decision         = "allow"
  session_duration = "24h"

  include {
    everyone = true
  }
}

# Maximal resource (all optional fields populated)
resource "cloudflare_access_policy" "maximal" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-maximal"
  decision         = "allow"
  session_duration = "24h"

  approval_group {
    approvals_needed = 2
    email_addresses  = ["approver1@example.com", "approver2@example.com"]
  }

  include {
    email        = ["maximal1@example.com", "maximal2@example.com"]
    email_domain = ["maximal.example.com"]
    geo          = ["US", "CA"]
    ip           = ["203.0.113.0/24"]
  }

  exclude {
    email = ["blocked@example.com"]
    ip    = ["203.0.113.100"]
  }

  require {
    email = ["required@example.com"]
  }
}

# TKT-002 + TKT-005: email_domain conversion (block → list, list → {domain=...})
resource "cloudflare_access_policy" "email_domain_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-email-domain"
  decision         = "allow"
  session_duration = "18h"

  include {
    email_domain = ["cloudflare.com"]
  }
}

# TKT-006: any_valid_service_token true → {}
resource "cloudflare_access_policy" "any_service_token_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-any-service-token"
  decision         = "non_identity"
  session_duration = "18h"

  include {
    any_valid_service_token = true
  }
}

# TKT-006: any_valid_service_token = false should be omitted, email_domain kept
resource "cloudflare_access_policy" "no_service_token_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-no-service-token"
  decision         = "allow"
  session_duration = "18h"

  include {
    any_valid_service_token = false
    email_domain            = ["cloudflare.com"]
  }
}

# TKT-005: multiple email domains
resource "cloudflare_access_policy" "multi_email_domain_policy" {
  account_id       = var.cloudflare_account_id
  name             = "${local.name_prefix}-multi-email-domain"
  decision         = "allow"
  session_duration = "18h"

  include {
    email_domain = ["cloudflare.com", "example.com"]
  }
}
