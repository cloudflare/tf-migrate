locals {
  name_prefix = "cftftest"
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test variables for variable/local reference preservation tests
# For E2E: These will use values from the first zone lookup to ensure they work
variable "account_id" {
  type    = string
  default = ""
}

variable "zone_name" {
  type    = string
  default = ""
}

locals {
  # For E2E: Use empty string, will be overridden by direct reference to by_id
  zone_domain   = ""
  cf_account_id = ""
}

# Zone datasource v4 test fixtures
# Testing all lookup scenarios

# Scenario 1: zone_id lookup (no changes)
data "cloudflare_zone" "by_id" {
  zone_id = var.cloudflare_zone_id
}

# Scenario 2: name only lookup
# Uses the zone name from by_id lookup so it works with any zone
data "cloudflare_zone" "by_name" {
  filter = {
    name = data.cloudflare_zone.by_id.name
  }
}

# Scenario 3: account_id + name lookup
# Note: v4 provider requires name or zone_id, so we include name for E2E testing
data "cloudflare_zone" "by_account" {
  filter = {
    name = data.cloudflare_zone.by_id.name
    account = {
      id = var.cloudflare_account_id
    }
  }
}

# Scenario 4: name + account_id lookup (same as scenario 3 but different instance)
data "cloudflare_zone" "by_name_and_account" {
  filter = {
    name = data.cloudflare_zone.by_id.name
    account = {
      id = var.cloudflare_account_id
    }
  }
}

# Scenario 5: with variable references (tests that var refs are preserved)
# For E2E: Uses the zone name from by_id so it actually works
data "cloudflare_zone" "with_variables" {
  filter = {
    name = coalesce(var.zone_name, data.cloudflare_zone.by_id.name)
    account = {
      id = coalesce(var.account_id, var.cloudflare_account_id)
    }
  }
}

# Scenario 6: another zone_id lookup (tests multiple zone_id lookups)
# For E2E: Uses the same zone so it actually exists
data "cloudflare_zone" "another_by_id" {
  zone_id = var.cloudflare_zone_id
}

# Scenario 7: name with local reference (tests that local refs are preserved)
# For E2E: Uses the zone name from by_id so it actually works
data "cloudflare_zone" "from_local" {
  filter = {
    name = coalesce(local.zone_domain, data.cloudflare_zone.by_id.name)
    account = {
      id = coalesce(local.cf_account_id, var.cloudflare_account_id)
    }
  }
}
