# TKT-003: zero_trust_access_policy — application_id and precedence must move to access_application

## Status
Open

## Reported by
Research team (terraform-cfaccounts MR !7756)

## Summary
In v4, `cloudflare_access_policy` had `application_id` and `precedence` attributes
that linked a policy to an application. In v5, policies are account-level resources
and this binding is done via the `policies` block on `cloudflare_zero_trust_access_application`.

The migrator leaves `application_id` in the policy resource, which fails state
upgrade with: `unsupported attribute "application_id"`.

## v4 Original
```hcl
resource "cloudflare_access_policy" "allow_users" {
  application_id = cloudflare_access_application.my_app.id
  zone_id        = var.zone_id
  precedence     = 1
  name           = "allow users"
  decision       = "allow"

  include {
    email_domain = ["cloudflare.com"]
  }
}
```

## After tf-migrate (broken)
```hcl
resource "cloudflare_zero_trust_access_policy" "allow_users" {
  application_id = cloudflare_zero_trust_access_application.my_app.id  # ← should be removed
  account_id     = var.account_id
  precedence     = 1                                                     # ← should be removed
  name           = "allow users"
  decision       = "allow"

  include = [{
    email_domain = { domain = "cloudflare.com" }
  }]
}
```

## Expected after tf-migrate
```hcl
# Policy becomes account-level (no application_id or precedence)
resource "cloudflare_zero_trust_access_policy" "allow_users" {
  account_id = var.account_id
  name       = "allow users"
  decision   = "allow"

  include = [{
    email_domain = { domain = "cloudflare.com" }
  }]
}

# Application gets the binding
resource "cloudflare_zero_trust_access_application" "my_app" {
  # ... existing fields ...
  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.allow_users.id
      precedence = 1
    }
  ]
}
```

## Error from plan-error-1.log
```
Warning: Failed to decode resource from state
Error decoding "cloudflare_zero_trust_access_policy.allow_any_service_token"
from prior state: unsupported attribute "application_id"

Error: Unable to Read Previously Saved State for UpgradeResourceState
AttributeName("connection_rules"): invalid JSON, expected "{", got "["
```

## Complexity
This is a complex cross-resource transformation. The migrator needs to:
1. Remove `application_id` and `precedence` from the policy
2. Add `policies = [{ id = ..., precedence = ... }]` to the linked application

If the application is in a different file, cross-file modification is needed.
If the application_id is a literal string (not a reference), it may not be possible
to automatically find the application resource.

## Fix location
`internal/resources/zero_trust_access_policy/v4_to_v5.go`
