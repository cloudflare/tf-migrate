variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID"
  type        = string
}

resource "cloudflare_dlp_profile" "credit_cards" {
  account_id          = var.cloudflare_account_id
  name                = "Credit Card Detection"
  description         = "Custom profile for detecting credit card numbers"
  type                = "custom"
  allowed_match_count = 5

  entry {
    id      = "visa-card-pattern"
    name    = "Visa Card"
    enabled = true
    pattern {
      regex      = "4[0-9]{12}(?:[0-9]{3})?"
      validation = "luhn"
    }
  }

  entry {
    id      = "mastercard-pattern"
    name    = "Mastercard"
    enabled = true
    pattern {
      regex      = "5[1-5][0-9]{14}"
      validation = "luhn"
    }
  }

  entry {
    id      = "amex-pattern"
    name    = "American Express"
    enabled = false
    pattern {
      regex      = "3[47][0-9]{13}"
      validation = "luhn"
    }
  }
}

resource "cloudflare_dlp_profile" "ssn_detection" {
  account_id          = var.cloudflare_account_id
  name                = "SSN Detection"
  type                = "custom"
  allowed_match_count = 3

  entry {
    name    = "SSN Pattern"
    enabled = true
    pattern {
      regex = "[0-9]{3}-[0-9]{2}-[0-9]{4}"
    }
  }
}

resource "cloudflare_dlp_profile" "minimal" {
  account_id          = var.cloudflare_account_id
  name                = "Minimal Profile"
  type                = "custom"
  allowed_match_count = 1

  entry {
    name    = "Simple Pattern"
    enabled = true
    pattern {
      regex = "test[0-9]{1,10}"
    }
  }
}