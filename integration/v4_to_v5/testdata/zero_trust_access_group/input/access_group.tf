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

# Pattern 1: Simple email selector
resource "cloudflare_access_group" "simple_email" {
  account_id = var.cloudflare_account_id
  name       = "Simple Email Group"

  include {
    email = ["user1@example.com"]
  }
}

# Pattern 2: Multiple selector types
resource "cloudflare_access_group" "multiple_selectors" {
  account_id = var.cloudflare_account_id
  name       = "Multiple Selectors Group"

  include {
    email = ["admin@example.com", "manager@example.com"]
    ip    = ["192.168.1.0/24", "10.0.0.0/8"]
  }
}

# Pattern 3: for_each with map
resource "cloudflare_access_group" "foreach_map" {
  for_each = {
    team1 = "Engineering Team"
    team2 = "Product Team"
    team3 = "Design Team"
  }

  account_id = var.cloudflare_account_id
  name       = each.value

  include {
    email_domain = ["example.com"]
  }
}

# Pattern 4: for_each with set
resource "cloudflare_access_group" "foreach_set" {
  for_each = toset(["contractors", "vendors", "partners"])

  account_id = var.cloudflare_account_id
  name       = "${each.value}-group"

  include {
    ip = ["192.0.2.1/32"]
  }
}

# Pattern 5: count-based resources
resource "cloudflare_access_group" "count_based" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "Count Group ${count.index}"

  include {
    ip = ["10.${count.index}.0.0/16"]
  }
}

# Pattern 6: Conditional resource
resource "cloudflare_access_group" "conditional" {
  count = var.cloudflare_account_id != "" ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "Conditional Group"

  include {
    everyone = true
  }
}

# Pattern 7: Boolean selectors
resource "cloudflare_access_group" "booleans" {
  account_id = var.cloudflare_account_id
  name       = "Boolean Selectors Group"

  include {
    everyone = true
  }

  exclude {
    certificate = true
  }
}

# Pattern 8: Email domain selector
resource "cloudflare_access_group" "email_domains" {
  account_id = var.cloudflare_account_id
  name       = "Email Domain Group"

  include {
    email_domain = ["company.com", "partner.com", "vendor.com"]
  }
}

# Pattern 9: Geo selector
resource "cloudflare_access_group" "geo" {
  account_id = var.cloudflare_account_id
  name       = "Geo Group"

  include {
    geo = ["US", "CA", "GB"]
  }
}

# Pattern 10: All three rule types
resource "cloudflare_access_group" "all_rules" {
  account_id = var.cloudflare_account_id
  name       = "All Rules Group"

  include {
    email = ["user@example.com"]
  }

  exclude {
    ip = ["192.168.0.0/16"]
  }

  require {
    email_domain = ["example.com"]
  }
}

# Pattern 11: Multiple selectors in each rule
resource "cloudflare_access_group" "complex" {
  account_id = var.cloudflare_account_id
  name       = "Complex Group"

  include {
    email        = ["admin@example.com", "manager@example.com"]
    email_domain = ["example.com"]
    ip           = ["10.0.0.0/8"]
  }

  exclude {
    ip  = ["10.0.1.0/24"]
    geo = ["CN", "RU"]
  }
}

# Pattern 12: Service token selector
resource "cloudflare_access_group" "service_tokens" {
  account_id = var.cloudflare_account_id
  name       = "Service Token Group"

  include {
    any_valid_service_token = true
  }
}

# Pattern 13: Group and email list selectors
resource "cloudflare_access_group" "lists" {
  account_id = var.cloudflare_account_id
  name       = "Lists Group"

  include {
    ip         = ["192.0.2.10/32", "192.0.2.11/32", "198.51.100.1/32"]
    email_list = ["list-id-1"]
    ip_list    = ["iplist-id-1", "iplist-id-2"]
  }
}

# Pattern 14: Login method and device posture
resource "cloudflare_access_group" "auth_methods" {
  account_id = var.cloudflare_account_id
  name       = "Auth Methods Group"

  include {
    login_method   = ["method-id-1", "method-id-2"]
    device_posture = ["posture-id-1"]
  }
}

# Pattern 15: Common name selector
resource "cloudflare_access_group" "common_name" {
  account_id = var.cloudflare_account_id
  name       = "Common Name Group"

  include {
    common_name = "client.example.com"
  }
}

# Pattern 16: Auth method selector
resource "cloudflare_access_group" "auth_method_selector" {
  account_id = var.cloudflare_account_id
  name       = "Auth Method Selector Group"

  include {
    auth_method = "email"
  }
}

# Pattern 17: Lifecycle meta-arguments
resource "cloudflare_access_group" "lifecycle_test" {
  account_id = var.cloudflare_account_id
  name       = "Lifecycle Test Group"

  include {
    email = ["lifecycle@example.com"]
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Pattern 18: Terraform functions
resource "cloudflare_access_group" "functions" {
  account_id = var.cloudflare_account_id
  name       = join("-", ["Function", "Test", "Group"])

  include {
    email = [for i in range(2) : "user${i}@example.com"]
  }
}

# Pattern 20: Cross-resource references
resource "cloudflare_access_group" "parent" {
  account_id = var.cloudflare_account_id
  name       = "Parent Group"

  include {
    email = ["parent@example.com"]
  }
}

resource "cloudflare_access_group" "child" {
  account_id = var.cloudflare_account_id
  name       = "Child Group - References ${cloudflare_access_group.parent.name}"

  include {
    email = ["child@example.com"]
  }

  depends_on = [cloudflare_access_group.parent]
}
