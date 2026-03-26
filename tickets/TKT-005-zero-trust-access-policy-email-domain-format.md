# TKT-005: zero_trust_access_policy — email_domain changed from list to object

## Status
Open

## Reported by
Research team (terraform-cfaccounts MR !7756)

## Summary
In v4, `email_domain` inside include/exclude/require was a list of domain strings.
In v5, it's an object with a `domain` field.

## v4 Original
```hcl
include {
  email_domain = ["cloudflare.com"]
}
```

## After tf-migrate (broken — list not converted)
```hcl
include {
  email_domain = ["cloudflare.com"]
}
```

## Expected after tf-migrate
```hcl
include = [{
  email_domain = { domain = "cloudflare.com" }
}]
```

## Multiple domains
v4 allowed multiple domains in the list. In v5 each domain needs its own
include block:
```hcl
# v4
include {
  email_domain = ["cloudflare.com", "example.com"]
}

# v5
include = [
  { email_domain = { domain = "cloudflare.com" } },
  { email_domain = { domain = "example.com" } },
]
```

## Fix location
`internal/resources/zero_trust_access_policy/v4_to_v5.go`
