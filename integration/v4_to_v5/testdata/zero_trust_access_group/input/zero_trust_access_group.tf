# Zero Trust Access Group with complex rules
resource "cloudflare_access_group" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Example Group"

  include {
    email        = ["user1@example.com", "user2@example.com"]
    email_domain = ["example.com"]
    everyone     = true
    geo          = ["US", "CA"]
  }

  exclude {
    ip            = ["192.0.2.1"]
    email         = ["excluded@example.com"]
    certificate   = true
  }

  require {
    group = ["group-id-1", "group-id-2"]
  }
}
