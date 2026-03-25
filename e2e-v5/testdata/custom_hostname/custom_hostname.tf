variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_domain" {
  type = string
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

resource "cloudflare_custom_hostname" "basic" {
  zone_id  = var.cloudflare_zone_id
  hostname = "${local.name_prefix}-app.${var.cloudflare_domain}"

  custom_metadata = {
    environment = "test"
  }

  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = false
    settings = {
      http2           = "on"
      min_tls_version = "1.2"
      tls_1_3         = "on"
    }
  }
}

resource "cloudflare_custom_hostname" "no_settings" {
  zone_id  = var.cloudflare_zone_id
  hostname = "${local.name_prefix}-nosettings.${var.cloudflare_domain}"

  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = false
  }
}

resource "cloudflare_custom_hostname" "wildcard_true" {
  zone_id  = var.cloudflare_zone_id
  hostname = "${local.name_prefix}-wildcard-true.${var.cloudflare_domain}"

  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = true
    settings = {
      tls_1_3 = "on"
    }
  }
}

resource "cloudflare_custom_hostname" "wildcard_false" {
  zone_id  = var.cloudflare_zone_id
  hostname = "${local.name_prefix}-wildcard-false.${var.cloudflare_domain}"

  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = false
    settings = {
      http2 = "on"
    }
  }
}

resource "cloudflare_custom_hostname" "settings_passthrough" {
  zone_id  = var.cloudflare_zone_id
  hostname = "${local.name_prefix}-settings.${var.cloudflare_domain}"

  custom_metadata = {
    environment = "test"
    owner       = "terraform"
  }

  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = false
    settings = {
      http2           = "on"
      min_tls_version = "1.2"
      early_hints     = "off"
      ciphers         = ["ECDHE-RSA-AES128-GCM-SHA256"]
      tls_1_3         = "on"
    }
  }
}
