resource "cloudflare_worker_route" "api" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "api.example.com/*"
  script_name = "api-worker"
}

resource "cloudflare_workers_route" "web" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern     = "www.example.com/*"
  script_name = "web-worker"
}

resource "cloudflare_worker_route" "no_script" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  pattern = "static.example.com/*"
}
