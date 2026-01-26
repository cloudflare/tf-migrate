# Integration tests for zero_trust_device_managed_networks v4→v5 migration
# Tests comprehensive patterns: for_each, count, conditionals, variables

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (not used by this resource)"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (not used by this resource)"
  type        = string
}

locals {
  name_prefix = "cftftest"

  networks_map = {
    "corporate" = {
      host = "corporate.cf-tf-test.com"
      port = "443"
      hash = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
    }
    "staging" = {
      host = "staging.cf-tf-test.com"
      port = "8443"
      hash = "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"
    }
    "production" = {
      host = "prod.cf-tf-test.com"
      port = "443"
      hash = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
    }
  }
}

resource "cloudflare_device_managed_networks" "basic" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-basic-network"
  type       = "tls"

  config {
    tls_sockaddr = "basic.cf-tf-test.com:443"
    sha256       = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
  }
}

resource "cloudflare_device_managed_networks" "map_foreach" {
  for_each = local.networks_map

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${each.key}"
  type       = "tls"

  config {
    tls_sockaddr = "${each.value.host}:${each.value.port}"
    sha256       = each.value.hash
  }
}

resource "cloudflare_device_managed_networks" "set_foreach" {
  for_each = toset(["alpha", "beta", "gamma", "delta"])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-set-${each.value}"
  type       = "tls"

  config {
    tls_sockaddr = "${each.value}.cf-tf-test.com:443"
    sha256       = "1111111111111111111111111111111111111111111111111111111111111111"
  }
}

resource "cloudflare_device_managed_networks" "counted" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-counted-${count.index}"
  type       = "tls"

  config {
    tls_sockaddr = "counted-${count.index}.cf-tf-test.com:443"
    sha256       = "2222222222222222222222222222222222222222222222222222222222222222"
  }
}

locals {
  enable_test_network = true
  enable_dev_network  = false
}

resource "cloudflare_device_managed_networks" "conditional_enabled" {
  count = local.enable_test_network ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-enabled"
  type       = "tls"

  config {
    tls_sockaddr = "conditional.cf-tf-test.com:443"
    sha256       = "3333333333333333333333333333333333333333333333333333333333333333"
  }
}

resource "cloudflare_device_managed_networks" "conditional_disabled" {
  count = local.enable_dev_network ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-conditional-disabled"
  type       = "tls"

  config {
    tls_sockaddr = "disabled.cf-tf-test.com:443"
    sha256       = "4444444444444444444444444444444444444444444444444444444444444444"
  }
}

resource "cloudflare_device_managed_networks" "with_functions" {
  account_id = var.cloudflare_account_id
  name       = join("-", [local.name_prefix, "functions", "test"])
  type       = "tls"

  config {
    tls_sockaddr = "functions.cf-tf-test.com:443"
    sha256       = "functions1234567890abcdef1234567890abcdef1234567890abcdef1234567"
  }
}

resource "cloudflare_device_managed_networks" "ipv6" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-ipv6-network"
  type       = "tls"

  config {
    tls_sockaddr = "[2001:db8::1]:443"
    sha256       = "ipv61234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
  }
}

resource "cloudflare_device_managed_networks" "custom_port" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-custom-port"
  type       = "tls"

  config {
    tls_sockaddr = "custom-port.cf-tf-test.com:8443"
    sha256       = "5555555555555555555555555555555555555555555555555555555555555555"
  }
}

resource "cloudflare_device_managed_networks" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-lifecycle-test"
  type       = "tls"

  config {
    tls_sockaddr = "lifecycle.cf-tf-test.com:443"
    sha256       = "6666666666666666666666666666666666666666666666666666666666666666"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_device_managed_networks" "special_hash" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-special-hash"
  type       = "tls"

  config {
    tls_sockaddr = "special.cf-tf-test.com:443"
    sha256       = "ABCDEF1234567890abcdef1234567890ABCDEF1234567890abcdef1234567890"
  }
}

variable "custom_network_name" {
  type    = string
  default = "custom"
}

resource "cloudflare_device_managed_networks" "with_variables" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${var.custom_network_name}"
  type       = "tls"

  config {
    tls_sockaddr = "${var.custom_network_name}.cf-tf-test.com:443"
    sha256       = "7777777777777777777777777777777777777777777777777777777777777777"
  }
}
