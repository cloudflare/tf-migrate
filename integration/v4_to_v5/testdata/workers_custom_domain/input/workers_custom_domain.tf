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
  workers_service_name = "edge-worker"
  workers_hostname     = "api.${var.cloudflare_domain}"
}

resource "cloudflare_worker_domain" "example" {
  account_id  = var.cloudflare_account_id
  hostname    = local.workers_hostname
  service     = local.workers_service_name
  zone_id     = var.cloudflare_zone_id
  environment = "production"
}
