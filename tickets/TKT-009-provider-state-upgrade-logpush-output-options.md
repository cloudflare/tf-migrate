# TKT-009: Provider — cloudflare_logpush_job state upgrade fails (output_options: expected "{", got "[")

## Status
Open — provider bug in cloudflare-terraform-next

## Error
```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  with cloudflare_logpush_job.audit_logs_logpush_job
  AttributeName("output_options"): invalid JSON, expected "{", got "["
```

Affects all 3 logpush jobs in the research team workspace.

## Root cause
The v5 provider's UpgradeState handler for `cloudflare_logpush_job` expects
`output_options` to be a JSON object, but v4 stored it as a JSON array `[]`.

## Fix location
`internal/services/logpush_job/migration/` in cloudflare-terraform-next
