# E2E test file for custom_ssl.
#
# Self-contained with real RSA cert/key (required by Cloudflare API) and a small
# resource set to stay within the certificate insert rate limit.
#
# Key migration paths covered:
#   - geo_restrictions string -> { label = "..." } nested attribute  (full)
#   - minimal config (no optional fields)                           (minimal)
#   - custom_ssl_priority blocks removed on migration               (with_priority)

variable "cloudflare_zone_id" {
  type = string
}

variable "cloudflare_account_id" {
  type = string
}

variable "cloudflare_domain" {
  type = string
}

locals {
  name_prefix = "cftftest"

  cert = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIIDDDCCAfSgAwIBAgIJAMOSfEVCsh4cMA0GCSqGSIb3DQEBCwUAMCMxITAfBgNV
    BAMMGGNvcnQudGVycmFmb3JtLmNmYXBpLm5ldDAeFw0yNjAzMDYyMzUyNDVaFw0y
    NzAzMDYyMzUyNDVaMCMxITAfBgNVBAMMGGNvcnQudGVycmFmb3JtLmNmYXBpLm5l
    dDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALpfS5ZnoXqD4UJp7mWN
    g9jVGK4jMiRH87+OVZ9LVdEo8sJWuRwWuZQQMnOozstvXst1SduXYQrI41twz8Hg
    cPrwb6v5J3rqIonKFxz9Dydte1Umkl8zV7uIKQ+3BnChsatkhVD6vZUt0gfRs4DQ
    RMtHBzGfIK/fdvMU2FFrF1xZ5o1WiiGakTpNaRuPLOcuXjXSsJLrqrZC5yUX2bjz
    IsRYf9tiDDtWMH89gU3y4TSnO98zoSi3c2iAqhogzY6Z7pyGUQwByFsIbfvrf+FC
    OhC2CtV6lSnHcZKymwARaebKeKD8Uav6XORuW9iFPDx2h3Nw+B4K8OzIjgzPN1/k
    rykCAwEAAaNDMEEwPwYDVR0RBDgwNoIYY29ydC50ZXJyYWZvcm0uY2ZhcGkubmV0
    ghoqLmNvcnQudGVycmFmb3JtLmNmYXBpLm5ldDANBgkqhkiG9w0BAQsFAAOCAQEA
    gcFwO2ID02ZJcbcJxsFnWtfYz0t05FpZaiuNvTPcdg+HgFrum8E2NCvU25J5imxG
    qrzsNp8zjinwS+zuxOlwxze9o6K6nTYvek83bhtKkxjocVkEhHgQuN8xavBU5Jd7
    nz2Mr+B5e+jAPUxQqvi6P+nUkIjnhWysWMG2uT13GJpXVo3nTB4sN/XoKZGBfRpU
    cBRSp/+swRmJMXkQAwIW7Lm1GbzriD3uCvgGFdVGrsxg3Zurg5wFcbrYavn6cKCO
    /4vr1adg/EAfIUwWWUBHZtA+AISJCEQQOLijgyHVWFzfkcjCXGHQgo67iFFYq+Un
    3GC53UxWk41y3QpuJ6ejTA==
    -----END CERTIFICATE-----
  EOT

  key = <<-EOT
    -----BEGIN PRIVATE KEY-----
    MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC6X0uWZ6F6g+FC
    ae5ljYPY1RiuIzIkR/O/jlWfS1XRKPLCVrkcFrmUEDJzqM7Lb17LdUnbl2EKyONb
    cM/B4HD68G+r+Sd66iKJyhcc/Q8nbXtVJpJfM1e7iCkPtwZwobGrZIVQ+r2VLdIH
    0bOA0ETLRwcxnyCv33bzFNhRaxdcWeaNVoohmpE6TWkbjyznLl410rCS66q2Qucl
    F9m48yLEWH/bYgw7VjB/PYFN8uE0pzvfM6Eot3NogKoaIM2Ome6chlEMAchbCG37
    63/hQjoQtgrVepUpx3GSspsAEWnmynig/FGr+lzkblvYhTw8dodzcPgeCvDsyI4M
    zzdf5K8pAgMBAAECggEALqtMR0Z+BireDn5uRxnPyU1bV8fSd4lY/T/MKw53V9/0
    IjwLMIB0SiJgL9w2pHSn/TTKoOVgVI4HeM9gBwGH6R6qKBtFCp90tKJZdVXdJJdi
    yejVwGcf8gLfnWLMhwnGbs/GHogbTy7hKDoXxArjHzATGhbp3YCMzcQLgx/ZArPG
    a2t9WcAvF0rVmMJS6eVlTOZ5ak9w6UC08qVTH8WhOY+cFws2d11HdQfpX1pqH1BN
    0o+VcbeVbUacQIavfqxKw3U09BEyczSGYhUvgBv4KSJxeWXT4e4wJD/PE9A84C0i
    TOS5EKE2Y23RVLMNF0miAnhC9FjCnr4+nB3sDcJqAQKBgQDymxP2bVxSxYGTB/zA
    9Bk+bsUTprlg2HUXTOaV5+mhYYtOMuiLhMuPy3itWtLOUPDFx1VAa0Y9nsJn/mgS
    WJDS4Zl5qHXVvQ/DT/KeK7uQqbbP6yCLlXx1aN0qXnZ5T4ZyYbZy/xAgveHtS8zo
    N8XgHheZok5sI4CuYR8ODR4KwQKBgQDEqW2qcbAvBWchAZZTnHhtGT/WeMeOAkfz
    g7vOSjtcXagGu/gRRz6ZfZI3r/0/red04OvvADjs9QfY84nu3s7CmYcMr/cbVy6h
    jWOCZI7w1HKjyr4qN33aalJul/m91wUW5CQ5CWIdgOl8QN7MoHQLtpdCrQjctix5
    0TY0F4HGaQKBgQDhF62P6KvOSF4Ok0yZomGBobjMoNZC2tLZCYqv73q/NwfPSECm
    olFUW07eWPRaZJLgji+1E1MafSCW6F6bFv1YC+UgEYMzCrWDW7wZsS3X7P8nLlsF
    526QaPk7BGYb7AMsQSjMzYajOkpSpw+5LXY0mPcAnqzwfIg6QvZTTSxggQKBgHal
    zW7+hg/oT47fOUWaaiFQEW6gkayAfc5R1NWhfWy9aGkfsIskE4Vg9/025TAtCC5A
    oLchyDZVonVmgPonXFCVdZ/W7duF3rFC7x008/QiCEP/RnmL3xcN/EuSzu6UshJc
    c+ohWht4seTv8js8Nqb2cw2b/XPDSNP5v5zv7bC5AoGBAMvUobc0NFq2w1fBZJrf
    Bj0omv2VhjAmGx4wFOEB2zekEW+X5gXzybrmvPdJSIEIfMBwaotjozMExJhxlCmk
    2An0KHBJNFXXxm3SEddQz9NdYXVe64Eg41Eay75/trHVjm3DRsi+fwQYzim5gbJc
    wYguhpa8ijCldZ+RxUP8vRNt
    -----END PRIVATE KEY-----
  EOT
}

# Pattern 1: Full config - exercises geo_restrictions string -> { label } transform
resource "cloudflare_custom_ssl" "full" {
  zone_id = var.cloudflare_zone_id

  custom_ssl_options {
    certificate      = local.cert
    private_key      = local.key
    bundle_method    = "force"
    type             = "legacy_custom"
    geo_restrictions = "us"
  }
}

# Pattern 2: Minimal config - no geo_restrictions or priority blocks
resource "cloudflare_custom_ssl" "minimal" {
  zone_id = var.cloudflare_zone_id

  custom_ssl_options {
    certificate   = local.cert
    private_key   = local.key
    bundle_method = "force"
    type          = "legacy_custom"
  }
}

# Pattern 3: custom_ssl_priority blocks - must be removed on migration
resource "cloudflare_custom_ssl" "with_priority" {
  zone_id = var.cloudflare_zone_id

  custom_ssl_options {
    certificate   = local.cert
    private_key   = local.key
    bundle_method = "force"
    type          = "legacy_custom"
  }

  custom_ssl_priority {
    id       = "cert-001"
    priority = 1
  }

  custom_ssl_priority {
    id       = "cert-002"
    priority = 2
  }
}
