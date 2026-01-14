# Notification Policy Webhooks Migration Guide (v4 → v5)

This guide explains how `cloudflare_notification_policy_webhooks` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_notification_policy_webhooks` | `cloudflare_notification_policy_webhooks` | No change |
| All fields | Unchanged | Unchanged | No change |
| Schema | `url` optional | `url` required | Validation tightened |


---

## Migration Overview

**This is a version-bump-only migration.** No field or state transformations occur. The v5 schema makes the `url` field explicitly required (it was always required by the API).

---

## Migration Example

**v4 Configuration:**
```hcl
resource "cloudflare_notification_policy_webhooks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Webhook Integration"
  url        = "https://hooks.example.com/webhook"
  secret     = "my-webhook-secret"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_notification_policy_webhooks" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Webhook Integration"
  url        = "https://hooks.example.com/webhook"
  secret     = "my-webhook-secret"
}
```

**What Changed:** Nothing

---

## Field Reference

All fields unchanged:
- `account_id` - Account identifier (required)
- `name` - Webhook name (required)
- `url` - Webhook URL (required - was always required by API, now explicit in schema)
- `secret` - Webhook secret (optional)

---

## Testing Your Migration

After migration, verify:

1. **Zero drift:**
   ```bash
   terraform plan
   # Should show: No changes. Your infrastructure matches the configuration.
   ```

---

## Additional Resources

- Integration tests: `integration/v4_to_v5/testdata/notification_policy_webhooks/`
- Migration code: `internal/resources/notification_policy_webhooks/v4_to_v5.go`

---


**Complexity Rating: ⭐ LOW**

No transformations - schema version update only. The `url` field was always required by the API; v5 makes this explicit in the schema.
