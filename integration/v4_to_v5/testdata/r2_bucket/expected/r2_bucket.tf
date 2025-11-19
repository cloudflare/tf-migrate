variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

# Test Case 1: Basic R2 bucket with required fields only
resource "cloudflare_r2_bucket" "basic" {
  account_id = var.cloudflare_account_id
  name       = "test-bucket"
}

# Test Case 2: R2 bucket with location (uppercase - v4 style)
resource "cloudflare_r2_bucket" "with_location_upper" {
  account_id = var.cloudflare_account_id
  name       = "bucket-wnam"
  location   = "WNAM"
}

# Test Case 3: R2 bucket with location (must be uppercase)
resource "cloudflare_r2_bucket" "with_location_lower" {
  account_id = var.cloudflare_account_id
  name       = "bucket-eeur"
  location   = "EEUR"
}

# Test Case 4: R2 bucket with variable reference
resource "cloudflare_r2_bucket" "with_variable" {
  account_id = var.cloudflare_account_id
  name       = "variable-bucket"
}

# Test Case 5: Multiple buckets with different configs
resource "cloudflare_r2_bucket" "multi1" {
  account_id = var.cloudflare_account_id
  name       = "multi-bucket-1"
}

resource "cloudflare_r2_bucket" "multi2" {
  account_id = var.cloudflare_account_id
  name       = "multi-bucket-2"
  location   = "APAC"
}
