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
  common_account     = var.cloudflare_account_id
  name_prefix        = "v5-upgrade-${replace(var.from_version, ".", "-")}-dlp2"
  tags               = ["migration", "test", "v4_to_v5"]
  gateway_precedence = 100000 + tonumber(split(".", var.from_version)[1]) * 1000

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












# ============================================================================
# Pattern Group 9: Cross-File References (Resource Rename Test)
# ============================================================================

# Pattern 9 tests that cross-file references are updated when resource names change.
# The migrator renames:
#   cloudflare_dlp_profile -> cloudflare_zero_trust_dlp_custom_profile
#   cloudflare_zero_trust_dlp_profile -> cloudflare_zero_trust_dlp_custom_profile
# We create dependent resources (gateway policies) that reference these DLP profiles via profile_id attribute.
# After migration, the attribute references must be updated to use the new resource names.



# Dependent resources that reference the above DLP profiles
# Using realistic gateway policies with profile_id references

# Gateway policy referencing old-name DLP profile via attribute
resource "cloudflare_zero_trust_gateway_policy" "uses_old_dlp_profile" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix} Gateway Policy - Old DLP Profile"
  description = "Policy using old-name DLP profile"
  action      = "block"
  precedence  = local.gateway_precedence
  enabled     = true
  traffic     = "any(dns.domains[*] == \"dlp-old.example.com\")"

  # Attribute reference that needs updating after migration
  # profile_id = cloudflare_zero_trust_dlp_custom_profile.ref_source_old_name.id
}

# Gateway policy referencing new-name DLP profile via attribute
resource "cloudflare_zero_trust_gateway_policy" "uses_new_dlp_profile" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix} Gateway Policy - New DLP Profile"
  description = "Policy using new-name DLP profile"
  action      = "allow"
  precedence  = local.gateway_precedence + 1
  enabled     = true
  traffic     = "any(dns.domains[*] == \"dlp-new.example.com\")"

  # Attribute reference that needs updating after migration
  # profile_id = cloudflare_zero_trust_dlp_custom_profile.ref_source_new_name.id
}

# Pattern 1: Basic profiles with entries
resource "cloudflare_zero_trust_dlp_custom_profile" "credit_cards_basic" {
  account_id          = local.common_account
  name                = "${local.name_prefix}-credit-cards-basic"
  description         = "Basic profile for detecting credit card numbers"
  allowed_match_count = var.max_allowed_matches


  entries = [{
    name    = "${local.name_prefix}-visa-card"
    enabled = true
    pattern = {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
    }, {
    name    = "${local.name_prefix}-mastercard"
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
    name    = "${local.name_prefix}-ssn-pattern"
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
    name    = "${local.name_prefix}-simple-pattern"
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
    name    = "${local.name_prefix}-${each.key}-pattern"
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
  name                = "${local.name_prefix}-v5-pii-${each.value}"
  allowed_match_count = 2

  entries = [{
    name    = "${local.name_prefix}-v5-pii-${each.value}-pattern"
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
    name    = "${local.name_prefix}-pattern-${count.index}"
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
    name    = "${local.name_prefix}-all-credit-cards"
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
    name    = "${local.name_prefix}-pii-pattern"
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
    name    = "${local.name_prefix}-api-key"
    enabled = true
    pattern = {
      regex = "api_key[\\s:=][a-zA-Z0-9]{20,30}"
    }
    }, {
    name    = "${local.name_prefix}-password"
    enabled = false
    pattern = {
      regex = "password[\\s:=][a-zA-Z0-9]{8,16}"
    }
    }, {
    name    = "${local.name_prefix}-token"
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
    name    = "${local.name_prefix}-entry-with-null-id"
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
    name    = "${local.name_prefix}-protected-pattern"
    enabled = true
    pattern = {
      regex = "PROTECTED[0-9]{1,4}"
    }
  }]
}


# Using old v4 name (cloudflare_dlp_profile) -> becomes cloudflare_zero_trust_dlp_custom_profile
resource "cloudflare_zero_trust_dlp_custom_profile" "ref_source_old_name" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-ref-old-name"
  description         = "Referenced by gateway policy (old name)"
  allowed_match_count = 10

  entries = [{
    name    = "${local.name_prefix}-reference-pattern-old"
    enabled = true
    pattern = {
      regex = "REF-OLD-[0-9]{4}"
    }
  }]
}


# Using second v4 name (cloudflare_zero_trust_dlp_profile) -> becomes cloudflare_zero_trust_dlp_custom_profile
resource "cloudflare_zero_trust_dlp_custom_profile" "ref_source_new_name" {
  account_id          = var.cloudflare_account_id
  name                = "${local.name_prefix}-ref-new-name"
  description         = "Referenced by gateway policy (new name)"
  allowed_match_count = 15

  entries = [{
    name    = "${local.name_prefix}-reference-pattern-new"
    enabled = true
    pattern = {
      regex = "REF-NEW-[0-9]{4}"
    }
  }]
}


variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}
