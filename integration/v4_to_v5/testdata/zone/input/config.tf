resource "cloudflare_zone" "example" {
  zone       = "example.com"
  account_id = "f037e56e89293a057740de681ac9abbe"
  type       = "full"
}

resource "cloudflare_zone" "with_removed_attrs" {
  zone       = "test.example.com"
  account_id = "f037e56e89293a057740de681ac9abbe"
  jump_start = true
  plan       = "pro"
  paused     = true
}
