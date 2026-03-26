variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_domain" {
  type = string
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

locals {
  # Use different IPs from v4→v5 tests to avoid "duplicate_of_existing" errors.
  # v4→v5 uses 198.51.100.x and 192.0.2.x; v5 upgrade uses 100.64.x.x (shared address space RFC 6598).
  minor = tonumber(split(".", var.from_version)[1])
}

resource "cloudflare_access_rule" "example_block" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "v5-upgrade block IP"
  configuration = {
    target = "ip"
    value  = "100.64.${local.minor}.10"
  }
}

resource "cloudflare_access_rule" "example_challenge" {
  account_id = var.cloudflare_account_id
  mode       = "challenge"
  notes      = "v5-upgrade challenge IP"
  configuration = {
    target = "ip"
    value  = "100.64.${local.minor}.11"
  }
}

resource "cloudflare_access_rule" "example_whitelist" {
  zone_id = var.cloudflare_zone_id
  mode    = "whitelist"
  notes   = "v5-upgrade allow IP"
  configuration = {
    target = "ip"
    value  = "100.64.${local.minor}.12"
  }
}

resource "cloudflare_access_rule" "example_js_challenge" {
  account_id = var.cloudflare_account_id
  mode       = "js_challenge"
  notes      = "v5-upgrade js_challenge IP"
  configuration = {
    target = "ip"
    value  = "100.64.${local.minor}.13"
  }
}

resource "cloudflare_access_rule" "target_ip_range" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "v5-upgrade block range"
  configuration = {
    target = "ip_range"
    value  = "100.64.${local.minor}.0/24"
  }
}

resource "cloudflare_access_rule" "target_ip6" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "v5-upgrade block IPv6"
  configuration = {
    target = "ip6"
    value  = "2001:0db8:00${local.minor}:0000:0000:0000:0000:0001"
  }
}

resource "cloudflare_access_rule" "with_lifecycle" {
  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "v5-upgrade lifecycle"
  configuration = {
    target = "ip"
    value  = "100.64.${local.minor}.99"
  }
  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_access_rule" "backup_blocks" {
  count = 2

  account_id = var.cloudflare_account_id
  mode       = "block"
  notes      = "v5-upgrade backup ${count.index}"
  configuration = {
    target = "ip"
    value  = "100.64.${local.minor}.${count.index + 20}"
  }
}
