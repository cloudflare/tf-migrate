# Zone settings example - no transformation needed
resource "cloudflare_zone_settings_override" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"

  settings {
    always_use_https         = "on"
    automatic_https_rewrites = "on"
    brotli                   = "on"
    browser_cache_ttl        = 14400
    cache_level              = "aggressive"
    early_hints              = "on"
    email_obfuscation        = "on"
    hotlink_protection       = "off"
    http2                    = "on"
    http3                    = "on"
    ip_geolocation           = "on"
    ipv6                     = "on"
    min_tls_version          = "1.2"
    opportunistic_encryption = "on"
    opportunistic_onion      = "on"
    polish                   = "lossless"
    ssl                      = "strict"
    tls_1_3                  = "on"
    websockets               = "on"
    zero_rtt                 = "on"
  }
}
