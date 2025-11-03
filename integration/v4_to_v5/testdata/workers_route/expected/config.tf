resource "cloudflare_workers_route" "api" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "api.example.com/*"
  script  = "api-worker"
}

resource "cloudflare_workers_route" "web" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "www.example.com/*"
  script  = "web-worker"
}

resource "cloudflare_workers_route" "no_script" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "static.example.com/*"
}
