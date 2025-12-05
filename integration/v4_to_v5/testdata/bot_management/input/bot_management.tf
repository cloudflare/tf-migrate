variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Comprehensive bot management test with available fields
# Note: bot_management is a singleton per zone - only one config per zone
resource "cloudflare_bot_management" "test" {
  zone_id                         = var.cloudflare_zone_id
  enable_js                       = true
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = false
  suppress_session_score          = false
  auto_update_model               = true
  ai_bots_protection              = "disabled"
}
