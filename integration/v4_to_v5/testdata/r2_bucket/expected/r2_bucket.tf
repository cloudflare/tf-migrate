# Test Case 1: Basic R2 bucket with required fields only
resource "cloudflare_r2_bucket" "basic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-bucket"
}

# Test Case 2: R2 bucket with location (uppercase - v4 style)
resource "cloudflare_r2_bucket" "with_location_upper" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "bucket-wnam"
  location   = "WNAM"
}

# Test Case 3: R2 bucket with location (lowercase - also valid)
resource "cloudflare_r2_bucket" "with_location_lower" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "bucket-eeur"
  location   = "eeur"
}

# Test Case 4: R2 bucket with variable reference
variable "account_id" {
  type = string
}

resource "cloudflare_r2_bucket" "with_variable" {
  account_id = var.account_id
  name       = "variable-bucket"
}

# Test Case 5: Multiple buckets with different configs
resource "cloudflare_r2_bucket" "multi1" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "multi-bucket-1"
}

resource "cloudflare_r2_bucket" "multi2" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "multi-bucket-2"
  location   = "APAC"
}
