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

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

# Test Case 1: Single header characteristic
resource "cloudflare_api_shield" "single_header" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }
}

# Test Case 2: Single cookie characteristic
resource "cloudflare_api_shield" "single_cookie" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }
}

# Test Case 3: Multiple header characteristics
resource "cloudflare_api_shield" "multiple_headers" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-api-key"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-client-id"
  }
}

# Test Case 4: Multiple cookie characteristics
resource "cloudflare_api_shield" "multiple_cookies" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "user_token"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "auth_token"
  }
}

# Test Case 5: Mixed header and cookie characteristics
resource "cloudflare_api_shield" "mixed_types" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-api-key"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "user_token"
  }
}

# Test Case 6: Header with common name
resource "cloudflare_api_shield" "common_header_1" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "X-Auth-Token"
  }
}

# Test Case 7: Cookie with common name
resource "cloudflare_api_shield" "common_cookie_1" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "JSESSIONID"
  }
}

# Test Case 8: Multiple characteristics with varied order
resource "cloudflare_api_shield" "varied_order" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "auth_cookie"
  }

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-Forwarded-For"
  }
}

# Test Case 9: Header with lowercase name
resource "cloudflare_api_shield" "lowercase_header" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "content-type"
  }
}

# Test Case 10: Cookie with uppercase name
resource "cloudflare_api_shield" "uppercase_cookie" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "SESSIONID"
  }
}

# Test Case 11: Multiple headers with hyphens
resource "cloudflare_api_shield" "hyphenated_headers" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "x-api-key"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-client-id"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-request-id"
  }
}

# Test Case 12: Multiple cookies with underscores
resource "cloudflare_api_shield" "underscore_cookies" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "user_id"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session_token"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "csrf_token"
  }
}

# Test Case 13: Single characteristic with capitalized type
resource "cloudflare_api_shield" "basic_auth" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }
}

# Test Case 14: Cookie for session management
resource "cloudflare_api_shield" "session_management" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "cookie"
    name = "session_key"
  }
}

# Test Case 15: Multiple characteristics for OAuth
resource "cloudflare_api_shield" "oauth_flow" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-OAuth-Token"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "oauth_state"
  }
}

# Test Case 16: JWT authentication headers
resource "cloudflare_api_shield" "jwt_auth" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-JWT-Token"
  }
}

# Test Case 17: API key authentication
resource "cloudflare_api_shield" "api_key_auth" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "X-API-Key"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-API-Secret"
  }
}

# Test Case 18: Bearer token authentication
resource "cloudflare_api_shield" "bearer_auth" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }
}

# Test Case 19: Custom authentication scheme
resource "cloudflare_api_shield" "custom_auth" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "X-Custom-Auth"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "custom_session"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-Custom-Token"
  }
}

# Test Case 20: Multi-factor authentication
resource "cloudflare_api_shield" "mfa_auth" {
  zone_id = var.cloudflare_zone_id

  auth_id_characteristics {
    type = "header"
    name = "Authorization"
  }

  auth_id_characteristics {
    type = "header"
    name = "X-MFA-Token"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "mfa_session"
  }
}
