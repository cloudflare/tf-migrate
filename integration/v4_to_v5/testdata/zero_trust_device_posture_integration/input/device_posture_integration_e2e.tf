# E2E Test Configuration for Device Posture Integration Migration
# This file uses CrowdStrike S2S integration type with real credentials from environment variables
# Required environment variables:
# - CLOUDFLARE_CROWDSTRIKE_CLIENT_ID
# - CLOUDFLARE_CROWDSTRIKE_CLIENT_SECRET
# - CLOUDFLARE_CROWDSTRIKE_API_URL
# - CLOUDFLARE_CROWDSTRIKE_CUSTOMER_ID

variable "cloudflare_account_id" {
  type = string
}

# These variables are provided by the e2e runner but not used by device_posture_integration
variable "cloudflare_zone_id" {
  type    = string
  default = ""
}

variable "cloudflare_domain" {
  type    = string
  default = ""
}

variable "crowdstrike_client_id" {
  type = string
}

variable "crowdstrike_client_secret" {
  type      = string
  sensitive = true
}

variable "crowdstrike_api_url" {
  type = string
}

variable "crowdstrike_customer_id" {
  type = string
}

locals {
  e2e_prefix = "tf-e2e-migrate"
}

# Test Case 1: Deprecated resource name with all fields
# Tests: cloudflare_device_posture_integration → cloudflare_zero_trust_device_posture_integration rename
# Tests: config block → attribute transformation
resource "cloudflare_device_posture_integration" "deprecated_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-deprecated"
  type       = "crowdstrike_s2s"
  interval   = "24h"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 2: Current resource name with all fields
# Tests: No resource name change (already using current name)
# Tests: config block → attribute transformation
resource "cloudflare_zero_trust_device_posture_integration" "current_name" {
  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-current"
  type       = "crowdstrike_s2s"
  interval   = "12h"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 3: Non-standard interval value
# Tests: Various interval formats (not just 24h)
# Tests: config block → attribute transformation
resource "cloudflare_zero_trust_device_posture_integration" "no_interval" {
  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-nointerval"
  type       = "crowdstrike_s2s"
  interval   = "168h"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 4: With deprecated identifier field
# Tests: identifier field is removed during migration
# Tests: Deprecated resource name → current name
# Tests: config block → attribute transformation
resource "cloudflare_device_posture_integration" "with_identifier" {
  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-identifier"
  type       = "crowdstrike_s2s"
  interval   = "6h"
  identifier = "legacy-e2e-identifier"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 5: Count-based resources
# Tests: Multiple resources with count
# Tests: Different interval values
# Tests: config block → attribute transformation
resource "cloudflare_zero_trust_device_posture_integration" "count_test" {
  count = 2

  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-count-${count.index}"
  type       = "crowdstrike_s2s"
  interval   = count.index == 0 ? "1h" : "2h"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 6: for_each with map
# Tests: for_each with map iteration
# Tests: Variable interval values
# Tests: Deprecated resource name with for_each
resource "cloudflare_device_posture_integration" "foreach_deprecated" {
  for_each = {
    hourly = "1h"
    daily  = "24h"
  }

  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-foreach-dep-${each.key}"
  type       = "crowdstrike_s2s"
  interval   = each.value

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 7: Cross-resource reference
# Tests: References between resources work after migration
# Tests: Deprecated name → current name for referenced resource
resource "cloudflare_device_posture_integration" "primary" {
  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-primary"
  type       = "crowdstrike_s2s"
  interval   = "24h"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

resource "cloudflare_zero_trust_device_posture_integration" "secondary" {
  account_id = var.cloudflare_account_id
  name       = "${cloudflare_device_posture_integration.primary.name}-secondary"
  type       = "crowdstrike_s2s"
  interval   = cloudflare_device_posture_integration.primary.interval

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }
}

# Test Case 8: Lifecycle meta-arguments
# Tests: Lifecycle blocks are preserved during migration
# Tests: config block → attribute transformation with lifecycle
resource "cloudflare_zero_trust_device_posture_integration" "lifecycle_test" {
  account_id = var.cloudflare_account_id
  name       = "${local.e2e_prefix}-lifecycle"
  type       = "crowdstrike_s2s"
  interval   = "24h"

  config {
    api_url       = var.crowdstrike_api_url
    client_id     = var.crowdstrike_client_id
    client_secret = var.crowdstrike_client_secret
    customer_id   = var.crowdstrike_customer_id
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Test Case 9: Comments preservation
# Tests: Comments are preserved during migration
resource "cloudflare_device_posture_integration" "with_comments" {
  # This is a comment before the resource
  account_id = var.cloudflare_account_id # inline comment
  name       = "${local.e2e_prefix}-comments"
  type       = "crowdstrike_s2s"
  interval   = "24h"

  # Comment before config block
  config {
    api_url       = var.crowdstrike_api_url       # API URL comment
    client_id     = var.crowdstrike_client_id     # Client ID comment
    client_secret = var.crowdstrike_client_secret # Secret comment
    customer_id   = var.crowdstrike_customer_id   # Customer ID comment
  }
  # Comment after config block
}

# Summary of E2E Test Cases:
# Total: 13 resource instances (1 + 1 + 1 + 1 + 2 + 2 + 2 + 1 + 1 + 1)
#
# Coverage:
# - Deprecated resource name rename: 6 instances (deprecated_name, with_identifier, foreach_deprecated x2, primary, with_comments)
# - Current resource name (no rename): 7 instances (current_name, no_interval, count_test x2, secondary, lifecycle_test)
# - Identifier field removal: 1 instance (with_identifier)
# - Different interval values: All instances (1h, 2h, 6h, 12h, 24h, 168h)
# - Config block → attribute: All instances
# - Count-based: 2 instances (count_test[0], count_test[1])
# - for_each with map: 2 instances (foreach_deprecated["hourly"], foreach_deprecated["daily"])
# - Cross-resource references: 2 instances (primary, secondary)
# - Lifecycle meta-arguments: 1 instance (lifecycle_test)
# - Comments preservation: 1 instance (with_comments)
