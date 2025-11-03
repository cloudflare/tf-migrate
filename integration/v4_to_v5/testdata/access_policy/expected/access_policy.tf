# Access policy with complex include/exclude rules
resource "cloudflare_zero_trust_access_policy" "example" {
  application_id = "app-id-123"
  zone_id        = "0da42c8d2132a9ddaf714f9e7c920711"
  name           = "Example Policy"
  precedence     = 1
  decision       = "allow"

  include {
    email        = ["user1@example.com", "user2@example.com"]
    email_domain = ["example.com"]
    everyone     = true
  }

  exclude {
    ip = ["192.0.2.1/32", "198.51.100.0/24"]
  }
}

# Access policy with array expansion
resource "cloudflare_zero_trust_access_policy" "array_expansion" {
  application_id = "app-id-456"
  account_id     = "f037e56e89293a057740de681ac9abbe"
  name           = "Array Expansion Policy"
  precedence     = 2
  decision       = "deny"

  include {
    geo   = ["US", "CA"]
    group = ["group-id-1", "group-id-2"]
  }
}
