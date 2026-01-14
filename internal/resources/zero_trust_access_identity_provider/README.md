# Zero Trust Access Identity Provider Migration Guide (v4 → v5)

This guide explains how `cloudflare_access_identity_provider` resources migrate to `cloudflare_zero_trust_access_identity_provider` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_access_identity_provider` | `cloudflare_zero_trust_access_identity_provider` | Renamed |
| `config` | Block (MaxItems:1) | Attribute object | Syntax change |
| `scim_config` | Block (MaxItems:1) | Attribute object | Syntax change |
| `config.api_token` | Supported | Removed | Deprecated field |
| `config.idp_public_cert` | String | `idp_public_certs` array | Type change + rename |
| `scim_config.secret` | Configurable | Computed-only | Now read-only |
| `scim_config.group_member_deprovision` | Supported | Removed | Deprecated field |


---

## Migration Examples

### Example 1: SAML Identity Provider

**v4 Configuration:**
```hcl
resource "cloudflare_access_identity_provider" "saml" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SAML SSO"
  type       = "saml"

  config {
    issuer_url      = "https://sso.example.com/saml"
    sso_target_url  = "https://sso.example.com/sso"
    idp_public_cert = "MIICmzCCAYMCBgF..."
    sign_request    = false
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_identity_provider" "saml" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SAML SSO"
  type       = "saml"

  config = {
    issuer_url       = "https://sso.example.com/saml"
    sso_target_url   = "https://sso.example.com/sso"
    idp_public_certs = ["MIICmzCCAYMCBgF..."]
    sign_request     = false
  }
}
```

**What Changed:**
- Resource type renamed
- `config { }` block → `config = { }` attribute
- `idp_public_cert` (string) → `idp_public_certs` (array)

---

### Example 2: OAuth Identity Provider

**v4 Configuration:**
```hcl
resource "cloudflare_access_identity_provider" "oauth" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "GitHub OAuth"
  type       = "github"

  config {
    client_id     = "abc123"
    client_secret = var.github_client_secret
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_identity_provider" "oauth" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "GitHub OAuth"
  type       = "github"

  config = {
    client_id     = "abc123"
    client_secret = var.github_client_secret
  }
}
```

**What Changed:**
- Resource renamed
- Block → attribute syntax

---

### Example 3: With SCIM Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_access_identity_provider" "okta_scim" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Okta with SCIM"
  type       = "okta"

  config {
    client_id     = "okta-client-id"
    client_secret = var.okta_secret
    okta_account  = "https://example.okta.com"
  }

  scim_config {
    enabled                    = true
    group_member_deprovision   = true
    seat_deprovision           = true
    user_deprovision           = true
    secret                     = var.scim_secret
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_identity_provider" "okta_scim" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Okta with SCIM"
  type       = "okta"

  config = {
    client_id     = "okta-client-id"
    client_secret = var.okta_secret
    okta_account  = "https://example.okta.com"
  }

  scim_config = {
    enabled           = true
    seat_deprovision  = true
    user_deprovision  = true
    # secret is now computed-only, remove from config
    # group_member_deprovision removed (deprecated)
  }
}
```

**What Changed:**
- Both blocks → attributes
- `scim_config.secret` removed (now computed-only in v5)
- `scim_config.group_member_deprovision` removed (deprecated)

---

### Example 4: Minimal Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_access_identity_provider" "google" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Google Workspace"
  type       = "google"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_identity_provider" "google" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Google Workspace"
  type       = "google"

  config = {}
}
```

**What Changed:**
- Empty `config = {}` added if not present

---

