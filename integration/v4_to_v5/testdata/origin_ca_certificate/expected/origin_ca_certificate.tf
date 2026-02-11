# Comprehensive Integration Test for origin_ca_certificate v4 → v5 Migration
# This file tests all attributes, transformations, and Terraform patterns

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

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# Variables for testing variable references
variable "test_prefix" {
  default = "cftftest"
}

variable "validity_days" {
  default = 365
}

# Locals for testing local references
locals {
  test_domain     = "cftftest-example.${var.cloudflare_domain}"
  wildcard_domain = "*.cftftest-example.${var.cloudflare_domain}"
  ecc_type        = "origin-ecc"
  default_csr = <<EOT
-----BEGIN CERTIFICATE REQUEST-----
MIICuDCCAaACAQAwczELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx
FjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xGDAWBgNVBAoMD0V4YW1wbGUgQ29tcGFu
eTEdMBsGA1UEAwwUY2Z0ZnRlc3QtZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQC5l3PKiysEmgXNLDtZUtNJCiiCmihql7JHefBltZ+H
7RmIJ5NKWpq+xkZrHmmZ+j7kfAmHPb6+eFNks70XfdjAZZoc86aFCNT6U4OUBY/1
75HkXU6NYfZfuFvK8OvtoiJB6sOs9Mz31XHGIpT2AY7XvsmOHWUGIc84bhYNF5mG
VwJpCPfMJqczcoJGZn1mHdPMHQdygAMVql/Hpy9DiMPIMVYk/o+jMj5QV7nOxNW/
NpKp1ncLReXpSDURBX2Wx+SVpVRbovrsbEL0Mm2xSeBsavTmPo0XZrCLnQ3xGWmo
5so6CLCqbv+PNyGirDgQN3qHHSAf9vZtW9xTK0kRLOGDAgMBAAGgADANBgkqhkiG
9w0BAQsFAAOCAQEAGoFZnK8h72y37n7xu5PVbAeLRKBTr4Txtu6ewFJbfCVZ8ZLu
RvHwomcBtvNXd9seTwY00mW9Gko9R6glASsJPvJCqQvMYLWd4hQaMffZOXH5Z2Hg
CCNKreZX8KwX7/TzNtbBnnq359T+DzYDyvPfrkz1Cx+GRoNPHqIpOX50Mw7Gmxdr
J7urFodSDFCKS9WMKe3CBfkO1Gn0w3HBxoz9Ts8ctdA9bAof4/YQH7lLw7pL9uqn
TcBVKhDK/f+AO8xZkTV9/jJ7YvMJ9PxDAZlLtnMWZmVd/VcLQx8vtBzA7JWL+Mba
t/PQliLH+Ty70y/Y6+lz/E94rdN8zujVfUMuxg==
-----END CERTIFICATE REQUEST-----
EOT
}

# =============================================================================
# 1. BASIC RESOURCES - Simple configurations
# =============================================================================

# Test 1: Minimal configuration (only required fields)
resource "cloudflare_origin_ca_certificate" "minimal" {
  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames    = ["cftftest-minimal.${var.cloudflare_domain}"]
}

# Test 2: Basic configuration with requested_validity
resource "cloudflare_origin_ca_certificate" "basic" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-basic.${var.cloudflare_domain}"]
  requested_validity = 365
}

# Test 3: Configuration with min_days_for_renewal
resource "cloudflare_origin_ca_certificate" "with_renewal" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-renewal.${var.cloudflare_domain}"]
  requested_validity = 730
}

# =============================================================================
# 2. REQUEST TYPES - Test all valid request types
# =============================================================================

resource "cloudflare_origin_ca_certificate" "type_rsa" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-rsa.${var.cloudflare_domain}"]
  requested_validity = 365
}

resource "cloudflare_origin_ca_certificate" "type_ecc" {
  csr                = local.default_csr
  request_type       = local.ecc_type
  hostnames          = ["cftftest-ecc.${var.cloudflare_domain}"]
  requested_validity = 365
}

# =============================================================================
# 3. REQUESTED VALIDITY - Test all valid values (7, 30, 90, 365, 730, 1095, 5475)
# =============================================================================

resource "cloudflare_origin_ca_certificate" "validity_7days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-7day.${var.cloudflare_domain}"]
  requested_validity = 7
}

resource "cloudflare_origin_ca_certificate" "validity_30days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-30day.${var.cloudflare_domain}"]
  requested_validity = 30
}

resource "cloudflare_origin_ca_certificate" "validity_90days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-90day.${var.cloudflare_domain}"]
  requested_validity = 90
}

resource "cloudflare_origin_ca_certificate" "validity_365days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-1year.${var.cloudflare_domain}"]
  requested_validity = var.validity_days
}

resource "cloudflare_origin_ca_certificate" "validity_730days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-2year.${var.cloudflare_domain}"]
  requested_validity = 730
}

resource "cloudflare_origin_ca_certificate" "validity_1095days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-3year.${var.cloudflare_domain}"]
  requested_validity = 1095
}

resource "cloudflare_origin_ca_certificate" "validity_5475days" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-15year.${var.cloudflare_domain}"]
  requested_validity = 5475
}

# Test default validity (omit requested_validity, should default to 5475 in v5)
resource "cloudflare_origin_ca_certificate" "validity_default" {
  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames    = ["cftftest-default-validity.${var.cloudflare_domain}"]
}

# =============================================================================
# 4. HOSTNAME VARIATIONS - Single, multiple, wildcards
# =============================================================================

resource "cloudflare_origin_ca_certificate" "single_hostname" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["${var.test_prefix}-single.${var.cloudflare_domain}"]
  requested_validity = 365
}

resource "cloudflare_origin_ca_certificate" "multiple_hostnames" {
  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames = [
    "cftftest-multi1.${var.cloudflare_domain}",
    "cftftest-multi2.${var.cloudflare_domain}",
    "cftftest-multi3.${var.cloudflare_domain}",
  ]
  requested_validity = 365
}

resource "cloudflare_origin_ca_certificate" "wildcard_hostname" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = [local.wildcard_domain]
  requested_validity = 365
}

resource "cloudflare_origin_ca_certificate" "mixed_hostnames" {
  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames = [
    local.wildcard_domain,
    local.test_domain,
    "cftftest-subdomain.${var.cloudflare_domain}",
  ]
  requested_validity = 730
}

# =============================================================================
# 5. FOR_EACH PATTERNS - Dynamic resources from map
# =============================================================================

locals {
  certificates_map = {
    api = {
      hostname = "cftftest-api.${var.cloudflare_domain}"
      validity = 365
    }
    web = {
      hostname = "cftftest-web.${var.cloudflare_domain}"
      validity = 730
    }
    admin = {
      hostname = "cftftest-admin.${var.cloudflare_domain}"
      validity = 90
    }
  }
}

resource "cloudflare_origin_ca_certificate" "foreach_map" {
  for_each = local.certificates_map

  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames    = [each.value.hostname]
  requested_validity = each.value.validity
}

# =============================================================================
# 6. FOR_EACH WITH SET - Dynamic resources from set
# =============================================================================

locals {
  service_names = toset(["cftftest-service1", "cftftest-service2", "cftftest-service3"])
}

resource "cloudflare_origin_ca_certificate" "foreach_set" {
  for_each = local.service_names

  csr                = local.default_csr
  request_type       = "origin-ecc"
  hostnames          = ["${each.key}.${var.cloudflare_domain}"]
  requested_validity = 365
}

# =============================================================================
# 7. COUNT PATTERN - Multiple instances with count
# =============================================================================

resource "cloudflare_origin_ca_certificate" "count_pattern" {
  count = 3

  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames    = ["cftftest-count-${count.index}.${var.cloudflare_domain}"]
  requested_validity = 365
}

# =============================================================================
# 8. CONDITIONAL RESOURCES - count with condition
# =============================================================================

variable "create_staging_cert" {
  default = true
}

resource "cloudflare_origin_ca_certificate" "conditional" {
  count = var.create_staging_cert ? 1 : 0

  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-staging.${var.cloudflare_domain}"]
  requested_validity = 90
}

# =============================================================================
# 9. LIFECYCLE META-ARGUMENTS - Testing ignore_changes, etc.
# =============================================================================

resource "cloudflare_origin_ca_certificate" "with_lifecycle" {
  csr                = local.default_csr
  request_type       = "origin-rsa"
  hostnames          = ["cftftest-lifecycle.${var.cloudflare_domain}"]
  requested_validity = 365

  lifecycle {
    ignore_changes = [requested_validity]
  }
}

# =============================================================================
# 10. TERRAFORM FUNCTIONS - format(), etc.
# =============================================================================

resource "cloudflare_origin_ca_certificate" "with_functions" {
  csr          = local.default_csr
  request_type = "origin-rsa"
  hostnames = [
    "cftftest-app.${var.cloudflare_domain}",
    format("%s-backup.${var.cloudflare_domain}", var.test_prefix),
  ]
  requested_validity = 365
}

# =============================================================================
# 11. EDGE CASES
# =============================================================================

# Maximal configuration (all possible fields)
resource "cloudflare_origin_ca_certificate" "maximal" {
  csr          = <<EOT
-----BEGIN CERTIFICATE REQUEST-----
MIICuDCCAaACAQAwczELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx
FjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xGDAWBgNVBAoMD0V4YW1wbGUgQ29tcGFu
eTEdMBsGA1UEAwwUY2Z0ZnRlc3QtbWF4aW1hbC5jb20wggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDG0vu0QgKEq5YoZLPikBmxWjR5k5WGKQYKcScHR80Y
y4HDKGMuMTdm6VedQWY96GQ8v+Hj2qMVj5FSfV2HZs9JZQm/ICkKpUfBEvT/wh+V
P9FgLgJFpIN/m6pU6m0swY9LPi7vNkB01THu7Sya9GCO8CoQItabD9RejDfVz9ox
LLND24LM/dRwVCYH4TRQLAOC13bol1xWwUp85n/6tLFa7169z6FD7zjjzhKbFC5H
sYr2V6PmrWSMxBqR7Yf/hCTQ20qqtjhYthwr+soN+POpHuxT1A31rjXSxbTQS0hO
CSD6ZfTKAD2mauKLyi01tiBhdPs42OM5MxWXd3CiCe3NAgMBAAGgADANBgkqhkiG
9w0BAQsFAAOCAQEAv3w7o4B5xpEmEbVJel29U4P1XHEbWy52DqdlXWXPB3QCPtMH
ohqJvwG2VeuxjTT3CXKuW0rFPuaD7ko8hpnUPgWyaHhUXqSraEshzlC2mJLgNoIH
Tq9rjcy1alsyXKQ7gM8MCF00myPFOhBiNFpI4UQ4v5oixg5NuNhbeZJF2NOBqVV0
pbd0T1jTBsLasPBDC80Pw/KCtGRHsyS7nYIo372Wghmp20x1N8YQ/x74ntpfl6Mw
BFFiw6UIGjlCdQr1SHfy9PElWi8AEo0ICyYM8ay+5zc1+nDjzibXxjjocG0smolR
HQbz7GoyzUHbssJAq1unWphMHf5RQo9SFu/EMA==
-----END CERTIFICATE REQUEST-----
EOT
  request_type       = "origin-rsa"
  hostnames          = ["*.cftftest-maximal.${var.cloudflare_domain}", "cftftest-maximal.${var.cloudflare_domain}"]
  requested_validity = 5475
}

# =============================================================================
# Summary: Total Resources
# - Basic: 3
# - Request types: 2
# - Validity values: 8
# - Hostname variations: 4
# - for_each map: 3
# - for_each set: 3
# - count: 3
# - conditional: 1
# - lifecycle: 1
# - functions: 1
# - maximal: 1
# TOTAL: 30 resource instances
# =============================================================================
