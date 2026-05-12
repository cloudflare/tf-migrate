# Cross-file references to cloudflare_zone.*.zone
# These should be rewritten to .name after migration

# Pattern: Direct .zone reference in another resource
resource "cloudflare_record" "cftftest_zone_xref" {
  zone_id = cloudflare_zone.minimal.id
  name    = "cftftest-xref.${cloudflare_zone.minimal.zone}"
  type    = "TXT"
  content = cloudflare_zone.minimal.zone
}

# Pattern: .zone reference in string interpolation
resource "cloudflare_record" "cftftest_zone_xref_interp" {
  zone_id = cloudflare_zone.minimal.id
  name    = "cftftest-xref-interp.${cloudflare_zone.minimal.zone}"
  type    = "TXT"
  content = "zone=${cloudflare_zone.minimal.zone}"
}

# Pattern: .zone reference in a local
locals {
  cftftest_zone_domain = cloudflare_zone.minimal.zone
}
