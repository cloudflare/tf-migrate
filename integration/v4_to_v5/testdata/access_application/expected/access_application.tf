# Access application with policies list of IDs
resource "cloudflare_zero_trust_access_application" "example" {
  zone_id          = "0da42c8d2132a9ddaf714f9e7c920711"
  name             = "Example Application"
  domain           = "example.com"
  session_duration = "24h"
  policies         = [{ id = "policy-id-1" }, { id = "policy-id-2" }]
  type             = "self_hosted"
}

# Access application with single policy
resource "cloudflare_zero_trust_access_application" "single_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Single Policy App"
  domain     = "single.example.com"
  policies   = [{ id = "policy-id-3" }]
  type       = "self_hosted"
}

# Access application without policies
resource "cloudflare_zero_trust_access_application" "no_policies" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "No Policies App"
  domain  = "nopolicies.example.com"
  type    = "self_hosted"
}
