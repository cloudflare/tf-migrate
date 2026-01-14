# Zero Trust Access Application Migration Guide (v4 → v5)

This guide explains how `cloudflare_access_application` / `cloudflare_zero_trust_access_application` resources migrate to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_access_application` | `cloudflare_zero_trust_access_application` | Renamed |
| Alt resource name | `cloudflare_zero_trust_access_application` | `cloudflare_zero_trust_access_application` | No change |
| `domain_type` | Supported | Removed | Deprecated field |
| `type` default | No default | Defaults to "self_hosted" | Default added |
| `policies` | String array | Object array with id/precedence | Structure change |
| `allowed_idps` | `toset([...])` | Direct array | Function wrapper removed |
| `self_hosted_domains` | Unsorted | Sorted alphabetically | Ordering change |
| `cors_headers` | Block | Attribute object | Syntax change |
| `saas_app` | Block with nested blocks | Attribute with nested objects | Major restructuring |
| `scim_config` | Block with nested blocks | Attribute with nested objects | Restructuring |
| `landing_page_design` | Block | Attribute object | Syntax change |


---

## Migration Examples

### Example 1: Basic Self-Hosted Application

**v4 Configuration:**
```hcl
resource "cloudflare_access_application" "basic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Internal Dashboard"
  domain  = "dashboard.example.com"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_application" "basic" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "Internal Dashboard"
  domain  = "dashboard.example.com"
  type    = "self_hosted"  # Added by migration
}
```

**What Changed:**
- Resource type renamed
- `type = "self_hosted"` explicitly added

---

### Example 2: With Policies

**v4 Configuration:**
```hcl
resource "cloudflare_access_application" "with_policies" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "API Service"
  domain  = "api.example.com"

  policies = [
    cloudflare_access_policy.admin.id,
    cloudflare_access_policy.developers.id
  ]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_application" "with_policies" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "API Service"
  domain  = "api.example.com"
  type    = "self_hosted"

  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.admin.id
      precedence = 1
    },
    {
      id         = cloudflare_zero_trust_access_policy.developers.id
      precedence = 2
    }
  ]
}
```

**What Changed:**
- `policies` string array → object array
- Each policy gets `precedence` (1-indexed order)

---

### Example 3: With CORS Headers

**v4 Configuration:**
```hcl
resource "cloudflare_access_application" "cors" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "API with CORS"
  domain  = "api.example.com"

  cors_headers {
    allowed_methods   = ["GET", "POST"]
    allowed_origins   = ["https://app.example.com"]
    allow_credentials = true
    max_age           = 3600
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_application" "cors" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "API with CORS"
  domain  = "api.example.com"
  type    = "self_hosted"

  cors_headers = {
    allowed_methods   = ["GET", "POST"]
    allowed_origins   = ["https://app.example.com"]
    allow_credentials = true
    max_age           = 3600
  }
}
```

**What Changed:**
- `cors_headers { }` block → `cors_headers = { }` attribute

---

### Example 4: SAAS Application

**v4 Configuration:**
```hcl
resource "cloudflare_access_application" "saas" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Salesforce SSO"
  type       = "saas"

  saas_app {
    consumer_service_url = "https://example.salesforce.com"
    sp_entity_id         = "https://example.salesforce.com"
    name_id_format       = "email"

    custom_attribute {
      name   = "department"
      source {
        name_by_idp = {
          azure = "department"
          okta  = "dept"
        }
      }
    }

    custom_claim {
      name   = "groups"
      scope  = "groups"
      source {
        name = "user_groups"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_application" "saas" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Salesforce SSO"
  type       = "saas"

  saas_app = {
    consumer_service_url = "https://example.salesforce.com"
    sp_entity_id         = "https://example.salesforce.com"
    name_id_format       = "email"

    custom_attributes = [
      {
        name = "department"
        source = {
          name_by_idp = {
            azure = "department"
            okta  = "dept"
          }
        }
      }
    ]

    custom_claims = [
      {
        name  = "groups"
        scope = "groups"
        source = {
          name = "user_groups"
        }
      }
    ]
  }
}
```

**What Changed:**
- `saas_app { }` block → `saas_app = { }` attribute
- Nested `custom_attribute` blocks → `custom_attributes` array
- Nested `custom_claim` blocks → `custom_claims` array
- All nested blocks become objects

---

### Example 5: With SCIM Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_access_application" "scim" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SCIM Application"
  domain     = "scim.example.com"

  scim_config {
    enabled         = true
    remote_uri      = "https://api.example.com/scim/v2"
    idp_uid         = "email"
    deactivate_on_delete = true

    authentication {
      scheme   = "httpbasic"
      user     = "scim_user"
      password = var.scim_password
    }

    mappings {
      schema = "urn:ietf:params:scim:schemas:core:2.0:User"

      operations {
        create = true
        update = true
        delete = false
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_application" "scim" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SCIM Application"
  domain     = "scim.example.com"
  type       = "self_hosted"

  scim_config = {
    enabled              = true
    remote_uri           = "https://api.example.com/scim/v2"
    idp_uid              = "email"
    deactivate_on_delete = true

    authentication = {
      scheme   = "httpbasic"
      user     = "scim_user"
      password = var.scim_password
    }

    mappings = [{
      schema = "urn:ietf:params:scim:schemas:core:2.0:User"

      operations = {
        create = true
        update = true
        delete = false
      }
    }]
  }
}
```

**What Changed:**
- All blocks → attribute objects
- `mappings` block → array (allows multiple schemas)

---

