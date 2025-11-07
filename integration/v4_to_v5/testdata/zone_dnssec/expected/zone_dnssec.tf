# Zone DNSSEC with modified_on field (should be removed)
# Status should be added from state value (status = "active" in state)
resource "cloudflare_zone_dnssec" "example_active" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  status  = "active"
}

# Zone DNSSEC with minimal fields
# Status should be added from state value (status = "disabled" in state)
resource "cloudflare_zone_dnssec" "example_disabled" {
  zone_id = "1ea42c8d2132a9ddaf714f9e7c920722"
  status  = "disabled"
}

# Zone DNSSEC with only zone_id
# Status should be added from state value (status = "active" in state)
resource "cloudflare_zone_dnssec" "example_minimal" {
  zone_id = "2fa42c8d2132a9ddaf714f9e7c920733"
  status  = "active"
}

# Zone DNSSEC with null status in state
# Status should NOT be added when null in state
resource "cloudflare_zone_dnssec" "example_null_status" {
  zone_id = "3fb42c8d2132a9ddaf714f9e7c920744"
}
