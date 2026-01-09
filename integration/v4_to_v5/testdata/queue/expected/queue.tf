# Integration test for cloudflare_queue migration (v4 → v5)
# This test covers all Terraform patterns and queue field transformations

# Standard variables provided by test infrastructure
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

# Locals for DRY configuration
locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest"
  environment    = "test"
  queue_names = [
    "queue-alpha",
    "queue-beta",
    "queue-gamma"
  ]
}

# ============================================================================
# Pattern 1-2: Basic Resources with Variables and Locals
# ============================================================================

# Minimal queue with required fields only
resource "cloudflare_queue" "minimal" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-minimal-queue"
}

# Queue using locals
resource "cloudflare_queue" "with_locals" {
  account_id = local.common_account
  queue_name = "${local.name_prefix}-locals-queue"
}

# Queue with string interpolation
resource "cloudflare_queue" "with_interpolation" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-${local.environment}-queue"
}

# Queue with numbers and hyphens in name
resource "cloudflare_queue" "special_chars" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-special-chars-queue-123"
}

# ============================================================================
# Pattern 3: for_each with Maps
# ============================================================================

# Create multiple queues using for_each with a map
resource "cloudflare_queue" "map_based" {
  for_each = {
    "events"   = "events-processing"
    "jobs"     = "background-jobs"
    "messages" = "user-messages"
    "webhooks" = "webhook-delivery"
  }

  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-${each.value}-queue"
}

# ============================================================================
# Pattern 4: for_each with Sets (using toset())
# ============================================================================

# Create queues from a set of names
resource "cloudflare_queue" "set_based" {
  for_each = toset(local.queue_names)

  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-${each.value}"
}

# ============================================================================
# Pattern 5: Count-based Resources
# ============================================================================

# Create numbered queues using count
resource "cloudflare_queue" "count_based" {
  count = 5

  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-numbered-queue-${count.index + 1}"
}

# ============================================================================
# Pattern 6: Conditional Resource Creation
# ============================================================================

locals {
  enable_priority_queue   = true
  enable_deadletter_queue = false
}

# Queue created conditionally
resource "cloudflare_queue" "priority" {
  count = local.enable_priority_queue ? 1 : 0

  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-priority-queue"
}

# Queue NOT created (condition false)
resource "cloudflare_queue" "deadletter" {
  count = local.enable_deadletter_queue ? 1 : 0

  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-deadletter-queue"
}

# ============================================================================
# Pattern 7: Cross-resource References
# ============================================================================

# Queue that could be referenced by other resources
resource "cloudflare_queue" "main" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-main-processing-queue"
}

# Another queue that could reference the first (simulated dependency)
resource "cloudflare_queue" "dependent" {
  account_id = cloudflare_queue.main.account_id
  queue_name = "${local.name_prefix}-dependent-on-${cloudflare_queue.main.name}"
}

# ============================================================================
# Pattern 8: Lifecycle Meta-arguments
# ============================================================================

# Queue with lifecycle configuration
resource "cloudflare_queue" "with_lifecycle" {
  account_id = var.cloudflare_account_id

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = false
  }
  queue_name = "${local.name_prefix}-lifecycle-queue"
}

# Queue with ignore_changes
resource "cloudflare_queue" "ignore_changes" {
  account_id = var.cloudflare_account_id

  lifecycle {
    ignore_changes = [queue_name]
  }
  queue_name = "${local.name_prefix}-ignore-changes-queue"
}

# ============================================================================
# Pattern 9: Terraform Functions
# ============================================================================

# Queue using join() function
resource "cloudflare_queue" "with_join" {
  account_id = var.cloudflare_account_id
  queue_name = join("-", [local.name_prefix, "joined", "queue"])
}

# Queue using lower() function
resource "cloudflare_queue" "with_lower" {
  account_id = var.cloudflare_account_id
  queue_name = lower("${local.name_prefix}-LOWERCASE-QUEUE")
}

# Queue using format() function
resource "cloudflare_queue" "with_format" {
  account_id = var.cloudflare_account_id
  queue_name = format("%s-formatted-queue-%02d", local.name_prefix, 42)
}

# ============================================================================
# Additional Edge Cases
# ============================================================================

# Queue with longer name (within 63 char limit)
resource "cloudflare_queue" "long_name" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-long-name-queue-test-example"
}

# Queue with numbers in name
resource "cloudflare_queue" "with_numbers" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-queue-123-456-789"
}

# Queue with multiple hyphens
resource "cloudflare_queue" "mixed_separators" {
  account_id = var.cloudflare_account_id
  queue_name = "${local.name_prefix}-queue-with-multiple-hyphens"
}
