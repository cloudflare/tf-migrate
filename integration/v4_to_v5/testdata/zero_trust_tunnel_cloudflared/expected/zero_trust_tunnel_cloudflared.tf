variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# ========================================
# Basic Tunnel Resources
# ========================================




# ========================================
# Advanced Terraform Patterns for Testing
# ========================================

# Pattern 1: Variable references
variable "tunnel_prefix" {
  type    = string
  default = "var-test"
}

variable "config_source" {
  type    = string
  default = "local"
}

# Pattern 2: Local values with expressions
locals {
  tunnel_secret_base = "generated-secret-that-is-32-bytes-long"
  common_account_id  = var.cloudflare_account_id
  tunnel_suffix      = "prod"
  full_tunnel_name   = "${var.tunnel_prefix}-${local.tunnel_suffix}"
}


# Pattern 3: for_each with map
variable "application_tunnels" {
  type = map(object({
    secret     = string
    config_src = string
  }))
  default = {
    "api" = {
      secret     = "api-tunnel-secret-that-is-32-bytes"
      config_src = "local"
    }
    "web" = {
      secret     = "web-tunnel-secret-that-is-32-bytes"
      config_src = "cloudflare"
    }
    "backend" = {
      secret     = "backend-secret-that-is-32-bytes-ok"
      config_src = "local"
    }
  }
}


# Pattern 4: for_each with list converted to set
variable "environment_tunnels" {
  type = list(object({
    name       = string
    secret     = string
    config_src = string
  }))
  default = [
    {
      name       = "dev"
      secret     = "dev-environment-secret-32-bytes-min"
      config_src = "local"
    },
    {
      name       = "staging"
      secret     = "staging-environment-secret-32-byte"
      config_src = "cloudflare"
    },
    {
      name       = "prod"
      secret     = "prod-environment-secret-32-bytes-ok"
      config_src = "cloudflare"
    }
  ]
}


# Pattern 5: Count-based resources
variable "replica_count" {
  type    = number
  default = 3
}


# Pattern 6: Conditional resource creation
variable "enable_backup_tunnel" {
  type    = bool
  default = true
}





# Pattern 9: Using terraform expressions
variable "use_cloudflare_config" {
  type    = bool
  default = false
}




# Pattern 12: Complex expression for config_src
variable "is_production" {
  type    = bool
  default = true
}

# Basic tunnel with minimal configuration
resource "cloudflare_zero_trust_tunnel_cloudflared" "minimal" {
  account_id    = var.cloudflare_account_id
  name          = "minimal-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("test-secret-that-is-at-least-32-bytes-long")
}
# Tunnel with local config source
resource "cloudflare_zero_trust_tunnel_cloudflared" "local_config" {
  account_id    = var.cloudflare_account_id
  name          = "local-config-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("another-secret-32-bytes-or-longer-here")
}
# Tunnel with cloudflare config source
resource "cloudflare_zero_trust_tunnel_cloudflared" "cloudflare_config" {
  account_id    = var.cloudflare_account_id
  name          = "cloudflare-config-tunnel"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("remote-tunnel-secret-32-bytes-minimum")
}
# Tunnel using variables and locals
resource "cloudflare_zero_trust_tunnel_cloudflared" "with_vars" {
  account_id    = local.common_account_id
  name          = local.full_tunnel_name
  config_src    = var.config_source
  tunnel_secret = base64encode(local.tunnel_secret_base)
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "applications" {
  for_each = var.application_tunnels

  account_id    = var.cloudflare_account_id
  name          = "${each.key}-tunnel"
  config_src    = each.value.config_src
  tunnel_secret = base64encode(each.value.secret)
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "environments" {
  for_each = { for idx, tunnel in var.environment_tunnels : tunnel.name => tunnel }

  account_id    = var.cloudflare_account_id
  name          = "${each.value.name}-env-tunnel"
  config_src    = each.value.config_src
  tunnel_secret = base64encode(each.value.secret)
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "replicas" {
  count = var.replica_count

  account_id    = var.cloudflare_account_id
  name          = "replica-tunnel-${count.index + 1}"
  config_src    = "local"
  tunnel_secret = base64encode("replica-${count.index}-secret-32-bytes-long")
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "backup" {
  count = var.enable_backup_tunnel ? 1 : 0

  account_id    = var.cloudflare_account_id
  name          = "backup-tunnel"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("backup-tunnel-secret-32-bytes-long")
}
# Pattern 7: Cross-resource references
resource "cloudflare_zero_trust_tunnel_cloudflared" "primary" {
  account_id    = var.cloudflare_account_id
  name          = "primary-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("primary-tunnel-secret-32-bytes-long")
}
# Tunnel that references another tunnel in its name
resource "cloudflare_zero_trust_tunnel_cloudflared" "secondary" {
  account_id    = var.cloudflare_account_id
  name          = "${cloudflare_zero_trust_tunnel_cloudflared.primary.name}-secondary"
  config_src    = "cloudflare"
  tunnel_secret = base64encode("secondary-tunnel-secret-32-bytes-ok")
}
# Pattern 8: Resource with lifecycle meta-arguments
resource "cloudflare_zero_trust_tunnel_cloudflared" "protected" {
  account_id = var.cloudflare_account_id
  name       = "protected-tunnel"
  config_src = "local"

  lifecycle {
    prevent_destroy       = false
    create_before_destroy = true
  }
  tunnel_secret = base64encode("protected-tunnel-secret-32-bytes-ok")
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "conditional_config" {
  account_id    = var.cloudflare_account_id
  name          = "conditional-config-tunnel"
  config_src    = var.use_cloudflare_config ? "cloudflare" : "local"
  tunnel_secret = base64encode("conditional-secret-32-bytes-or-more")
}
# Pattern 10: Tunnel with base64encode function
resource "cloudflare_zero_trust_tunnel_cloudflared" "encoded" {
  account_id    = var.cloudflare_account_id
  name          = "encoded-tunnel"
  config_src    = "local"
  tunnel_secret = base64encode("this-secret-is-base64-encoded-32b")
}
# Pattern 11: Tunnel using string interpolation
resource "cloudflare_zero_trust_tunnel_cloudflared" "interpolated" {
  account_id    = var.cloudflare_account_id
  name          = "${var.tunnel_prefix}-interpolated-${local.tunnel_suffix}"
  config_src    = "local"
  tunnel_secret = base64encode("interpolated-secret-32-bytes-or-more")
}
resource "cloudflare_zero_trust_tunnel_cloudflared" "complex_config" {
  account_id    = var.cloudflare_account_id
  name          = "${var.is_production ? "prod" : "dev"}-complex-tunnel"
  config_src    = var.is_production ? "cloudflare" : "local"
  tunnel_secret = base64encode("complex-tunnel-secret-32-bytes-long")
}
