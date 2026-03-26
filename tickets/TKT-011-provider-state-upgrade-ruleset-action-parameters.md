# TKT-011: Provider — cloudflare_ruleset state upgrade fails (action_parameters: expected "{", got "[")

## Status
Open — provider bug in cloudflare-terraform-next

## Error
```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  with cloudflare_ruleset.terraform_managed_resource_*
  AttributeName("rules").ElementKeyInt(0).AttributeName("action_parameters"):
  invalid JSON, expected "{", got "["
```

Affects both rulesets in the research team workspace.

## Root cause
The v5 provider's UpgradeState handler (or current schema) for `cloudflare_ruleset`
expects `action_parameters` to be a JSON object, but the state has it stored as
a JSON array `[]`.

## Fix location
`internal/services/ruleset/` in cloudflare-terraform-next — either the
UpgradeState handler or the current schema's state deserialization.
