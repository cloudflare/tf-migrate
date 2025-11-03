resource "cloudflare_worker_domain" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "subdomain.example.com"
  service    = "my-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}

resource "cloudflare_worker_domain" "with_environment" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  hostname    = "app.example.com"
  service     = "app-service"
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  environment = "production"
}

resource "cloudflare_workers_custom_domain" "already_correct" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  hostname   = "correct.example.com"
  service    = "correct-service"
  zone_id    = "0da42c8d2132a9ddaf714f9e7c920711"
}
