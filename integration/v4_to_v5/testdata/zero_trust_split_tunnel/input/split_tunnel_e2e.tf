# E2E test for cloudflare_split_tunnel → device profile merge + removed {} blocks.
# Regression test for https://github.com/cloudflare/tf-migrate/issues/289
#
# cloudflare_split_tunnel is removed in v5. The migrator:
#   1. Merges static tunnels into the device profile's exclude/include attribute
#   2. Generates removed {} blocks for state cleanup
#   3. Removes the original split_tunnel resource blocks
#
# The e2e runner strips split_tunnel entries from state via removeObsoleteStateEntries
# (the v5 provider has no schema for this type). The removed {} blocks handle real-world
# state cleanup via `terraform apply`.
#
# Note: v5 enforces that include and exclude are mutually exclusive on a device profile,
# so this test only uses exclude mode.

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID (not used by this account-scoped resource)"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain (not used by this account-scoped resource)"
  type        = string
}

# Default profile — split tunnels without policy_id target this
resource "cloudflare_zero_trust_device_profiles" "cftftest_split_tunnel_default" {
  account_id  = var.cloudflare_account_id
  name        = "Default Profile"
  description = "Default profile for split tunnel E2E test"
  default     = true

  allow_mode_switch = true
  auto_connect      = 0
  captive_portal    = 180
}

# First exclude split tunnel — multiple tunnels blocks
resource "cloudflare_split_tunnel" "cftftest_default_exclude" {
  account_id = var.cloudflare_account_id
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
  }

  tunnels {
    address     = "10.0.0.0/8"
    description = "Internal RFC1918"
  }
}

# Second exclude split tunnel — tests merging multiple split_tunnel resources
resource "cloudflare_split_tunnel" "cftftest_default_exclude_2" {
  account_id = var.cloudflare_account_id
  mode       = "exclude"

  tunnels {
    address     = "172.16.0.0/12"
    description = "Private range"
  }
}
