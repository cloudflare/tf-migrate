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
  name_prefix          = "v5-upgrade-${replace(var.from_version, ".", "-")}"
  workers_service_name = "${local.name_prefix}-workers-custom-domain-${substr(var.cloudflare_account_id, 0, 8)}"
  workers_hostname     = "${local.name_prefix}-workers-custom-domain-${substr(var.cloudflare_account_id, 0, 8)}.${var.cloudflare_domain}"
  workers_content      = "addEventListener('fetch', event => { event.respondWith(new Response('ok')); });"
}

# Prerequisite Worker service for domain attachment.
resource "cloudflare_workers_script" "service" {
  account_id  = var.cloudflare_account_id
  content     = local.workers_content
  script_name = local.workers_service_name
}


resource "cloudflare_workers_custom_domain" "example" {
  account_id  = var.cloudflare_account_id
  hostname    = local.workers_hostname
  service     = local.workers_service_name
  zone_id     = var.cloudflare_zone_id
  environment = "production"

  depends_on = [cloudflare_workers_script.service]
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
