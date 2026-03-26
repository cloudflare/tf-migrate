# TKT-007: Provider — connection_rules state upgrade fails (expected "{", got "[")

## Status
Open — provider bug in cloudflare-terraform-next

## Error
```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  with cloudflare_zero_trust_access_policy.allow_cloudflare_com_emails
  AttributeName("connection_rules"): invalid JSON, expected "{", got "["
```

## Root cause
The v5 provider's UpgradeState handler for `cloudflare_zero_trust_access_policy`
expects `connection_rules` to be a JSON object `{}` in v4 state, but v4 stored
it as a JSON array `[]`. This is a schema mismatch in the state migration handler.

## Note
This error is partially caused by TKT-002 (include block not converted), which
leaves the schema in an inconsistent state. After re-running tf-migrate with the
TKT-002 fix, the config will be correct. However the state upgrade failure may
still occur if the v4 state has connection_rules as `[]`.

## Fix location
`internal/services/zero_trust_access_policy/migration/v500/handler.go` in
cloudflare-terraform-next — the UpgradeState handler needs to handle `[]` as
an empty/null connection_rules value.
