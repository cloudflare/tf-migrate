resource "cloudflare_zero_trust_dlp_custom_profile" "credit_cards" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "Credit Card Detection"
  description         = "Custom profile for detecting credit card numbers"
  allowed_match_count = 5



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
    }, {
    name    = "American Express"
    enabled = false
    pattern = {
      regex      = "3[47][0-9]{13}"
      validation = "luhn"
    }
  }]
}

resource "cloudflare_zero_trust_dlp_custom_profile" "ssn_detection" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "SSN Detection"
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
  account_id          = "f037e56e89293a057740de681ac9abbe"
  name                = "Minimal Profile"
  allowed_match_count = 1

  entries = [{
    name    = "Simple Pattern"
    enabled = true
    pattern = {
      regex = "test[0-9]+"
    }
  }]
}