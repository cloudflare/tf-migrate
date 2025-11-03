resource "cloudflare_regional_hostname" "with_timeouts" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional.example.com"
  region_key = "us"

}

resource "cloudflare_regional_hostname" "without_timeouts" {
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname   = "regional2.example.com"
  region_key = "eu"
}

resource "cloudflare_regional_hostname" "with_var" {
  zone_id    = var.zone_id
  hostname   = "regional3.example.com"
  region_key = "ca"

}
