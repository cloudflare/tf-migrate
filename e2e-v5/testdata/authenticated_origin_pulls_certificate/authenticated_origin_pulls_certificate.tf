variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

resource "tls_private_key" "zone" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "zone" {
  private_key_pem       = tls_private_key.zone.private_key_pem
  validity_period_hours = 8760
  is_ca_certificate     = false

  subject {
    common_name  = "${local.name_prefix}-aop-zone.${var.cloudflare_domain}"
    organization = "tf-migrate"
  }

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
    "client_auth",
  ]
}

resource "tls_private_key" "hostname" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "hostname" {
  private_key_pem       = tls_private_key.hostname.private_key_pem
  validity_period_hours = 8760
  is_ca_certificate     = false

  subject {
    common_name  = "${local.name_prefix}-aop-host.${var.cloudflare_domain}"
    organization = "tf-migrate"
  }

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
    "client_auth",
  ]
}

resource "cloudflare_authenticated_origin_pulls_certificate" "per_zone_1" {
  zone_id     = var.cloudflare_zone_id
  certificate = tls_self_signed_cert.zone.cert_pem
  private_key = tls_private_key.zone.private_key_pem
}

resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "per_hostname_1" {
  zone_id     = var.cloudflare_zone_id
  certificate = tls_self_signed_cert.hostname.cert_pem
  private_key = tls_private_key.hostname.private_key_pem
}

output "per_zone_1_status" {
  value = cloudflare_authenticated_origin_pulls_certificate.per_zone_1.status
}
