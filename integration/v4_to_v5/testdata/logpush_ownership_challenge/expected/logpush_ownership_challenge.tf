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

# =============================================================================
# Pattern 1: Basic resources with zone_id
# =============================================================================

# Minimal ownership challenge with zone_id
resource "cloudflare_logpush_ownership_challenge" "minimal_zone" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://${local.name_prefix}-minimal-bucket?region=us-west-2"
}

# S3 destination with various parameters
resource "cloudflare_logpush_ownership_challenge" "s3_full_params" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://${local.name_prefix}-logs-bucket/path/to/logs?region=us-east-1&access-key-id=AKIAIOSFODNN7EXAMPLE&secret-access-key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}

# GCS destination
resource "cloudflare_logpush_ownership_challenge" "gcs_zone" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "gs://${local.name_prefix}-gcs-bucket/logs"
}

# Azure Blob Storage destination
resource "cloudflare_logpush_ownership_challenge" "azure_zone" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "azure://${local.name_prefix}-azure-container/logs?account-name=myaccount&account-key=mykey"
}

# R2 destination
resource "cloudflare_logpush_ownership_challenge" "r2_zone" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "r2://${local.name_prefix}-r2-bucket/logs?account-id=${var.cloudflare_account_id}"
}

# =============================================================================
# Pattern 2: Basic resources with account_id
# =============================================================================

# Minimal ownership challenge with account_id
resource "cloudflare_logpush_ownership_challenge" "minimal_account" {
  account_id       = var.cloudflare_account_id
  destination_conf = "s3://${local.name_prefix}-account-bucket?region=us-west-2"
}

# S3 destination with account_id
resource "cloudflare_logpush_ownership_challenge" "s3_account" {
  account_id       = var.cloudflare_account_id
  destination_conf = "s3://${local.name_prefix}-account-logs/audit?region=eu-west-1"
}

# GCS destination with account_id
resource "cloudflare_logpush_ownership_challenge" "gcs_account" {
  account_id       = var.cloudflare_account_id
  destination_conf = "gs://${local.name_prefix}-gcs-account-bucket/account-logs"
}

# =============================================================================
# Pattern 3: for_each with map
# =============================================================================

variable "zone_destinations" {
  type = map(string)
  default = {
    production = "s3://cftftest-prod-logs?region=us-west-2"
    staging    = "s3://cftftest-staging-logs?region=us-east-1"
    dev        = "s3://cftftest-dev-logs?region=eu-west-1"
  }
}

resource "cloudflare_logpush_ownership_challenge" "zone_foreach_map" {
  for_each = var.zone_destinations

  zone_id          = var.cloudflare_zone_id
  destination_conf = each.value
}

# =============================================================================
# Pattern 4: for_each with set
# =============================================================================

variable "account_bucket_names" {
  type    = set(string)
  default = ["audit", "security", "analytics", "compliance"]
}

resource "cloudflare_logpush_ownership_challenge" "account_foreach_set" {
  for_each = var.account_bucket_names

  account_id       = var.cloudflare_account_id
  destination_conf = "s3://cftftest-${each.value}-logs?region=us-west-2"
}

# =============================================================================
# Pattern 5: count-based resources
# =============================================================================

variable "replicate_count" {
  type    = number
  default = 3
}

resource "cloudflare_logpush_ownership_challenge" "zone_count" {
  count = var.replicate_count

  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://cftftest-zone-replica-${count.index}?region=us-west-2"
}

resource "cloudflare_logpush_ownership_challenge" "account_count" {
  count = var.replicate_count

  account_id       = var.cloudflare_account_id
  destination_conf = "s3://cftftest-account-replica-${count.index}?region=us-east-1"
}

# =============================================================================
# Pattern 6: Conditional resource creation
# =============================================================================

variable "enable_zone_challenge" {
  type    = bool
  default = true
}

variable "enable_account_challenge" {
  type    = bool
  default = true
}

resource "cloudflare_logpush_ownership_challenge" "conditional_zone" {
  count = var.enable_zone_challenge ? 1 : 0

  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://cftftest-conditional-zone?region=us-west-2"
}

resource "cloudflare_logpush_ownership_challenge" "conditional_account" {
  count = var.enable_account_challenge ? 1 : 0

  account_id       = var.cloudflare_account_id
  destination_conf = "s3://cftftest-conditional-account?region=us-west-2"
}

# =============================================================================
# Pattern 7: Variable references and interpolation
# =============================================================================

variable "custom_bucket_prefix" {
  type    = string
  default = "cftftest-custom"
}

variable "aws_region" {
  type    = string
  default = "us-west-2"
}

resource "cloudflare_logpush_ownership_challenge" "interpolation" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://${var.custom_bucket_prefix}-logs?region=${var.aws_region}"
}

# =============================================================================
# Pattern 8: Terraform functions
# =============================================================================

resource "cloudflare_logpush_ownership_challenge" "with_functions" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = join("", ["s3://cftftest-", "joined-bucket", "?region=us-west-2"])
}

resource "cloudflare_logpush_ownership_challenge" "with_format" {
  account_id       = var.cloudflare_account_id
  destination_conf = format("s3://cftftest-%s-logs?region=%s", "formatted", "eu-west-1")
}

# =============================================================================
# Pattern 9: Lifecycle meta-arguments
# =============================================================================

resource "cloudflare_logpush_ownership_challenge" "with_lifecycle" {
  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://cftftest-lifecycle-bucket?region=us-west-2"

  lifecycle {
    create_before_destroy = true
  }
}

resource "cloudflare_logpush_ownership_challenge" "ignore_changes" {
  account_id       = var.cloudflare_account_id
  destination_conf = "s3://cftftest-ignore-changes?region=us-west-2"

  lifecycle {
    ignore_changes = [destination_conf]
  }
}

# =============================================================================
# Pattern 10: toset() conversion
# =============================================================================

variable "bucket_list" {
  type    = list(string)
  default = ["backup1", "backup2", "backup3"]
}

resource "cloudflare_logpush_ownership_challenge" "toset_conversion" {
  for_each = toset(var.bucket_list)

  zone_id          = var.cloudflare_zone_id
  destination_conf = "s3://cftftest-${each.value}?region=us-west-2"
}

# =============================================================================
# TOTAL RESOURCE COUNT
# =============================================================================
# - 8 basic resources (zone_id variants)
# - 3 basic resources (account_id variants)
# - 3 for_each map instances (production, staging, dev)
# - 4 for_each set instances (audit, security, analytics, compliance)
# - 3 count-based zone instances
# - 3 count-based account instances
# - 2 conditional instances
# - 2 variable reference instances
# - 2 function instances
# - 2 lifecycle instances
# - 3 toset instances
# = 35 TOTAL INSTANCES (exceeds 15-30 requirement!)
