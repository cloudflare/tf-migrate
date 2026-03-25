variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing (unused by this module)"
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

locals {
  name_prefix = "v5-upgrade-${replace(var.from_version, ".", "-")}"
}

resource "tls_private_key" "ca" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "ca" {
  private_key_pem       = tls_private_key.ca.private_key_pem
  validity_period_hours = 8760
  is_ca_certificate     = true

  subject {
    common_name  = "${local.name_prefix}-mtls-ca"
    organization = "tf-migrate"
  }

  allowed_uses = [
    "cert_signing",
    "crl_signing",
    "digital_signature",
    "key_encipherment",
  ]
}

resource "tls_private_key" "leaf" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "leaf" {
  private_key_pem       = tls_private_key.leaf.private_key_pem
  validity_period_hours = 8760
  is_ca_certificate     = false

  subject {
    common_name  = "${local.name_prefix}-mtls-leaf"
    organization = "tf-migrate"
  }

  allowed_uses = [
    "digital_signature",
    "key_encipherment",
    "server_auth",
    "client_auth",
  ]
}

resource "cloudflare_mtls_certificate" "basic_ca" {
  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = tls_self_signed_cert.ca.cert_pem
  name         = "${local.name_prefix}-basic-ca"
}

resource "cloudflare_mtls_certificate" "basic_leaf" {
  account_id   = var.cloudflare_account_id
  ca           = false
  certificates = tls_self_signed_cert.leaf.cert_pem
  private_key  = tls_private_key.leaf.private_key_pem
  name         = "${local.name_prefix}-basic-leaf"
}
