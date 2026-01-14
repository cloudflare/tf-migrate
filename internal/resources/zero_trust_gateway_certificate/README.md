# Zero Trust Gateway Certificate Migration Guide (v4 → v5)

This guide explains how `cloudflare_zero_trust_gateway_certificate` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_zero_trust_gateway_certificate` | `cloudflare_zero_trust_gateway_certificate` | No change |
| Removed fields | `custom`, `gateway_managed`, `id`, `qs_pack_id` | - | Deprecated |
| Type conversion | `validity_period_days` int | int64 (float64 in state) | Numeric type |


---

## Migration Examples

### Example 1: Custom Certificate

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_gateway_certificate" "example" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  activate            = true
  validity_period_days = 365
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_gateway_certificate" "example" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  activate             = true
  validity_period_days = 365
}
```

**What Changed:**
- Configuration unchanged
- Deprecated fields removed from state if present

---

