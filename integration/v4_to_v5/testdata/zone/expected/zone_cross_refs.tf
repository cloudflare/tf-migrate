# Cross-file references to cloudflare_zone.*.zone
# These should be rewritten to .name after migration



# Pattern: .zone reference in a local
locals {
  cftftest_zone_domain = cloudflare_zone.minimal.name
}

# Pattern: Direct .zone reference in another resource
resource "cloudflare_dns_record" "cftftest_zone_ref" {
  zone_id = cloudflare_zone.minimal.id
  name    = cloudflare_zone.minimal.name
  type    = "CNAME"
  content = "example.com"
  ttl     = 1
}

moved {
  from = cloudflare_record.cftftest_zone_ref
  to   = cloudflare_dns_record.cftftest_zone_ref
}

# Pattern: .zone reference in string interpolation
resource "cloudflare_dns_record" "cftftest_zone_interpolation" {
  zone_id = cloudflare_zone.minimal.id
  name    = "cftftest-sub.${cloudflare_zone.minimal.name}"
  type    = "CNAME"
  content = "example.com"
  ttl     = 1
}

moved {
  from = cloudflare_record.cftftest_zone_interpolation
  to   = cloudflare_dns_record.cftftest_zone_interpolation
}
