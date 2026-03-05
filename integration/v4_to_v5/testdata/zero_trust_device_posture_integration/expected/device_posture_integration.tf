# Pattern 1-2: Variables & Locals (no defaults for account_id/zone_id)
variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_zone_id" {
  type = string
}

variable "integration_interval" {
  type    = string
  default = "12h"
}

variable "enable_integrations" {
  type    = bool
  default = true
}

locals {
  integration_prefix = "tf-acc-test"
  ws1_api_url        = "https://as123.awmdm.com/api"
  ws1_auth_url       = "https://na.uemauth.vmwservices.com/connect/token"
}














# Total resource instances:
# - map_integrations: 3 (workspace_one, crowdstrike, uptycs)
# - set_integrations: 3 (intune, kolide, sentinelone_s2s)
# - count_integrations: 3
# - conditional: 2
# - primary: 1
# - secondary: 1
# - lifecycle_test: 1
# - prevent_destroy: 1
# - function_test: 1
# - minimal: 1
# - all_fields: 1
# - no_interval: 1
# - dynamic_test: 1
# TOTAL: 20 resource instances

# Pattern 3: for_each with maps (3-5 resources)
resource "cloudflare_zero_trust_device_posture_integration" "map_integrations" {
  for_each = {
    workspace_one = {
      type      = "workspace_one"
      api_url   = local.ws1_api_url
      auth_url  = local.ws1_auth_url
      client_id = "ws1-client-id"
      secret    = "ws1-secret"
    }
    crowdstrike = {
      type        = "crowdstrike_s2s"
      api_url     = null
      auth_url    = null
      client_id   = "cs-client-id"
      secret      = "cs-secret"
      customer_id = "cs-customer"
    }
    uptycs = {
      type      = "uptycs"
      api_url   = "https://uptycs.example.com"
      auth_url  = null
      client_id = "uptycs-client"
      secret    = "uptycs-key"
    }
  }

  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-${each.key}"
  type       = each.value.type
  interval   = var.integration_interval

  config = {
    api_url       = each.value.api_url
    auth_url      = each.value.auth_url
    client_id     = each.value.client_id
    client_secret = each.value.secret
    customer_id   = lookup(each.value, "customer_id", null)
    client_key    = lookup(each.value, "client_key", null)
  }
}

moved {
  from = cloudflare_device_posture_integration.map_integrations
  to   = cloudflare_zero_trust_device_posture_integration.map_integrations
}

# Pattern 4: for_each with sets (3-5 items)
resource "cloudflare_zero_trust_device_posture_integration" "set_integrations" {
  for_each = toset(["intune", "kolide", "sentinelone_s2s"])

  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-${each.key}"
  type       = each.key
  interval   = "24h"

  config = {
    client_id     = "${each.key}-client"
    client_secret = "${each.key}-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.set_integrations
  to   = cloudflare_zero_trust_device_posture_integration.set_integrations
}

# Pattern 5: count-based resources (at least 3)
resource "cloudflare_zero_trust_device_posture_integration" "count_integrations" {
  count = 3

  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-count-${count.index}"
  type       = "custom_s2s"
  interval   = "1h"

  config = {
    api_url              = "https://custom-api-${count.index}.example.com"
    access_client_id     = "access-id-${count.index}"
    access_client_secret = "access-secret-${count.index}"
  }
}

moved {
  from = cloudflare_device_posture_integration.count_integrations
  to   = cloudflare_zero_trust_device_posture_integration.count_integrations
}

# Pattern 6: Conditional resource creation (count with ternary)
resource "cloudflare_zero_trust_device_posture_integration" "conditional" {
  count = var.enable_integrations ? 2 : 0

  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-conditional-${count.index}"
  type       = "tanium_s2s"

  interval = "24h"
  config = {
    api_url       = "https://tanium-${count.index}.example.com"
    client_secret = "tanium-secret-${count.index}"
  }
}

moved {
  from = cloudflare_device_posture_integration.conditional
  to   = cloudflare_zero_trust_device_posture_integration.conditional
}

# Pattern 7: Cross-resource references
resource "cloudflare_zero_trust_device_posture_integration" "primary" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-primary"
  type       = "workspace_one"
  interval   = "24h"

  config = {
    api_url       = "https://primary.example.com"
    auth_url      = "https://auth.primary.example.com"
    client_id     = "primary-client"
    client_secret = "primary-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.primary
  to   = cloudflare_zero_trust_device_posture_integration.primary
}

resource "cloudflare_zero_trust_device_posture_integration" "secondary" {
  # References the primary integration's name
  account_id = var.cloudflare_account_id
  name       = "${cloudflare_zero_trust_device_posture_integration.primary.name}-secondary"
  type       = "crowdstrike_s2s"
  interval   = cloudflare_zero_trust_device_posture_integration.primary.interval

  config = {
    client_id     = "secondary-client"
    client_secret = "secondary-secret"
    customer_id   = "secondary-customer"
  }
}

moved {
  from = cloudflare_device_posture_integration.secondary
  to   = cloudflare_zero_trust_device_posture_integration.secondary
}

# Pattern 8: Lifecycle meta-arguments
resource "cloudflare_zero_trust_device_posture_integration" "lifecycle_test" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-lifecycle"
  type       = "uptycs"
  interval   = "24h"


  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      config[0].client_key
    ]
  }
  config = {
    api_url    = "https://uptycs-lifecycle.example.com"
    client_id  = "lifecycle-client"
    client_key = "lifecycle-key"
  }
}

moved {
  from = cloudflare_device_posture_integration.lifecycle_test
  to   = cloudflare_zero_trust_device_posture_integration.lifecycle_test
}

resource "cloudflare_zero_trust_device_posture_integration" "prevent_destroy" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-prevent-destroy"
  type       = "kolide"
  interval   = "24h"


  lifecycle {
    prevent_destroy = true
  }
  config = {
    client_id            = "prevent-client"
    client_secret        = "prevent-secret"
    access_client_id     = "prevent-access-id"
    access_client_secret = "prevent-access-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.prevent_destroy
  to   = cloudflare_zero_trust_device_posture_integration.prevent_destroy
}

# Pattern 9: Terraform functions
resource "cloudflare_zero_trust_device_posture_integration" "function_test" {
  account_id = var.cloudflare_account_id
  name       = upper("${local.integration_prefix}-functions")
  type       = "workspace_one"
  interval   = "24h"

  config = {
    api_url       = join("/", ["https://functions.example.com", "api", "v1"])
    auth_url      = format("https://%s.auth.example.com", "functions")
    client_id     = base64encode("function-client")
    client_secret = base64encode("function-secret")
  }
}

moved {
  from = cloudflare_device_posture_integration.function_test
  to   = cloudflare_zero_trust_device_posture_integration.function_test
}

# Additional edge cases
resource "cloudflare_zero_trust_device_posture_integration" "minimal" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-minimal"
  type       = "custom_s2s"
  interval   = "30m"

  config = {
    api_url = "https://minimal.example.com"
  }
}

moved {
  from = cloudflare_device_posture_integration.minimal
  to   = cloudflare_zero_trust_device_posture_integration.minimal
}

resource "cloudflare_zero_trust_device_posture_integration" "all_fields" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-all-fields"
  type       = "workspace_one"
  interval   = "6h"

  config = {
    api_url              = "https://all-fields.example.com"
    auth_url             = "https://auth.all-fields.example.com"
    client_id            = "all-fields-client"
    client_secret        = "all-fields-secret"
    customer_id          = "all-fields-customer"
    client_key           = "all-fields-key"
    access_client_id     = "all-fields-access-id"
    access_client_secret = "all-fields-access-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.all_fields
  to   = cloudflare_zero_trust_device_posture_integration.all_fields
}

resource "cloudflare_zero_trust_device_posture_integration" "no_interval" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-no-interval"
  type       = "intune"

  interval = "24h"
  config = {
    client_id     = "no-interval-client"
    client_secret = "no-interval-secret"
  }
}

moved {
  from = cloudflare_device_posture_integration.no_interval
  to   = cloudflare_zero_trust_device_posture_integration.no_interval
}

# Dynamic block example
resource "cloudflare_zero_trust_device_posture_integration" "dynamic_test" {
  account_id = var.cloudflare_account_id
  name       = "${local.integration_prefix}-dynamic"
  type       = "crowdstrike_s2s"
  interval   = "24h"

  dynamic "config" {
    for_each = [1]
    content {
      client_id     = "dynamic-client"
      client_secret = "dynamic-secret"
      customer_id   = "dynamic-customer"
    }
  }
}

moved {
  from = cloudflare_device_posture_integration.dynamic_test
  to   = cloudflare_zero_trust_device_posture_integration.dynamic_test
}
