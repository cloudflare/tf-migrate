variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# ========================================
# Tunnel Resources (Referenced by Routes Below)
# ========================================
# These tunnel resources are included to support cross-resource
# references in the tunnel_route resources below

# Basic tunnel with minimal configuration
resource "cloudflare_tunnel" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "route-minimal-tunnel"
  secret     = base64encode("test-secret-that-is-at-least-32-bytes-long")
  config_src = "local"
}

# Tunnel with local config source
resource "cloudflare_tunnel" "local_config" {
  account_id = var.cloudflare_account_id
  name       = "route-local-config-tunnel"
  secret     = base64encode("another-secret-32-bytes-or-longer-here")
  config_src = "local"
}

# Tunnel with cloudflare config source
resource "cloudflare_tunnel" "cloudflare_config" {
  account_id = var.cloudflare_account_id
  name       = "route-cloudflare-config-tunnel"
  secret     = base64encode("remote-tunnel-secret-32-bytes-minimum")
  config_src = "cloudflare"
}

# ========================================
# Tunnel Resources - Advanced Patterns
# ========================================

# Variable references for tunnels
variable "tunnel_prefix" {
  type    = string
  default = "var-test"
}

variable "config_source" {
  type    = string
  default = "local"
}

# Local values with expressions for tunnels
locals {
  tunnel_secret_base = "generated-secret-that-is-32-bytes-long"
  common_account_id  = var.cloudflare_account_id
  tunnel_suffix      = "prod"
  full_tunnel_name   = "${var.tunnel_prefix}-${local.tunnel_suffix}"
}

# Tunnel using variables and locals
resource "cloudflare_tunnel" "with_vars" {
  account_id = local.common_account_id
  name       = "route-${local.full_tunnel_name}"
  secret     = base64encode(local.tunnel_secret_base)
  config_src = var.config_source
}

# for_each tunnels with map
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

resource "cloudflare_tunnel" "applications" {
  for_each = var.application_tunnels

  account_id = var.cloudflare_account_id
  name       = "route-${each.key}-tunnel"
  secret     = base64encode(each.value.secret)
  config_src = each.value.config_src
}

# for_each tunnels with list converted to set
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

resource "cloudflare_tunnel" "environments" {
  for_each = { for idx, tunnel in var.environment_tunnels : tunnel.name => tunnel }

  account_id = var.cloudflare_account_id
  name       = "route-${each.value.name}-env-tunnel"
  secret     = base64encode(each.value.secret)
  config_src = each.value.config_src
}

# Count-based tunnel resources
variable "replica_count" {
  type    = number
  default = 3
}

resource "cloudflare_tunnel" "replicas" {
  count = var.replica_count

  account_id = var.cloudflare_account_id
  name       = "route-replica-tunnel-${count.index + 1}"
  secret     = base64encode("replica-${count.index}-secret-32-bytes-long")
  config_src = "local"
}

# Conditional tunnel creation
variable "enable_backup_tunnel" {
  type    = bool
  default = true
}

resource "cloudflare_tunnel" "backup" {
  count = var.enable_backup_tunnel ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "route-backup-tunnel"
  secret     = base64encode("backup-tunnel-secret-32-bytes-long")
  config_src = "cloudflare"
}

# Cross-resource references between tunnels
resource "cloudflare_tunnel" "primary" {
  account_id = var.cloudflare_account_id
  name       = "route-primary-tunnel"
  secret     = base64encode("primary-tunnel-secret-32-bytes-long")
  config_src = "local"
}

resource "cloudflare_tunnel" "secondary" {
  account_id = var.cloudflare_account_id
  name       = "${cloudflare_tunnel.primary.name}-secondary"
  secret     = base64encode("secondary-tunnel-secret-32-bytes-ok")
  config_src = "cloudflare"
}

# Tunnel with lifecycle meta-arguments
resource "cloudflare_tunnel" "protected" {
  account_id = var.cloudflare_account_id
  name       = "route-protected-tunnel"
  secret     = base64encode("protected-tunnel-secret-32-bytes-ok")
  config_src = "local"

  lifecycle {
    prevent_destroy       = false
    create_before_destroy = true
  }
}

# Tunnel using terraform expressions
variable "use_cloudflare_config" {
  type    = bool
  default = false
}

resource "cloudflare_tunnel" "conditional_config" {
  account_id = var.cloudflare_account_id
  name       = "route-conditional-config-tunnel"
  secret     = base64encode("conditional-secret-32-bytes-or-more")
  config_src = var.use_cloudflare_config ? "cloudflare" : "local"
}

# Tunnel with base64encode function
resource "cloudflare_tunnel" "encoded" {
  account_id = var.cloudflare_account_id
  name       = "route-encoded-tunnel"
  secret     = base64encode("this-secret-is-base64-encoded-32b")
  config_src = "local"
}

# Tunnel using string interpolation
resource "cloudflare_tunnel" "interpolated" {
  account_id = var.cloudflare_account_id
  name       = "route-${var.tunnel_prefix}-interpolated-${local.tunnel_suffix}"
  secret     = base64encode("interpolated-secret-32-bytes-or-more")
  config_src = "local"
}

# Complex expression for config_src
variable "is_production" {
  type    = bool
  default = true
}

resource "cloudflare_tunnel" "complex_config" {
  account_id = var.cloudflare_account_id
  name       = "route-${var.is_production ? "prod" : "dev"}-complex-tunnel"
  secret     = base64encode("complex-tunnel-secret-32-bytes-long")
  config_src = var.is_production ? "cloudflare" : "local"
}

# ========================================
# Tunnel Route Resources
# ========================================
# These resources reference the tunnels defined above

# ========================================
# Basic Resource Configurations
# ========================================

# Test Case 1: Minimal resource referencing minimal tunnel
resource "cloudflare_tunnel_route" "minimal" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.minimal.id
  network    = "10.0.0.0/16"
}

# Test Case 2: Full resource with all optional fields referencing local_config tunnel
resource "cloudflare_tunnel_route" "full" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.local_config.id
  network    = "172.16.0.0/12"
  comment    = "Production tunnel route for internal services"
}

# Test Case 3: IPv6 network with comment referencing cloudflare_config tunnel
resource "cloudflare_tunnel_route" "ipv6" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.cloudflare_config.id
  network    = "2001:db8::/32"
  comment    = "IPv6 tunnel route"
}

# Test Case 4: Empty comment referencing with_vars tunnel
resource "cloudflare_tunnel_route" "empty_comment" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.with_vars.id
  network    = "192.168.0.0/16"
  comment    = ""
}

# Test Case 5: Special characters in comment referencing primary tunnel
resource "cloudflare_tunnel_route" "special_chars" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.primary.id
  network    = "10.1.0.0/16"
  comment    = "Route with special chars: !@#$%^&*() and unicode: éçà"
}

# ========================================
# Variable-Driven Configurations
# ========================================

variable "tunnel_networks" {
  type = map(object({
    network = string
    comment = string
  }))
  default = {
    "staging" = {
      network = "10.10.0.0/16"
      comment = "Staging environment"
    }
    "development" = {
      network = "10.20.0.0/16"
      comment = "Development environment"
    }
  }
}

# ========================================
# Local Values with Expressions
# ========================================

locals {
  # Network configuration
  network_prefix = "10.100"
  base_comment   = "Automated tunnel route"

  # Computed values
  primary_network = "${local.network_prefix}.0.0/16"
  backup_network  = "${local.network_prefix}.128.0/17"

  # Tags for routes
  environment_tags  = ["production", "automated", "managed"]
  route_description = "${local.base_comment} - ${local.environment_tags[0]}"
}

# ========================================
# Production-Like Patterns
# ========================================

# Pattern 1: for_each with map referencing applications tunnels
resource "cloudflare_tunnel_route" "environment_routes" {
  for_each = var.tunnel_networks

  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.applications[each.key == "staging" ? "api" : "web"].id
  network    = each.value.network
  comment    = "${each.key}: ${each.value.comment}"
}

# Pattern 2: for_each with set conversion referencing environment tunnels
variable "additional_networks" {
  type = list(object({
    env     = string
    network = string
    comment = string
  }))
  default = [
    {
      env     = "dev"
      network = "10.30.0.0/16"
      comment = "Additional network 1"
    },
    {
      env     = "staging"
      network = "10.40.0.0/16"
      comment = "Additional network 2"
    }
  ]
}

resource "cloudflare_tunnel_route" "additional_routes" {
  for_each = { for idx, net in var.additional_networks : net.network => net }

  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.environments[each.value.env].id
  network    = each.value.network
  comment    = each.value.comment
}

# Pattern 3: Count-based resources referencing replicas
variable "subnet_count" {
  type    = number
  default = 3
}

resource "cloudflare_tunnel_route" "subnet_routes" {
  count = var.subnet_count

  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.replicas[count.index].id
  network    = "10.${count.index + 50}.0.0/24"
  comment    = "Subnet ${count.index + 1} route"
}

# Pattern 4: Conditional resources using locals and backup tunnel
locals {
  enable_backup_routes = true
  backup_networks = local.enable_backup_routes ? [
    "10.200.0.0/16",
    "10.201.0.0/16"
  ] : []
}

resource "cloudflare_tunnel_route" "backup_routes" {
  for_each = toset(local.backup_networks)

  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.backup[0].id
  network    = each.value
  comment    = "Backup route for ${each.value}"
}

# ========================================
# Edge Cases and Complex Scenarios
# ========================================

# Test Case 6: Private IPv4 ranges referencing secondary tunnel
resource "cloudflare_tunnel_route" "private_ranges" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.secondary.id
  network    = "172.31.0.0/16"
  comment    = "Private range 172.16.0.0/12"
}

# Test Case 7: Large CIDR (small subnet) referencing protected tunnel
resource "cloudflare_tunnel_route" "small_subnet" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.protected.id
  network    = "192.168.1.0/28"
  comment    = "Small subnet /28"
}

# Test Case 8: IPv6 with virtual network referencing conditional_config tunnel
resource "cloudflare_tunnel_route" "ipv6_with_vnet" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.conditional_config.id
  network    = "fd00::/8"
  comment    = "IPv6 private network"
}

# Test Case 9: Multiple character encodings in comment referencing encoded tunnel
resource "cloudflare_tunnel_route" "unicode_comment" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.encoded.id
  network    = "10.250.0.0/16"
  comment    = "Multi-language: English, Español, 中文, Русский, العربية"
}

# Test Case 10: Comment at max length (100 chars) referencing interpolated tunnel
resource "cloudflare_tunnel_route" "long_comment" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.interpolated.id
  network    = "10.255.0.0/16"
  comment    = "This comment is exactly one hundred characters long to test the maximum API length constraint limit"
}

# Test Case 11: Computed values with expressions referencing complex_config tunnel
resource "cloudflare_tunnel_route" "computed_values" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.complex_config.id
  network    = "10.60.0.0/16"
  comment    = "Computed from locals"
}

# Test Case 12: Route with cross-reference to tunnel name
resource "cloudflare_tunnel_route" "cross_reference" {
  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_tunnel.minimal.id
  network    = "10.245.0.0/16"
  comment    = "Route for tunnel: ${cloudflare_tunnel.minimal.name}"
}
