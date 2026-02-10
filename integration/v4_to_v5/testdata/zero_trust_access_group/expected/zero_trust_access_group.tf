# Variables (no defaults for account_id and zone_id per testing guide)
variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

locals {
  name_prefix = "cftftest"
}





















# Pattern 1: Simple email selector
resource "cloudflare_zero_trust_access_group" "simple_email" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Simple Email Group"

  include = [
    {
      email = {
        email = "user1@example.com"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.simple_email
  to   = cloudflare_zero_trust_access_group.simple_email
}

# Pattern 2: Multiple selector types
resource "cloudflare_zero_trust_access_group" "multiple_selectors" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Multiple Selectors Group"

  include = [
    {
      email = {
        email = "admin@example.com"
      }
    },
    {
      email = {
        email = "manager@example.com"
      }
    },
    {
      ip = {
        ip = "192.168.1.0/24"
      }
    },
    {
      ip = {
        ip = "10.0.0.0/8"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.multiple_selectors
  to   = cloudflare_zero_trust_access_group.multiple_selectors
}

# Pattern 3: for_each with map
resource "cloudflare_zero_trust_access_group" "foreach_map" {
  for_each = {
    team1 = "Engineering Team"
    team2 = "Product Team"
    team3 = "Design Team"
  }

  account_id = var.cloudflare_account_id
  name       = format("%s %s", local.name_prefix, each.value)

  include = [
    {
      email_domain = {
        domain = "example.com"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.foreach_map
  to   = cloudflare_zero_trust_access_group.foreach_map
}

# Pattern 4: for_each with set
resource "cloudflare_zero_trust_access_group" "foreach_set" {
  for_each = toset(["contractors", "vendors", "partners"])

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix}-${each.value}-group"

  include = [
    {
      ip = {
        ip = "192.0.2.1/32"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.foreach_set
  to   = cloudflare_zero_trust_access_group.foreach_set
}

# Pattern 5: count-based resources
resource "cloudflare_zero_trust_access_group" "count_based" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Count Group ${count.index}"

  include = [
    {
      ip = {
        ip = "10.${count.index}.0.0/16"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.count_based
  to   = cloudflare_zero_trust_access_group.count_based
}

# Pattern 6: Conditional resource
resource "cloudflare_zero_trust_access_group" "conditional" {
  count = var.cloudflare_account_id != "" ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Conditional Group"

  include = [
    {
      everyone = {}
    },
  ]
}

moved {
  from = cloudflare_access_group.conditional
  to   = cloudflare_zero_trust_access_group.conditional
}

# Pattern 7: Boolean selectors
resource "cloudflare_zero_trust_access_group" "booleans" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Boolean Selectors Group"

  include = [
    {
      everyone = {}
    },
  ]

  exclude = [
    {
      certificate = {}
    },
  ]
}

moved {
  from = cloudflare_access_group.booleans
  to   = cloudflare_zero_trust_access_group.booleans
}

# Pattern 8: Email domain selector
resource "cloudflare_zero_trust_access_group" "email_domains" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Email Domain Group"

  include = [
    {
      email_domain = {
        domain = "company.com"
      }
    },
    {
      email_domain = {
        domain = "partner.com"
      }
    },
    {
      email_domain = {
        domain = "vendor.com"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.email_domains
  to   = cloudflare_zero_trust_access_group.email_domains
}

# Pattern 9: Geo selector
resource "cloudflare_zero_trust_access_group" "geo" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Geo Group"

  include = [
    {
      geo = {
        country_code = "US"
      }
    },
    {
      geo = {
        country_code = "CA"
      }
    },
    {
      geo = {
        country_code = "GB"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.geo
  to   = cloudflare_zero_trust_access_group.geo
}

# Pattern 10: All three rule types
resource "cloudflare_zero_trust_access_group" "all_rules" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} All Rules Group"

  include = [
    {
      email = {
        email = "user@example.com"
      }
    },
  ]

  exclude = [
    {
      ip = {
        ip = "192.168.0.0/16"
      }
    },
  ]

  require = [
    {
      email_domain = {
        domain = "example.com"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.all_rules
  to   = cloudflare_zero_trust_access_group.all_rules
}

# Pattern 11: Multiple selectors in each rule
resource "cloudflare_zero_trust_access_group" "complex" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Complex Group"

  include = [
    {
      email = {
        email = "admin@example.com"
      }
    },
    {
      email = {
        email = "manager@example.com"
      }
    },
    {
      ip = {
        ip = "10.0.0.0/8"
      }
    },
    {
      email_domain = {
        domain = "example.com"
      }
    },
  ]

  exclude = [
    {
      ip = {
        ip = "10.0.1.0/24"
      }
    },
    {
      geo = {
        country_code = "CN"
      }
    },
    {
      geo = {
        country_code = "RU"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.complex
  to   = cloudflare_zero_trust_access_group.complex
}

# Pattern 12: Service token selector
resource "cloudflare_zero_trust_access_group" "service_tokens" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Service Token Group"

  include = [
    {
      any_valid_service_token = {}
    },
  ]
}

moved {
  from = cloudflare_access_group.service_tokens
  to   = cloudflare_zero_trust_access_group.service_tokens
}

# Pattern 13: Group and email list selectors
resource "cloudflare_zero_trust_access_group" "lists" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Lists Group"

  include = [
    {
      email_list = {
        id = "list-id-1"
      }
    },
    {
      ip_list = {
        id = "iplist-id-1"
      }
    },
    {
      ip_list = {
        id = "iplist-id-2"
      }
    },
    {
      ip = {
        ip = "192.0.2.10/32"
      }
    },
    {
      ip = {
        ip = "192.0.2.11/32"
      }
    },
    {
      ip = {
        ip = "198.51.100.1/32"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.lists
  to   = cloudflare_zero_trust_access_group.lists
}

# Pattern 14: Login method and device posture
resource "cloudflare_zero_trust_access_group" "auth_methods" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Auth Methods Group"

  include = [
    {
      login_method = {
        id = "method-id-1"
      }
    },
    {
      login_method = {
        id = "method-id-2"
      }
    },
    {
      device_posture = {
        integration_uid = "posture-id-1"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.auth_methods
  to   = cloudflare_zero_trust_access_group.auth_methods
}

# Pattern 15: Common name selector
resource "cloudflare_zero_trust_access_group" "common_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Common Name Group"

  include = [
    {
      common_name = {
        common_name = "client.example.com"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.common_name
  to   = cloudflare_zero_trust_access_group.common_name
}

# Pattern 16: Auth method selector
resource "cloudflare_zero_trust_access_group" "auth_method_selector" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Auth Method Selector Group"

  include = [
    {
      auth_method = {
        auth_method = "email"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.auth_method_selector
  to   = cloudflare_zero_trust_access_group.auth_method_selector
}

# Pattern 17: Lifecycle meta-arguments
resource "cloudflare_zero_trust_access_group" "lifecycle_test" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Lifecycle Test Group"

  include = [
    {
      email = {
        email = "lifecycle@example.com"
      }
    },
  ]

  lifecycle {
    create_before_destroy = true
  }
}

moved {
  from = cloudflare_access_group.lifecycle_test
  to   = cloudflare_zero_trust_access_group.lifecycle_test
}

# Pattern 18: Terraform functions
resource "cloudflare_zero_trust_access_group" "functions" {
  account_id = var.cloudflare_account_id
  name       = join("-", [local.name_prefix, "Function", "Test", "Group"])

  include = [for i in range(2) : {
    email = {
      email = "user${i}@example.com"
    }
  }]
}

moved {
  from = cloudflare_access_group.functions
  to   = cloudflare_zero_trust_access_group.functions
}

# Pattern 20: Cross-resource references
resource "cloudflare_zero_trust_access_group" "parent" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Parent Group"

  include = [
    {
      email = {
        email = "parent@example.com"
      }
    },
  ]
}

moved {
  from = cloudflare_access_group.parent
  to   = cloudflare_zero_trust_access_group.parent
}

resource "cloudflare_zero_trust_access_group" "child" {
  account_id = var.cloudflare_account_id
  name       = "${local.name_prefix} Child Group - References ${cloudflare_zero_trust_access_group.parent.name}"

  include = [
    {
      email = {
        email = "child@example.com"
      }
    },
  ]

  depends_on = [cloudflare_zero_trust_access_group.parent]
}

moved {
  from = cloudflare_access_group.child
  to   = cloudflare_zero_trust_access_group.child
}
