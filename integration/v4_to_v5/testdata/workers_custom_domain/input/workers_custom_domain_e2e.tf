variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

locals {
  workers_service_name = "cftftest-workers-custom-domain-${substr(var.cloudflare_account_id, 0, 8)}"
  workers_hostname     = "workers-custom-domain-${substr(var.cloudflare_account_id, 0, 8)}.${var.cloudflare_domain}"
  workers_content      = "addEventListener('fetch', event => { event.respondWith(new Response('ok')); });"
}

# Prerequisite Worker service for domain attachment.
resource "cloudflare_workers_script" "service" {
  account_id = var.cloudflare_account_id
  name       = local.workers_service_name
  content    = local.workers_content
}

resource "cloudflare_worker_domain" "example" {
  account_id  = var.cloudflare_account_id
  hostname    = local.workers_hostname
  service     = local.workers_service_name
  zone_id     = var.cloudflare_zone_id
  environment = "production"

  depends_on = [cloudflare_workers_script.service]
}
