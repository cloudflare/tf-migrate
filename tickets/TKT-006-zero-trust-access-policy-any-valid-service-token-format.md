# TKT-006: zero_trust_access_policy — any_valid_service_token changed from bool to empty object

## Status
Open

## Reported by
Research team (terraform-cfaccounts MR !7756)

## Summary
In v4, `any_valid_service_token` inside include/exclude/require was a boolean.
In v5, it's an empty object `{}`.

## v4 Original
```hcl
include {
  any_valid_service_token = true
}
```

## After tf-migrate (broken — bool not converted)
```hcl
include {
  any_valid_service_token = true
}
```

## Expected after tf-migrate
```hcl
include = [{
  any_valid_service_token = {}
}]
```

## Note
When `any_valid_service_token = false` in v4, the field should simply be
omitted from the v5 config (not converted to an empty object).

## Fix location
`internal/resources/zero_trust_access_policy/v4_to_v5.go`
