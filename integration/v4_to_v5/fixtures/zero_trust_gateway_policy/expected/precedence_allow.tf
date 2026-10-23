resource "cloudflare_zero_trust_gateway_policy" "cftftest_harbor_staging" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest Harbor Staging"
  action      = "allow"
  filters     = ["l4"]
  traffic     = "net.dst.ip in {192.0.2.2}"
  precedence  = 10116
}

# Duplicate v4 configuration values must not be automatically renumbered.
resource "cloudflare_zero_trust_gateway_policy" "cftftest_gitlab_staging" {
  account_id  = var.cloudflare_account_id
  name        = "cftftest GitLab Staging"
  action      = "allow"
  filters     = ["l4"]
  traffic     = "net.dst.ip in {192.0.2.1}"
  precedence  = 10116
}

moved {
  from = cloudflare_teams_rule.cftftest_gitlab_staging
  to   = cloudflare_zero_trust_gateway_policy.cftftest_gitlab_staging
}
