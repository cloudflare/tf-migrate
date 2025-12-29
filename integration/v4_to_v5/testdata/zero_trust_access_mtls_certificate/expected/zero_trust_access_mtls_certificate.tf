# Comprehensive Integration Tests for zero_trust_access_mtls_certificate
# This file tests v4 to v5 migration with all Terraform patterns

# Variables for DRY configuration
variable "cloudflare_account_id" {
  type    = string
  default = "f037e56e89293a057740de681ac9abbe"
}

variable "cloudflare_zone_id" {
  type    = string
  default = "0da42c8d2132a9ddaf714f9e7c920711"
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing (not used by this module but accepted for consistency)"
}

# Locals for naming consistency
locals {
  name_prefix = "cftftest"
  test_cert   = "-----BEGIN CERTIFICATE-----\nMIIDvTCCAqWgAwIBAgIEaUWrAzANBgkqhkiG9w0BAQsFADB0MQswCQYDVQQGEwJV\nUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAXBgNVBAoT\nEFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRlc3QuZXhh\nbXBsZS5jb20wHhcNMjUxMjE5MTk0NDAzWhcNMjYxMjE5MTk0NDAzWjB0MQswCQYD\nVQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAX\nBgNVBAoTEFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRl\nc3QuZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDa\ndyRoc6ALweI82g5JaoXsaPAnTve5VlL1iFE1Pa6qiJFhVMT+n5+Jr2LaHrcegme6\nBxZOeGDTSaxNRLMYcaozyHoEjwqOs3TGq+wp9ifB5F0eQQpkflEi6XU7rGYaZ8d7\nyaqr55avCQl215tNoZTlFo11g9NhT1EdsAMVKzS3cpPLmNV7/p6GJywuOOmb+uQN\nYC0hT+YKUz4XusIaseGW3zJ6piF4sRvZsFpLqcKLeXvwJkI+MPJOhtuEeEXVUBeZ\n2EUI5aonLkTxnW1wIf43wqB8xwMy7KbuLI+Z2blQeqYgPiCc5AznUEpL50Ktwn/1\nPNA5AxZUGGxrKL7zBi6/AgMBAAGjVzBVMA4GA1UdDwEB/wQEAwICpDATBgNVHSUE\nDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQTeUKqmzM8\nxWP9aa9Mm9C3LiqP7zANBgkqhkiG9w0BAQsFAAOCAQEA0qks6Gjv927SbcC5lJcI\nZqA/S77uB+/7dBv96ILT0s6FWWQ2FaCgf8fsGmG3SLzqRrKRo8FD5D2Hwr8P5r8b\nlbquFs/2LvQ3c6xnjaO4HyhzWJTNL1Zamxjr57TKpPF/2OyEPb7UPIyiiC7XmD1G\nqQRCgFaYHlcOXASE7PN/bHyg6aOWW3kyGJ1lP6HvpVOMC7+jQA5a/RB3ImsUKctD\n2x4RufuaDGiNAi2Taz7WzMYPA27dAQGT/W2ghbXpJ8DyPAg8ezCxFVlZ1GhBGcKF\nz1xxE+EquZ1beVurcP4LGGVq5knDBIFdRYFjzZxKtJJXj/cBKvwckeJuRioL/0ns\nZA==\n-----END CERTIFICATE-----"

  # Certificate names for different environments
  cert_names = {
    prod    = "${local.name_prefix}-prod-cert"
    staging = "${local.name_prefix}-staging-cert"
    dev     = "${local.name_prefix}-dev-cert"
  }

  # Hostnames for different applications
  app_hostnames = toset([
    "app1.cf-tf-test.com",
    "app2.cf-tf-test.com",
    "api.cf-tf-test.com"
  ])
}

##########################
# 1. BASIC RESOURCES
##########################

# Basic certificate with account_id and all fields
resource "cloudflare_zero_trust_access_mtls_certificate" "basic_account" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-basic-account"
  certificate = local.test_cert
  associated_hostnames = [
    "basic.cf-tf-test.com",
    "www.basic.cf-tf-test.com"
  ]
}

# Basic certificate with zone_id
resource "cloudflare_zero_trust_access_mtls_certificate" "basic_zone" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-basic-zone"
  certificate = local.test_cert
  associated_hostnames = [
    "zone.cf-tf-test.com"
  ]
}

# Minimal certificate - only required fields
resource "cloudflare_zero_trust_access_mtls_certificate" "minimal" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-minimal"
  certificate = local.test_cert
}

# Certificate with empty associated_hostnames
resource "cloudflare_zero_trust_access_mtls_certificate" "empty_hostnames" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-empty-hostnames"
  certificate          = local.test_cert
  associated_hostnames = []
}

# Certificate with single hostname
resource "cloudflare_zero_trust_access_mtls_certificate" "single_hostname" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-single"
  certificate          = local.test_cert
  associated_hostnames = ["single.cf-tf-test.com"]
}

# Certificate with many hostnames
resource "cloudflare_zero_trust_access_mtls_certificate" "many_hostnames" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-many-hostnames"
  certificate = local.test_cert
  associated_hostnames = [
    "host1.cf-tf-test.com",
    "host2.cf-tf-test.com",
    "host3.cf-tf-test.com",
    "host4.cf-tf-test.com",
    "host5.cf-tf-test.com"
  ]
}

##########################
# 2. FOR_EACH PATTERNS
##########################

# for_each with map - different environments
resource "cloudflare_zero_trust_access_mtls_certificate" "environments" {
  for_each = local.cert_names

  account_id           = var.cloudflare_account_id
  name                 = each.value
  certificate          = local.test_cert
  associated_hostnames = ["${each.key}.cf-tf-test.com"]
}

# for_each with set - application hostnames
resource "cloudflare_zero_trust_access_mtls_certificate" "apps" {
  for_each = local.app_hostnames

  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-${replace(each.value, ".", "-")}"
  certificate          = local.test_cert
  associated_hostnames = [each.value]
}

# for_each with explicit map
resource "cloudflare_zero_trust_access_mtls_certificate" "regions" {
  for_each = {
    us-east = "us-east-1"
    us-west = "us-west-2"
    eu      = "eu-central-1"
  }

  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-region-${each.key}"
  certificate          = local.test_cert
  associated_hostnames = ["${each.value}.cf-tf-test.com"]
}

##########################
# 3. COUNT PATTERN
##########################

# count-based resources
resource "cloudflare_zero_trust_access_mtls_certificate" "counted" {
  count = 3

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-counted-${count.index}"
  certificate = local.test_cert
  associated_hostnames = [
    "counted-${count.index}.cf-tf-test.com"
  ]
}

##########################
# 4. CONDITIONAL RESOURCES
##########################

variable "enable_backup_cert" {
  type    = bool
  default = true
}

# Conditional resource using count
resource "cloudflare_zero_trust_access_mtls_certificate" "conditional" {
  count = var.enable_backup_cert ? 1 : 0

  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-conditional"
  certificate          = local.test_cert
  associated_hostnames = ["conditional.cf-tf-test.com"]
}

##########################
# 5. VARIABLE REFERENCES
##########################

# Using variables for all values
resource "cloudflare_zero_trust_access_mtls_certificate" "variable_ref" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-variable-ref"
  certificate          = local.test_cert
  associated_hostnames = ["${local.name_prefix}.cf-tf-test.com"]
}

##########################
# 6. TERRAFORM FUNCTIONS
##########################

# Using join() function
resource "cloudflare_zero_trust_access_mtls_certificate" "with_join" {
  account_id  = var.cloudflare_account_id
  name        = join("-", [local.name_prefix, "joined", "cert"])
  certificate = local.test_cert
  associated_hostnames = [
    join(".", ["subdomain", "cf-tf-test", "com"])
  ]
}

# Using format() function
resource "cloudflare_zero_trust_access_mtls_certificate" "with_format" {
  account_id           = var.cloudflare_account_id
  name                 = format("%s-formatted-%02d", local.name_prefix, 1)
  certificate          = local.test_cert
  associated_hostnames = [format("app%02d.cf-tf-test.com", 1)]
}

##########################
# 7. CROSS-RESOURCE REFERENCES
##########################

# These would reference other resources in real scenarios
# For integration tests, we use hardcoded values
resource "cloudflare_zero_trust_access_mtls_certificate" "app_cert" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-app-cert"
  certificate          = local.test_cert
  associated_hostnames = ["app.cf-tf-test.com", "api.cf-tf-test.com"]
}

resource "cloudflare_zero_trust_access_mtls_certificate" "backend_cert" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-backend-cert"
  certificate = local.test_cert
  # Reference the same hostnames as app_cert would in real scenario
  associated_hostnames = ["backend.cf-tf-test.com"]
}

##########################
# 8. EDGE CASES
##########################

# Very long certificate name
resource "cloudflare_zero_trust_access_mtls_certificate" "long_name" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-very-long-descriptive-certificate-name-for-testing-limits"
  certificate = local.test_cert
}

# Special characters in name
resource "cloudflare_zero_trust_access_mtls_certificate" "special_chars" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-cert_with-special.chars"
  certificate          = local.test_cert
  associated_hostnames = ["special-chars.cf-tf-test.com"]
}

# Using zone_id for variety
resource "cloudflare_zero_trust_access_mtls_certificate" "zone_scoped" {
  zone_id              = var.cloudflare_zone_id
  name                 = "${local.name_prefix}-zone-scoped"
  certificate          = local.test_cert
  associated_hostnames = ["zone-scoped.cf-tf-test.com"]
}

##########################
# 9. LIFECYCLE META-ARGUMENTS
##########################

# Certificate with create_before_destroy
resource "cloudflare_zero_trust_access_mtls_certificate" "with_lifecycle" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-lifecycle"
  certificate = local.test_cert

  lifecycle {
    create_before_destroy = true
  }
}

# Certificate with ignore_changes
resource "cloudflare_zero_trust_access_mtls_certificate" "ignore_changes" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-ignore-changes"
  certificate          = local.test_cert
  associated_hostnames = ["ignore.cf-tf-test.com"]

  lifecycle {
    ignore_changes = [associated_hostnames]
  }
}

##########################
# 10. DEPENDS_ON
##########################

# Certificate with explicit dependency
resource "cloudflare_zero_trust_access_mtls_certificate" "dependent" {
  account_id           = var.cloudflare_account_id
  name                 = "${local.name_prefix}-dependent"
  certificate          = local.test_cert
  associated_hostnames = ["dependent.cf-tf-test.com"]

  depends_on = [
    cloudflare_zero_trust_access_mtls_certificate.basic_account
  ]
}

# Summary: This file contains 30+ resource instances covering:
# - Basic configurations (6 resources)
# - for_each patterns with maps and sets (9 resources)
# - count patterns (3 resources)
# - Conditional resources (1 resource)
# - Variable references (1 resource)
# - Terraform functions (2 resources)
# - Cross-resource references (2 resources)
# - Edge cases (3 resources)
# - Lifecycle meta-arguments (2 resources)
# - Dependencies (1 resource)
# Total: 30 resource instances
