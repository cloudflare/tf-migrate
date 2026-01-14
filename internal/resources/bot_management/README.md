# Bot Management Migration Guide (v4 → v5)

This guide explains how `cloudflare_bot_management` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_bot_management` | `cloudflare_bot_management` | No change |
| All fields | Unchanged | Unchanged | No change |
| Schema | v4 SDK | v5 Plugin Framework | Internal only |


---

## Migration Overview

**This is a version-bump-only migration.** The bot_management resource schema is 100% backward compatible between v4 and v5. No field names, types, or semantics have changed.

The migration exists purely for:
- Framework compatibility (SDK → Plugin Framework)
- Future extensibility
- Consistency with other resource migrations

---

## Migration Examples

### Example 1: Basic Bot Management Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_bot_management" "example" {
  zone_id           = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js         = true
  fight_mode        = false
  auto_update_model = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_bot_management" "example" {
  zone_id           = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js         = true
  fight_mode        = false
  auto_update_model = true
}
```

**What Changed:** Nothing - configuration is identical

---

### Example 2: Super Bot Fight Mode (SBFM) Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_bot_management" "sbfm" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = true
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_bot_management" "sbfm" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true
  optimize_wordpress              = true
}
```

**What Changed:** Nothing - configuration is identical

---

### Example 3: AI Bots Protection

**v4 Configuration:**
```hcl
resource "cloudflare_bot_management" "ai_protection" {
  zone_id            = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js          = true
  ai_bots_protection = "block"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_bot_management" "ai_protection" {
  zone_id            = "0da42c8d2132a9ddaf714f9e7c920711"
  enable_js          = true
  ai_bots_protection = "block"
}
```

**What Changed:** Nothing - configuration is identical

---

### Example 4: Comprehensive Configuration

**v4 Configuration:**
```hcl
resource "cloudflare_bot_management" "full" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"

  # Core settings
  enable_js                       = true
  fight_mode                      = false
  auto_update_model               = true

  # Super Bot Fight Mode
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true

  # Additional features
  optimize_wordpress              = true
  ai_bots_protection              = "block"
  suppress_session_score          = false
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_bot_management" "full" {
  zone_id                         = "0da42c8d2132a9ddaf714f9e7c920711"

  # Core settings
  enable_js                       = true
  fight_mode                      = false
  auto_update_model               = true

  # Super Bot Fight Mode
  sbfm_definitely_automated       = "block"
  sbfm_likely_automated           = "managed_challenge"
  sbfm_verified_bots              = "allow"
  sbfm_static_resource_protection = true

  # Additional features
  optimize_wordpress              = true
  ai_bots_protection              = "block"
  suppress_session_score          = false
}
```

**What Changed:** Nothing - configuration is identical

---

