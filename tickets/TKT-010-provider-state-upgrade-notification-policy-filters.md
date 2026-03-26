# TKT-010: Provider — cloudflare_notification_policy state upgrade fails (filters: expected "{", got "[")

## Status
Open — provider bug in cloudflare-terraform-next

## Error
```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  with cloudflare_notification_policy.expiring_service_token
  AttributeName("filters"): invalid JSON, expected "{", got "["
```

## Root cause
The v5 provider's UpgradeState handler for `cloudflare_notification_policy`
expects `filters` to be a JSON object, but v4 stored it as a JSON array `[]`.

## Fix location
`internal/services/notification_policy/migration/` in cloudflare-terraform-next
