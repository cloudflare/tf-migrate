# Notification Policy Migration Guide (v4 → v5)

This guide explains how `cloudflare_notification_policy` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_notification_policy` | `cloudflare_notification_policy` | No change |
| `filters` | Block (MaxItems:1) | Attribute object | Syntax change |
| `email_integration` | Multiple blocks | `mechanisms.email` array | Restructured |
| `webhooks_integration` | Multiple blocks | `mechanisms.webhooks` array | Restructured |
| `pagerduty_integration` | Multiple blocks | `mechanisms.pagerduty` array | Restructured |
| Integration `name` field | Included | Removed | Field dropped |
| Deprecated alert types | `weekly_account_overview`, `workers_alert` | Not supported | Validation warning |


---

## Migration Examples

### Example 1: Email Integration Only

**v4 Configuration:**
```hcl
resource "cloudflare_notification_policy" "email_only" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Email alerts for zone errors"
  description = "Alert via email on zone-level errors"
  enabled     = true
  alert_type  = "universal_ssl_event_type"

  email_integration {
    id   = "test@example.com"
    name = "Test Email"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_notification_policy" "email_only" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Email alerts for zone errors"
  description = "Alert via email on zone-level errors"
  enabled     = true
  alert_type  = "universal_ssl_event_type"

  mechanisms = {
    email = [
      { id = "test@example.com" }
    ]
  }
}
```

**What Changed:**
- `email_integration` blocks → `mechanisms.email` array
- `name` field removed (only `id` retained)

---

### Example 2: Webhook Integration

**v4 Configuration:**
```hcl
resource "cloudflare_notification_policy" "webhook" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Webhook alerts"
  description = "Send alerts to webhook"
  enabled     = true
  alert_type  = "load_balancing_pool_enablement_alert"

  webhooks_integration {
    id   = "webhook-id-123"
    name = "Production Webhook"
  }

  webhooks_integration {
    id   = "webhook-id-456"
    name = "Backup Webhook"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_notification_policy" "webhook" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Webhook alerts"
  description = "Send alerts to webhook"
  enabled     = true
  alert_type  = "load_balancing_pool_enablement_alert"

  mechanisms = {
    webhooks = [
      { id = "webhook-id-123" },
      { id = "webhook-id-456" }
    ]
  }
}
```

**What Changed:**
- Multiple `webhooks_integration` blocks → `mechanisms.webhooks` array
- `name` fields removed

---

### Example 3: Multiple Integration Types

**v4 Configuration:**
```hcl
resource "cloudflare_notification_policy" "multi" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Multi-channel alerts"
  description = "Alert via multiple channels"
  enabled     = true
  alert_type  = "health_check_status_notification"

  email_integration {
    id   = "admin@example.com"
    name = "Admin Email"
  }

  email_integration {
    id   = "ops@example.com"
    name = "Ops Email"
  }

  webhooks_integration {
    id   = "webhook-id-789"
    name = "Slack Webhook"
  }

  pagerduty_integration {
    id   = "pagerduty-id-abc"
    name = "On-call Team"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_notification_policy" "multi" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Multi-channel alerts"
  description = "Alert via multiple channels"
  enabled     = true
  alert_type  = "health_check_status_notification"

  mechanisms = {
    email = [
      { id = "admin@example.com" },
      { id = "ops@example.com" }
    ]
    webhooks = [
      { id = "webhook-id-789" }
    ]
    pagerduty = [
      { id = "pagerduty-id-abc" }
    ]
  }
}
```

**What Changed:**
- All integration blocks → unified `mechanisms` object
- Each integration type becomes an array within mechanisms
- All `name` fields removed

---

### Example 4: With Filters

**v4 Configuration:**
```hcl
resource "cloudflare_notification_policy" "filtered" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Filtered zone alerts"
  description = "Alert for specific zones"
  enabled     = true
  alert_type  = "zone_aop_custom_certificate_expiration_type"

  filters {
    zones = [
      "zone-id-1",
      "zone-id-2"
    ]
    slo = ["99.9"]
  }

  email_integration {
    id   = "alerts@example.com"
    name = "Alert Email"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_notification_policy" "filtered" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Filtered zone alerts"
  description = "Alert for specific zones"
  enabled     = true
  alert_type  = "zone_aop_custom_certificate_expiration_type"

  filters = {
    zones = [
      "zone-id-1",
      "zone-id-2"
    ]
    slo = ["99.9"]
  }

  mechanisms = {
    email = [
      { id = "alerts@example.com" }
    ]
  }
}
```

**What Changed:**
- `filters { }` block → `filters = { }` attribute
- Integration restructuring applied

---

### Example 5: Deprecated Alert Type (Warning)

**v4 Configuration:**
```hcl
resource "cloudflare_notification_policy" "deprecated" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Weekly overview"
  enabled     = true
  alert_type  = "weekly_account_overview"  # ⚠️ Deprecated

  email_integration {
    id   = "reports@example.com"
    name = "Reports"
  }
}
```

**Migration Warning:**
```
Warning: Deprecated alert type
│
│   on main.tf line 5:
│    5:   alert_type  = "weekly_account_overview"
│
│ Alert type "weekly_account_overview" is no longer supported in v5.
│ This resource may fail to apply with the v5 provider.
```

**What Changed:**
- Migration completes but adds warning diagnostic
- Resource may fail when applied with v5 provider

---

