# TKT-004: zero_trust_access_policy — service_token changed from list to object

## Status
Open

## Reported by
Research team (terraform-cfaccounts MR !7756)

## Summary
In v4, `service_token` inside include/exclude/require was a list of IDs.
In v5, it's an object with a `token_id` field.

## v4 Original
```hcl
include {
  service_token = [
    cloudflare_access_service_token.my_token.id,
  ]
}
```

## After tf-migrate (broken — list not converted)
```hcl
include {
  service_token = [
    cloudflare_access_service_token.my_token.id,
  ]
}
```

## Expected after tf-migrate
```hcl
include = [{
  service_token = {
    token_id = cloudflare_zero_trust_access_service_token.my_token.id
  }
}]
```

## Note
The research team confirmed the latest main branch of tf-migrate handled this
correctly: "the latest tagged version (v1.0.0-beta.5) produced some broken
results like clearing `service_token` in access_policies.tf, hence the use
of the latest main branch."

This suggests the fix may already be partially in place but needs verification
and e2e test coverage to prevent regression.

## Fix location
`internal/resources/zero_trust_access_policy/v4_to_v5.go`
The include/exclude/require content transformer.
