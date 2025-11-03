resource "cloudflare_zone_setting" "example_always_use_https" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "always_use_https"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_automatic_https_rewrites" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "automatic_https_rewrites"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_brotli" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "brotli"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_browser_cache_ttl" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "browser_cache_ttl"
  value      = 14400
}
resource "cloudflare_zone_setting" "example_cache_level" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "cache_level"
  value      = "aggressive"
}
resource "cloudflare_zone_setting" "example_early_hints" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "early_hints"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_email_obfuscation" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "email_obfuscation"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_hotlink_protection" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "hotlink_protection"
  value      = "off"
}
resource "cloudflare_zone_setting" "example_http2" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "http2"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_http3" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "http3"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_ip_geolocation" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "ip_geolocation"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_ipv6" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "ipv6"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_min_tls_version" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "min_tls_version"
  value      = "1.2"
}
resource "cloudflare_zone_setting" "example_opportunistic_encryption" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "opportunistic_encryption"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_opportunistic_onion" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "opportunistic_onion"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_polish" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "polish"
  value      = "lossless"
}
resource "cloudflare_zone_setting" "example_ssl" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "ssl"
  value      = "strict"
}
resource "cloudflare_zone_setting" "example_tls_1_3" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "tls_1_3"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_websockets" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "websockets"
  value      = "on"
}
resource "cloudflare_zone_setting" "example_zero_rtt" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  setting_id = "0rtt"
  value      = "on"
}
