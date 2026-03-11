# Integration Test for observatory_scheduled_test v4 → v5 Migration
# No config transformations needed - all fields remain unchanged

# Pattern 1: Variables
variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for testing"
  type        = string
}

# Pattern 2: Basic resource
resource "cloudflare_observatory_scheduled_test" "basic" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://basic.cf-tf-test.com"
  region    = "us-central1"
  frequency = "DAILY"
}

# Pattern 3: WEEKLY frequency
resource "cloudflare_observatory_scheduled_test" "weekly" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://weekly.cf-tf-test.com"
  region    = "europe-west1"
  frequency = "WEEKLY"
}

# Pattern 4: Different regions
resource "cloudflare_observatory_scheduled_test" "asia" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://asia.cf-tf-test.com"
  region    = "asia-east1"
  frequency = "DAILY"
}

resource "cloudflare_observatory_scheduled_test" "us_west" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://us-west.cf-tf-test.com"
  region    = "us-west1"
  frequency = "DAILY"
}

# Pattern 5: URL with path
resource "cloudflare_observatory_scheduled_test" "with_path" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://path.cf-tf-test.com/test/page"
  region    = "us-central1"
  frequency = "DAILY"
}

# Pattern 6: URL with trailing slash
resource "cloudflare_observatory_scheduled_test" "trailing_slash" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://trailing.cf-tf-test.com/"
  region    = "us-west1"
  frequency = "WEEKLY"
}

# Pattern 7: URL with query parameters
resource "cloudflare_observatory_scheduled_test" "with_query" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://query.cf-tf-test.com/page?param=value&foo=bar"
  region    = "europe-west1"
  frequency = "DAILY"
}

# Pattern 8: for_each with list
locals {
  test_domains = ["test1", "test2", "test3"]
}

resource "cloudflare_observatory_scheduled_test" "for_expression" {
  for_each = toset([for domain in local.test_domains : domain])

  zone_id   = var.cloudflare_zone_id
  url       = "https://${each.value}.cf-tf-test.com"
  region    = "us-central1"
  frequency = "DAILY"
}

# Pattern 9: count-based resources
resource "cloudflare_observatory_scheduled_test" "counted" {
  count = 2

  zone_id   = var.cloudflare_zone_id
  url       = "https://count-${count.index}.cf-tf-test.com"
  region    = "asia-east1"
  frequency = "WEEKLY"
}

# Pattern 10: String interpolation
resource "cloudflare_observatory_scheduled_test" "interpolated" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://${lower("CFTFTEST")}-interpolated.cf-tf-test.com"
  region    = "europe-west1"
  frequency = "DAILY"
}

# Pattern 11: Complex URL with subdomain and path
resource "cloudflare_observatory_scheduled_test" "complex_url" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://subdomain.domain.cf-tf-test.com/path/to/resource?key=value&foo=bar"
  region    = "asia-east1"
  frequency = "WEEKLY"
}

# Pattern 12: Resource depends_on
resource "cloudflare_observatory_scheduled_test" "depends_implicit" {
  zone_id   = var.cloudflare_zone_id
  url       = "https://depends.cf-tf-test.com"
  region    = "us-west1"
  frequency = "DAILY"

  # Implicit dependency through variable reference is enough
}
