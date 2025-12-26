# Variables (standard - provided by test framework)
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

# Resource-specific variables with defaults
variable "enable_credit_detection" {
  type    = bool
  default = true
}

variable "enable_pii_detection" {
  type    = bool
  default = false
}

variable "max_allowed_matches" {
  type    = number
  default = 5
}

# Locals for common values and computed configurations
locals {
  common_account = var.cloudflare_account_id
  name_prefix    = "cftftest-dlp"
  tags           = ["migration", "test", "v4_to_v5"]

  # Map for for_each iteration
  card_patterns = {
    visa = {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
      enabled    = true
    }
    mastercard = {
      regex      = "5[1-5][0-9]{14}"
      validation = "luhn"
      enabled    = true
    }
    amex = {
      regex      = "3[47][0-9]{13}"
      validation = "luhn"
      enabled    = false
    }
    discover = {
      regex      = "6(?:011|5[0-9]{2})[0-9]{12}"
      validation = "luhn"
      enabled    = true
    }
  }

  # Set for for_each iteration
  pii_types = toset([
    "ssn",
    "passport",
    "drivers_license",
    "tax_id"
  ])
}

# Pattern 1: Basic profiles with entries
resource "cloudflare_zero_trust_dlp_custom_profile" "credit_cards_basic" {
  account_id          = local.common_account
  name                = "${local.name_prefix}-credit-cards-basic"
  description         = "Basic profile for detecting credit card numbers"
  allowed_match_count = var.max_allowed_matches


  entries = [{
    name    = "Visa Card"
    enabled = true
    pattern = {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
    }, {
    name    = "Mastercard"
    enabled = true
    pattern = {
      regex      = "5[1-5][0-9]{14}"
      validation = "luhn"
    }
  }]
}

resource "cloudflare_zero_trust_dlp_custom_profile" "ssn_detection" {
  account_id          = var.cloudflare_account_id
  name                = join("-", [local.name_prefix, "ssn", "detection"])
  allowed_match_count = 3

  entries = [{
    name    = "SSN Pattern"
    enabled = true
    pattern = {
      regex = "[0-9]{3}-[0-9]{2}-[0-9]{4}"
    }
  }]
}

resource "cloudflare_zero_trust_dlp_custom_profile" "minimal" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-minimal"
  description         = "Minimal configuration for ${var.cloudflare_account_id}"
  allowed_match_count = 1

  entries = [{
    name    = "Simple Pattern"
    enabled = true
    pattern = {
      regex = "test[0-9]{1,10}"
    }
  }]
}

# Pattern 2: for_each with maps (4 instances)
resource "cloudflare_zero_trust_dlp_custom_profile" "card_profiles_map" {
  for_each = local.card_patterns

  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-card-${each.key}"
  description         = "Dedicated profile for ${each.key} cards"
  allowed_match_count = 10

  entries = [{
    name    = "${title(each.key)} Pattern"
    enabled = each.value.enabled
    pattern = {
      regex      = each.value.regex
      validation = each.value.validation
    }
  }]
}

# Pattern 3: for_each with sets (4 instances)
resource "cloudflare_zero_trust_dlp_custom_profile" "pii_profiles_set" {
  for_each = local.pii_types

  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-pii-${each.value}"
  allowed_match_count = 2

  entries = [{
    name    = "${title(replace(each.value, "_", " "))} Pattern"
    enabled = true
    pattern = {
      regex = "[A-Z0-9]{8,15}"
    }
  }]
}

# Pattern 4: count-based resources (3 instances)
resource "cloudflare_zero_trust_dlp_custom_profile" "counted_profiles" {
  count = 3

  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-counted-${count.index}"
  description         = "This is profile number ${count.index}"
  allowed_match_count = count.index + 1

  entries = [{
    name    = "Pattern ${count.index}"
    enabled = count.index % 2 == 0 ? true : false
    pattern = {
      regex = "pattern-${count.index}-[0-9]{1,4}"
    }
  }]
}

# Pattern 5: Conditional creation (1 instance when enabled)
resource "cloudflare_zero_trust_dlp_custom_profile" "conditional_credit" {
  count = var.enable_credit_detection ? 1 : 0

  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-conditional-credit"
  description         = "Created conditionally based on variable"
  allowed_match_count = 15

  entries = [{
    name    = "All Credit Cards"
    enabled = true
    pattern = {
      regex      = "(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})"
      validation = "luhn"
    }
  }]
}

resource "cloudflare_zero_trust_dlp_custom_profile" "conditional_pii" {
  count = var.enable_pii_detection ? 1 : 0

  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-conditional-pii"
  allowed_match_count = 20

  entries = [{
    name    = "PII Pattern"
    enabled = true
    pattern = {
      regex = "[A-Z]{2}[0-9]{6}"
    }
  }]
}

# Pattern 6: Multiple entries
resource "cloudflare_zero_trust_dlp_custom_profile" "dynamic_entries" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-dynamic-entries"
  description         = "Profile with multiple entries"
  allowed_match_count = 30



  entries = [{
    name    = "Api Key"
    enabled = true
    pattern = {
      regex = "api_key[\\s:=][a-zA-Z0-9]{20,30}"
    }
    }, {
    name    = "Password"
    enabled = false
    pattern = {
      regex = "password[\\s:=][a-zA-Z0-9]{8,16}"
    }
    }, {
    name    = "Token"
    enabled = true
    pattern = {
      regex = "token[\\s:=][a-zA-Z0-9._-]{20,30}"
    }
  }]
}

# Pattern 7: Profile with null values
resource "cloudflare_zero_trust_dlp_custom_profile" "with_nulls" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-with-nulls"
  description         = null
  allowed_match_count = 4

  entries = [{
    name    = "Entry with null ID"
    enabled = true
    pattern = {
      regex      = "NULL[0-9]{1,4}"
      validation = null
    }
  }]
}

# Pattern 8: Profile with lifecycle meta-arguments
resource "cloudflare_zero_trust_dlp_custom_profile" "prevent_destroy" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-protected"
  description         = "This profile should not be destroyed"
  allowed_match_count = 50


  lifecycle {
    prevent_destroy = false
  }
  entries = [{
    name    = "Protected Pattern"
    enabled = true
    pattern = {
      regex = "PROTECTED[0-9]{1,4}"
    }
  }]
}
