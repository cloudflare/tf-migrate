resource "cloudflare_zone" "test" {
  paused = false
  type   = "full"
  name   = "test.example.com"
  account = {
    id = "test123"
  }
}
