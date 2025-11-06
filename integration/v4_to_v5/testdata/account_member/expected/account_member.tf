# Test Case 1: Basic account member
resource "cloudflare_account_member" "basic_am" {
  account_id = "023e105f4ecef8ad9ca31a8372d0c353"
  email      = "user@example.com"
  roles      = ["3536bcfad5faccb999b47003c79917fb"]
}

# Test Case 2: Full account member
resource "cloudflare_account_member" "full_am" {
  account_id = "023e105f4ecef8ad9ca31a8372d0c353"
  status     = "accepted"
  email      = "user@example.com"
  roles = [
    "3536bcfad5faccb999b47003c79917fb",
    "68b329da9893e34099c7d8ad5cb9c940"
  ]
}
