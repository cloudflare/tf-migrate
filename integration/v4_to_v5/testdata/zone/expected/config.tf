resource "cloudflare_zone" "example" {
  type = "full"
  name = "example.com"
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
}

resource "cloudflare_zone" "with_removed_attrs" {
  paused = true
  name   = "test.example.com"
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
}
