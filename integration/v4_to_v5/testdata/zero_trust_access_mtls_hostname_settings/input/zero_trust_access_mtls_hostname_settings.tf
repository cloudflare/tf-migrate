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

# ========================================
# Basic Resources (Account-level)
# ========================================

# Minimal configuration - single settings block
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "minimal_account" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-minimal.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}

# With all fields populated
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "full_account" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-full.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = true
  }
}

# Multiple settings blocks
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "multiple_account" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-api.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = false
  }

  settings {
    hostname                      = "cftftest-web.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = true
  }

  settings {
    hostname                      = "cftftest-admin.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}

# ========================================
# Basic Resources (Zone-level)
# ========================================

# Minimal configuration - zone-level
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "minimal_zone" {
  zone_id = var.cloudflare_zone_id

  settings {
    hostname                      = "cftftest-zone-minimal.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}

# Zone-level with multiple settings
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "multiple_zone" {
  zone_id = var.cloudflare_zone_id

  settings {
    hostname                      = "cftftest-zone-api.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = true
  }

  settings {
    hostname                      = "cftftest-zone-web.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}

# ========================================
# Advanced Terraform Patterns
# ========================================

# Pattern 1: Variable references
variable "hostname_prefix" {
  type    = string
  default = "cftftest-var"
}

variable "enable_china_network" {
  type    = bool
  default = true
}

variable "enable_cert_forwarding" {
  type    = bool
  default = false
}

# Pattern 2: Local values with expressions
locals {
  name_prefix       = "cftftest"
  base_domain       = "cf-tf-test.com"
  environment       = "prod"
  full_hostname     = "${local.name_prefix}-${local.environment}.${local.base_domain}"
  common_account_id = var.cloudflare_account_id
}

# Using variables and locals
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "with_vars" {
  account_id = local.common_account_id

  settings {
    hostname                      = local.full_hostname
    china_network                 = var.enable_china_network
    client_certificate_forwarding = var.enable_cert_forwarding
  }

  settings {
    hostname                      = "${var.hostname_prefix}-backup.${local.base_domain}"
    china_network                 = !var.enable_china_network
    client_certificate_forwarding = !var.enable_cert_forwarding
  }
}

# Pattern 3: for_each with set
variable "hostnames_set" {
  type    = set(string)
  default = ["cftftest-app1.cf-tf-test.com", "cftftest-app2.cf-tf-test.com", "cftftest-app3.cf-tf-test.com"]
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "from_set" {
  for_each = var.hostnames_set

  account_id = var.cloudflare_account_id

  settings {
    hostname                      = each.value
    china_network                 = false
    client_certificate_forwarding = true
  }
}

# Pattern 4: for_each with map
variable "mtls_configs" {
  type = map(object({
    hostname              = string
    china_network         = bool
    cert_forwarding       = bool
  }))
  default = {
    "production" = {
      hostname        = "cftftest-production.cf-tf-test.com"
      china_network   = true
      cert_forwarding = true
    }
    "staging" = {
      hostname        = "cftftest-staging.cf-tf-test.com"
      china_network   = false
      cert_forwarding = false
    }
    "development" = {
      hostname        = "cftftest-development.cf-tf-test.com"
      china_network   = false
      cert_forwarding = true
    }
  }
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "from_map" {
  for_each = var.mtls_configs

  account_id = var.cloudflare_account_id

  settings {
    hostname                      = each.value.hostname
    china_network                 = each.value.china_network
    client_certificate_forwarding = each.value.cert_forwarding
  }
}

# Pattern 5: count
variable "environment_count" {
  type    = number
  default = 3
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "with_count" {
  count = var.environment_count

  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-env${count.index}.cf-tf-test.com"
    china_network                 = count.index == 0 ? true : false
    client_certificate_forwarding = count.index % 2 == 0
  }
}

# Pattern 6: Conditional creation
variable "create_optional_settings" {
  type    = bool
  default = true
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "conditional" {
  count = var.create_optional_settings ? 1 : 0

  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-conditional.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = false
  }
}

# Pattern 7: Complex locals with for expression
locals {
  regional_hostnames = ["us-east", "us-west", "eu-central", "ap-southeast"]
  regional_configs = {
    for region in local.regional_hostnames :
    region => {
      hostname = "${local.name_prefix}-${region}.${local.base_domain}"
      china    = region == "ap-southeast"
      forward  = contains(["us-east", "eu-central"], region)
    }
  }
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "regional" {
  for_each = local.regional_configs

  account_id = var.cloudflare_account_id

  settings {
    hostname                      = each.value.hostname
    china_network                 = each.value.china
    client_certificate_forwarding = each.value.forward
  }
}

# Pattern 8: Multiple settings with dynamic content
variable "services" {
  type = list(object({
    name            = string
    china_enabled   = bool
    forward_enabled = bool
  }))
  default = [
    {
      name            = "auth"
      china_enabled   = true
      forward_enabled = true
    },
    {
      name            = "gateway"
      china_enabled   = false
      forward_enabled = false
    },
    {
      name            = "proxy"
      china_enabled   = false
      forward_enabled = true
    }
  ]
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "services" {
  account_id = var.cloudflare_account_id

  dynamic "settings" {
    for_each = var.services
    content {
      hostname                      = "${local.name_prefix}-${settings.value.name}.${local.base_domain}"
      china_network                 = settings.value.china_enabled
      client_certificate_forwarding = settings.value.forward_enabled
    }
  }
}

# Pattern 9: Lifecycle meta-arguments
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "with_lifecycle" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-lifecycle.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = true
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [settings]
  }
}

# Pattern 10: Using Terraform functions
variable "domain_list" {
  type    = list(string)
  default = ["alpha", "beta", "gamma"]
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "with_functions" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "${local.name_prefix}-${join("-", var.domain_list)}.${local.base_domain}"
    china_network                 = length(var.domain_list) > 2
    client_certificate_forwarding = contains(var.domain_list, "alpha")
  }
}

# Pattern 11: Depends_on meta-argument
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "primary" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-primary.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = true
  }
}

resource "cloudflare_zero_trust_access_mtls_hostname_settings" "secondary" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-secondary.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }

  depends_on = [cloudflare_zero_trust_access_mtls_hostname_settings.primary]
}

# Pattern 12: Mixed china_network values in multiple settings
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "mixed_china" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-china-enabled.cf-tf-test.com"
    china_network                 = true
    client_certificate_forwarding = false
  }

  settings {
    hostname                      = "cftftest-china-disabled.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}

# Pattern 13: Mixed cert forwarding values
resource "cloudflare_zero_trust_access_mtls_hostname_settings" "mixed_forwarding" {
  account_id = var.cloudflare_account_id

  settings {
    hostname                      = "cftftest-forward-enabled.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = true
  }

  settings {
    hostname                      = "cftftest-forward-disabled.cf-tf-test.com"
    china_network                 = false
    client_certificate_forwarding = false
  }
}
