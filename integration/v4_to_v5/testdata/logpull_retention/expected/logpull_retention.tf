# Test Case 1: Basic logpull retention enabled
resource "cloudflare_logpull_retention" "enabled_zone" {
  zone_id = "d56084adb405e0b7e32c52321bf07be6"
  flag    = true
}

# Test Case 2: Logpull retention disabled
resource "cloudflare_logpull_retention" "disabled_zone" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
  flag    = false
}
