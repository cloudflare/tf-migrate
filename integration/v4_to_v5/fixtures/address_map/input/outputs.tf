output "ips" {
  value = [for entry in cloudflare_address_map.cftftest_empty[0].ips : entry.ip]
}

output "membership_can_delete" {
  value = [for entry in cloudflare_address_map.cftftest_empty[0].memberships : entry.can_delete]
}
