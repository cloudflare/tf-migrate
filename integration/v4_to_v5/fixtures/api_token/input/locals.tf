locals {
  names = ["DNS Write"]
  dns_id = data.cloudflare_api_token_permission_groups.all.zone["DNS Write"]
}
