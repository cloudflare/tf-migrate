variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Zone DNSSEC with modified_on field (should be removed)
# Status should be added from state value (status = "active" in state)
resource "cloudflare_zone_dnssec" "example_active" {
  zone_id     = var.cloudflare_zone_id
  modified_on = "2024-01-15T10:30:00Z"
}

# Zone DNSSEC with minimal fields
# Status should be added from state value (status = "disabled" in state)
resource "cloudflare_zone_dnssec" "example_disabled" {
  zone_id = var.cloudflare_zone_id
}

# Zone DNSSEC with only zone_id
# Status should be added from state value (status = "active" in state)
resource "cloudflare_zone_dnssec" "example_minimal" {
  zone_id = var.cloudflare_zone_id
}

# Zone DNSSEC with null status in state
# Status should NOT be added when null in state
resource "cloudflare_zone_dnssec" "example_null_status" {
  zone_id = var.cloudflare_zone_id
}
