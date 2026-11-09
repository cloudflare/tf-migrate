output "account_permission" {
  value = data.cloudflare_api_token_permission_groups_list.all.account["Account Settings Read"]
}
