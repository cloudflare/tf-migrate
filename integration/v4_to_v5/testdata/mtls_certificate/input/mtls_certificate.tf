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

locals {
  name_prefix = "cftftest"
  # Test CA certificates
  ca_certificates = {
    root_ca = {
      name        = "${local.name_prefix}-root-ca"
      certificate = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
    }
    intermediate_ca = {
      name        = "${local.name_prefix}-intermediate-ca"
      certificate = "-----BEGIN CERTIFICATE-----\nMIICUTCCAbqgAwIBAgIRAMDZ7VqH7wZPPJQjCfJCLbgwDQYJKoZIhvcNAQELBQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAw8V7VxRPfK7KQvvBJrKL5nYLJGxTv7LPQJ3aKQvL0WrL4xL3xvxL4xvLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLAgMBAAGjgYAwfjAMBgNVHRMBAf8EAjAAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAPBgNVHQ8BAf8EBQMDB6AAMB0GA1UdDgQWBBRfQ7G4G0pQ0L0L0L0L0L0L0L0LMB8GA1UdIwQYMBaAFGxQ0L0L0L0L0L0L0L0L0L0L0L0LMA0GCSqGSIb3DQEBCwUAA4GBAGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END CERTIFICATE-----"
    }
  }

  # Test leaf certificates
  leaf_certificates = {
    webserver = {
      name        = "${local.name_prefix}-webserver"
      certificate = "-----BEGIN CERTIFICATE-----\nMIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAMPFe1cUT3yuykL7wSayi+Z2CyRsU7+yz0Cd2ikLy9Fqy+MS98b8S+Mby8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8YCAwEAAQKBgGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END CERTIFICATE-----"
      private_key = "-----BEGIN PRIVATE KEY-----\nMIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAMPFe1cUT3yuykL7wSayi+Z2CyRsU7+yz0Cd2ikLy9Fqy+MS98b8S+Mby8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8YCAwEAAQKBgGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END PRIVATE KEY-----"
    }
    api_server = {
      name        = "${local.name_prefix}-api-server"
      certificate = "-----BEGIN CERTIFICATE-----\nMIICUTCCAbqgAwIBAgIRAMDZ7VqH7wZPPJQjCfJCLbhwDQYJKoZIhvcNAQELBQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAw8V7VxRPfK7KQvvBJrKL5nYLJGxTv7LPQJ3aKQvL0WrL4xL3xvxL4xvLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLAgMBAAGjgYAwfjAMBgNVHRMBAf8EAjAAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAPBgNVHQ8BAf8EBQMDB6AAMB0GA1UdDgQWBBRfQ7G4G0pQ0L0L0L0L0L0L0L0LMB8GA1UdIwQYMBaAFGxQ0L0L0L0L0L0L0L0L0L0L0L0LMA0GCSqGSIb3DQEBCwUAA4GBAGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END CERTIFICATE-----"
      private_key = "-----BEGIN PRIVATE KEY-----\nMIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAMPFe1cUT3yuykL7wSayi+Z2CyRsU7+yz0Cd2ikLy9Fqy+MS98b8S+Mby8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8YCAwEAAQKBgGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END PRIVATE KEY-----"
    }
  }
}

# Pattern 1: Basic CA certificates with explicit fields
resource "cloudflare_mtls_certificate" "basic_ca" {
  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
  name         = "${local.name_prefix}-basic-ca"
}

# Pattern 2: Minimal CA certificate without name
resource "cloudflare_mtls_certificate" "minimal_ca" {
  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
}

# Pattern 3: Leaf certificate with private key
resource "cloudflare_mtls_certificate" "basic_leaf" {
  account_id   = var.cloudflare_account_id
  ca           = false
  certificates = "-----BEGIN CERTIFICATE-----\nMIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAMPFe1cUT3yuykL7wSayi+Z2CyRsU7+yz0Cd2ikLy9Fqy+MS98b8S+Mby8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8YCAwEAAQKBgGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END CERTIFICATE-----"
  private_key  = "-----BEGIN PRIVATE KEY-----\nMIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAMPFe1cUT3yuykL7wSayi+Z2CyRsU7+yz0Cd2ikLy9Fqy+MS98b8S+Mby8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8b8S8vG/EvLxvxLy8YCAwEAAQKBgGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END PRIVATE KEY-----"
  name         = "${local.name_prefix}-basic-leaf"
}

# Pattern 4: Leaf certificate without private key
resource "cloudflare_mtls_certificate" "leaf_no_key" {
  account_id   = var.cloudflare_account_id
  ca           = false
  certificates = "-----BEGIN CERTIFICATE-----\nMIICUTCCAbqgAwIBAgIRAMDZ7VqH7wZPPJQjCfJCLbhwDQYJKoZIhvcNAQELBQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAw8V7VxRPfK7KQvvBJrKL5nYLJGxTv7LPQJ3aKQvL0WrL4xL3xvxL4xvLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLxvxLAgMBAAGjgYAwfjAMBgNVHRMBAf8EAjAAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAPBgNVHQ8BAf8EBQMDB6AAMB0GA1UdDgQWBBRfQ7G4G0pQ0L0L0L0L0L0L0L0LMB8GA1UdIwQYMBaAFGxQ0L0L0L0L0L0L0L0L0L0L0L0LMA0GCSqGSIb3DQEBCwUAA4GBAGxQ0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L0L\n-----END CERTIFICATE-----"
  name         = "${local.name_prefix}-leaf-no-key"
}

# Pattern 5: for_each with map of CA certificates
resource "cloudflare_mtls_certificate" "ca_set" {
  for_each = local.ca_certificates

  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = each.value.certificate
  name         = each.value.name
}

# Pattern 6: for_each with map of leaf certificates (with private keys)
resource "cloudflare_mtls_certificate" "leaf_set" {
  for_each = local.leaf_certificates

  account_id   = var.cloudflare_account_id
  ca           = false
  certificates = each.value.certificate
  private_key  = each.value.private_key
  name         = each.value.name
}

# Pattern 7: count-based creation
resource "cloudflare_mtls_certificate" "counted" {
  count = 3

  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
  name         = "${local.name_prefix}-counted-${count.index}"
}

# Pattern 8: for_each with set (using toset)
resource "cloudflare_mtls_certificate" "from_set" {
  for_each = toset(["alpha", "beta", "gamma"])

  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
  name         = "${local.name_prefix}-set-${each.value}"
}

# Pattern 9: Conditional creation
resource "cloudflare_mtls_certificate" "conditional" {
  count = var.cloudflare_account_id != "" ? 1 : 0

  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
  name         = "${local.name_prefix}-conditional"
}

# Pattern 10: Using locals for certificate content
locals {
  reused_certificate = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
}

resource "cloudflare_mtls_certificate" "with_local" {
  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = local.reused_certificate
  name         = "${local.name_prefix}-with-local"
}

# Pattern 11: String interpolation in name
resource "cloudflare_mtls_certificate" "interpolated" {
  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJALoM7P+9zyPoMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjASMRAwDgYDVQQDDAdUZXN0IENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDM8I8OwONnNJerGRUjV8zH5lPq1nKbwH8ueP4fxj7jxVXxfxFNTEy3fW8uKJGPzHYxQm3QxVqXfVq1xNEP8zQxNk3fXfWl3fVWxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVxfVwIDAQABMA0GCSqGSIb3DQEBCwUAA4GBAExaMZ8EQeYF9lKV5VZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlVZlV\n-----END CERTIFICATE-----"
  name         = "${local.name_prefix}-${var.cloudflare_account_id}-cert"
}

# Summary:
# - 11 distinct resource patterns
# - Total instances: 4 single + 2 for_each (4 instances) + 1 count (3 instances) + 1 for_each (3 instances) + 1 conditional + 1 local + 1 interpolated = ~18 instances
# - Patterns covered: basic, minimal, for_each with maps/sets, count, conditional, locals, interpolation
# - Certificate types: CA and leaf certificates
# - Optional fields: with/without name, with/without private_key
