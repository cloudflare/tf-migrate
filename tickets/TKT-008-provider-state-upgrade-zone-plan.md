# TKT-008: Provider — cloudflare_zone state upgrade fails (plan: expected "{", got "enterprise")

## Status
Open — provider bug in cloudflare-terraform-next

## Error
```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  with cloudflare_zone.req_mtls
  AttributeName("plan"): invalid JSON, expected "{", got "enterprise"
```

## Root cause
The v5 provider's UpgradeState handler for `cloudflare_zone` expects the `plan`
attribute to be a JSON object, but v4 stored it as a plain string `"enterprise"`.

## Fix location
`internal/services/zone/migration/` in cloudflare-terraform-next — the
UpgradeState handler needs to handle plain string values for `plan` and convert
them to the v5 object format.
