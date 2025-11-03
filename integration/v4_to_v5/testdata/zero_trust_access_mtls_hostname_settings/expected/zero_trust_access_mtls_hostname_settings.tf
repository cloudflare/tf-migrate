# Zero Trust Access mTLS Hostname Settings with dynamic blocks
resource "cloudflare_access_mtls_hostname_settings" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  dynamic "settings" {
    for_each = var.hostnames
    content {
      hostname                      = settings.value.hostname
      client_certificate_forwarding = settings.value.client_certificate_forwarding
    }
  }
}

# Static settings blocks
resource "cloudflare_access_mtls_hostname_settings" "static" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    hostname                      = "example.com"
    client_certificate_forwarding = true
  }

  settings {
    hostname                      = "api.example.com"
    client_certificate_forwarding = false
  }
}
