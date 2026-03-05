# Integration test for cloudflare_custom_hostname v4 to v5 migration

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

resource "cloudflare_custom_hostname" "basic" {
  zone_id  = var.cloudflare_zone_id
  hostname = "cftftest-app.${var.cloudflare_domain}"


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
  hostname = "cftftest-nosettings.${var.cloudflare_domain}"

  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = false
  }
}

resource "cloudflare_custom_hostname" "wildcard_true" {
  zone_id  = var.cloudflare_zone_id
  hostname = "cftftest-wildcard-true.${var.cloudflare_domain}"

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
  hostname = "cftftest-wildcard-false.${var.cloudflare_domain}"

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
  hostname = "cftftest-settings.${var.cloudflare_domain}"

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
