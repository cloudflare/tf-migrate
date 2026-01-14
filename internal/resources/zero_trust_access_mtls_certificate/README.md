# Zero Trust Access mTLS Certificate Migration Guide (v4 → v5)

This guide explains how `cloudflare_access_mutual_tls_certificate` / `cloudflare_zero_trust_access_mtls_certificate` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_access_mutual_tls_certificate` | `cloudflare_zero_trust_access_mtls_certificate` | Renamed |
| Alt resource name | `cloudflare_zero_trust_access_mtls_certificate` | `cloudflare_zero_trust_access_mtls_certificate` | No change |
| Fields | All preserved | All preserved | No change |
| State default | - | `associated_hostnames = []` | Added if missing |


---

## Migration Examples

### Example 1: Basic mTLS Certificate

**v4 Configuration:**
```hcl
resource "cloudflare_access_mutual_tls_certificate" "example" {
  account_id              = "f037e56e89293a057740de681ac9abbe"
  name                    = "Client Certificate"
  certificate             = file("client-cert.pem")
  associated_hostnames    = ["example.com", "app.example.com"]
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_mtls_certificate" "example" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  name                 = "Client Certificate"
  certificate          = file("client-cert.pem")
  associated_hostnames = ["example.com", "app.example.com"]
}
```

**What Changed:**
- Resource type: `cloudflare_access_mutual_tls_certificate` → `cloudflare_zero_trust_access_mtls_certificate`

---

### Example 2: Zone-Scoped Certificate

**v4 Configuration:**
```hcl
resource "cloudflare_access_mutual_tls_certificate" "zone_cert" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "Zone mTLS Cert"
  certificate = file("zone-cert.pem")
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_access_mtls_certificate" "zone_cert" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "Zone mTLS Cert"
  certificate = file("zone-cert.pem")
}
```
