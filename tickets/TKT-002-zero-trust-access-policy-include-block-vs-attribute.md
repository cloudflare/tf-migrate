# TKT-002: zero_trust_access_policy — include/exclude/require should be block not attribute list

## Status
Open

## Reported by
Research team (terraform-cfaccounts MR !7756)

## Summary
In v4, `include`, `exclude`, and `require` in `cloudflare_zero_trust_access_policy`
(formerly `cloudflare_access_policy`) are HCL **blocks**. In v5 they should be
**attribute lists** using `include = [{ ... }]` syntax.

The research team found the migrator did NOT convert these correctly and had to
do it manually.

## v4 Original
```hcl
resource "cloudflare_access_policy" "allow_cloudflare_com_emails" {
  include {
    email_domain = ["cloudflare.com"]
  }
}
```

## After tf-migrate (broken — blocks not converted to attribute list)
```hcl
resource "cloudflare_zero_trust_access_policy" "allow_cloudflare_com_emails" {
  include {
    email_domain = ["cloudflare.com"]  # still a block, not an attribute
  }
}
```

## Expected after tf-migrate
```hcl
resource "cloudflare_zero_trust_access_policy" "allow_cloudflare_com_emails" {
  include = [{
    email_domain = { domain = "cloudflare.com" }
  }]
}
```

## Note
This ticket covers ONLY the `include = [{ ... }]` structural conversion.
The actual content transformations (email_domain, service_token, any_valid_service_token)
are tracked separately in TKT-004, TKT-005, TKT-006.

## Error from plan-error-1.log
```
Error: Unable to Read Previously Saved State for UpgradeResourceState
  AttributeName("connection_rules"): invalid JSON, expected "{", got "["
```

The state upgrade fails because the schema mismatch prevents state from being
read at all. The `include` block syntax is fundamentally incompatible with v5.

## Fix location
`internal/resources/zero_trust_access_policy/v4_to_v5.go`
The `TransformConfig` function needs to convert `include`/`exclude`/`require`
blocks to attribute list syntax.
