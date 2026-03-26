# TKT-001: authenticated_origin_pulls — cert_id still references old resource type

## Status
Open

## Reported by
Research team (terraform-cfaccounts MR !7756)

## Summary
When `cloudflare_authenticated_origin_pulls_certificate` is renamed to
`cloudflare_authenticated_origin_pulls_hostname_certificate`, the `cert_id`
attribute in `cloudflare_authenticated_origin_pulls` resources still references
the OLD resource type name instead of the new one.

## v4 Original
```hcl
resource "cloudflare_authenticated_origin_pulls" "drand_aop" {
  for_each                               = local.endpoints_map
  zone_id                                = var.drand_zone_id
  authenticated_origin_pulls_certificate = cloudflare_authenticated_origin_pulls_certificate.drand_aop_cert[each.key].id
  hostname                               = "${each.value.node}-${each.value.suffix}.drand.cloudflare.com"
  enabled                                = true
}
```

## After tf-migrate (broken)
```hcl
resource "cloudflare_authenticated_origin_pulls" "drand_aop" {
  for_each = local.endpoints_map
  zone_id  = var.drand_zone_id
  cert_id  = cloudflare_authenticated_origin_pulls_certificate.drand_aop_cert[each.key].id
  # ↑ should be: cloudflare_authenticated_origin_pulls_hostname_certificate.drand_aop_cert[each.key].id
  hostname = "${each.value.node}-${each.value.suffix}.drand.cloudflare.com"
  enabled  = true
}
```

## Expected after tf-migrate
```hcl
resource "cloudflare_authenticated_origin_pulls" "drand_aop" {
  for_each = local.endpoints_map
  zone_id  = var.drand_zone_id
  cert_id  = cloudflare_authenticated_origin_pulls_hostname_certificate.drand_aop_cert[each.key].id
  hostname = "${each.value.node}-${each.value.suffix}.drand.cloudflare.com"
  enabled  = true
}
```

## Root Cause
The cross-file reference updater renames `cloudflare_authenticated_origin_pulls_certificate`
to `cloudflare_authenticated_origin_pulls_hostname_certificate` in resource blocks,
but does NOT update references to this resource type in attribute values of
OTHER resources (specifically the `cert_id` / `authenticated_origin_pulls_certificate`
attribute of `cloudflare_authenticated_origin_pulls`).

The research team had to manually fix this with sed:
```
sed -i '' -re 's/(cert_id *= *)cloudflare_authenticated_origin_pulls_certificate/\1cloudflare_authenticated_origin_pulls_hostname_certificate/' app_drand.tf app_kt_auditor_staging.tf
```

## Fix location
`internal/resources/authenticated_origin_pulls/` migrator — needs to update
`cert_id` attribute value references as part of cross-file postprocessing, OR
the global postprocessing needs to also rename type references inside attribute values.

## E2E Test
`integration/v4_to_v5/testdata/authenticated_origin_pulls/`
Add a test case with `cert_id` referencing `cloudflare_authenticated_origin_pulls_certificate`
and verify it is renamed to `cloudflare_authenticated_origin_pulls_hostname_certificate`.
