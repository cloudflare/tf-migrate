# Cross-file references to cloudflare_zone.*.zone
# These should be rewritten to .name after migration

# Pattern: Direct .zone reference in another resource
resource "cloudflare_record" "cftftest_zone_ref" {
  zone_id = cloudflare_zone.minimal.id
  name    = cloudflare_zone.minimal.zone
  type    = "CNAME"
  content = "example.com"
}

# Pattern: .zone reference in string interpolation
resource "cloudflare_record" "cftftest_zone_interpolation" {
  zone_id = cloudflare_zone.minimal.id
  name    = "cftftest-sub.${cloudflare_zone.minimal.zone}"
  type    = "CNAME"
  content = "example.com"
}

# Pattern: .zone reference in a local
locals {
  cftftest_zone_domain = cloudflare_zone.minimal.zone
}
