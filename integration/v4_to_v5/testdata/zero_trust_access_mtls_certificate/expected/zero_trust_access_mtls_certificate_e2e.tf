# E2E Test Configuration for zero_trust_access_mtls_certificate
# This file is used for end-to-end testing against the real Cloudflare API
# It contains minimal resources with unique certificates

variable "cloudflare_account_id" {
  type    = string
  default = "f037e56e89293a057740de681ac9abbe"
}

variable "cloudflare_zone_id" {
  type    = string
  default = "0da42c8d2132a9ddaf714f9e7c920711"
}

# Use static prefix for E2E tests to avoid timestamp-related plan drift
locals {
  name_prefix = "cftftest-e2e"

  # Three unique certificates for testing
  cert1 = <<EOT
-----BEGIN CERTIFICATE-----
MIIDvTCCAqWgAwIBAgIEaUWyljANBgkqhkiG9w0BAQsFADB0MQswCQYDVQQGEwJV
UzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAXBgNVBAoT
EFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRlc3QuZXhh
bXBsZS5jb20wHhcNMjUxMjE5MjAxNjIyWhcNMjYxMjE5MjAxNjIyWjB0MQswCQYD
VQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAX
BgNVBAoTEFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRl
c3QuZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDF
MxiaTyz/XpDiUdF07ObHkTQIlYjAW7phHnSzznvEWl2Tn3J77lNl7XQVeoPyIXDn
s1j93wj4Tf7UrgarBWxmDWD2nfgE6hROVhuYWhmNnNFqPIAV31KJZt4scE3sq1M+
dVttBTu5ItShdntt7QrE2E51lQobJME6yHIlaOQLiAaJTmsTR3ziNnSlR3y+/MDY
SqvLKEQYkLx5Y3GE2D34knWkgjDy7TL6N1bXutu/7clLRGUXsIpqVgv8VT2tkYzD
KsmRTO3S4tZlAXNjY40j4Y1zVfjDN9soiS5u0wLbNQq0hJF4p8FEZqiQTUR7gwDf
XwKjc3c2j9DoWdAOAK+HAgMBAAGjVzBVMA4GA1UdDwEB/wQEAwICpDATBgNVHSUE
DDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBRo5vWci6AQ
stsJGT/PyNfUglwsrzANBgkqhkiG9w0BAQsFAAOCAQEAfRid4vB6DhxkAdNMGn0I
aVpe1IP0cnsfrI4f8yShbwa7syEfpGbsdPmePmUnoTnXW/SZZ86ndT7uzYa2WIyE
PTQkcyScRVTGyU+Ze0T4S+tFm10QvjL7NmQrlKX/fDqho5RX+yP5li+adgBClowo
jIYvMgrg2SBKuLCR/JEragdGNBZTOkXT7vxA4ldzH70iBZLr/ODHVCFfCI+7mlaV
9jlPBN58LSZypTWBNraO1849fuYKVEGMafDVuFUGWwQQXE0zzUHyd1mNBQ/aRBmI
ULHVVSsA1PCbJ0Iq+Cnserb2j6EgZQqSlGo1D5mMPflWffjQQpgpbaq9xfNRt7gI
Ag==
-----END CERTIFICATE-----
EOT

  cert2 = <<EOT
-----BEGIN CERTIFICATE-----
MIIDvTCCAqWgAwIBAgIEaUWylzANBgkqhkiG9w0BAQsFADB0MQswCQYDVQQGEwJV
UzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAXBgNVBAoT
EFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRlc3QuZXhh
bXBsZS5jb20wHhcNMjUxMjE5MjAxNjIzWhcNMjYxMjE5MjAxNjIzWjB0MQswCQYD
VQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAX
BgNVBAoTEFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRl
c3QuZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCs
7VjWTxDB3qTrgAUFZ9lkI4jueQs4rO/spuRYrDqWjNXregux24YI6q4AbAG6m92Y
IIX7JzjHebHjX9RrxvkF6crhu0X5QrCgTgpZtfywBmURSPFq8VCGH0mQmsQFQW9K
XPX7056uoVsFCmXR8raF8pqg1XHtnmDKq5Dzj6HLwBzZR+oD3PG9tsQCITRloIGK
6cf5ndqv9AjdrqDOqWcIcAg9eD/G6eMzYqLNi/U8bY1BCqugqy8zlxyh9vVhGew1
0DxejH53QiHgdsRwJXo9Z2NO2laCLz+Pnoz+bGUY2UuwCL3e0poJ6+GHabEAhbYB
Uj+dgQIjbJLUFJXy5sSDAgMBAAGjVzBVMA4GA1UdDwEB/wQEAwICpDATBgNVHSUE
DDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBS8QTh7p+OK
0nz36FgxG1kjP2EKiDANBgkqhkiG9w0BAQsFAAOCAQEARAYVxpQtovXiptpEjaV3
F+lzdXCqV5lwZbkpdbdlys/KhKiBT9uVIJR1RUFdqB2jXPfeQy+MQkqvQyc0veM1
Sa4lyiN7vmRwnqMN9A0ZZnlN6JHZ93pFZ4ZvyBr4v4mkmHQLemYXHzaTohBPdE12
p/8Ck/T8mtcIYFxFjZC6xTdzt1EECllGYB0vZ1EFvceY+0Gevqy3ha8iB0c5KR1X
/llZVVCZN5vtOSwxnmxlct0OE7FNLuliijooXHqzVjCXxX7qpts5a6qJ8gg0RNoh
IIh3Hm55M4F2vvDlzAlOGxv6i2l1RvCXnlB7Jps2uphkfsO2xFv5YFKkaChYELLq
lg==
-----END CERTIFICATE-----
EOT

  cert3 = <<EOT
-----BEGIN CERTIFICATE-----
MIIDvTCCAqWgAwIBAgIEaUWymDANBgkqhkiG9w0BAQsFADB0MQswCQYDVQQGEwJV
UzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAXBgNVBAoT
EFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRlc3QuZXhh
bXBsZS5jb20wHhcNMjUxMjE5MjAxNjI0WhcNMjYxMjE5MjAxNjI0WjB0MQswCQYD
VQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDVNhbiBGcmFuY2lzY28xGTAX
BgNVBAoTEFRlc3QtSW50ZWdyYXRpb24xJTAjBgNVBAMTHGludGVncmF0aW9uLXRl
c3QuZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDG
3n+ATbARsi34DniDSKyX+UyqE8SaaSvbhAjdtzpzkbLMmWnKXtAGQsJUj+sXr+qq
vlirL1BX5Wl1TCnV3+M5LJbFSdspfs0xjGmduNmHiXw+BGq1Kn5Q3RUpgrH0j08M
WWBon2r+R5wLYEFWh7H2/+diSbGGRSGcayop8Dx4YpddaIHN91b+nbU9UUvOVKwH
hJr+uDrupQW9GqcXfQNJcq4Hj3THtH8Gmlo9UT8lDLCd0ZNK/8UsF1OTBvHd4u6a
0L2jVRyxP9UVW2lxd6K19Lp5VOdQAMiBGcB6kKZQLMizLxgCZ3fR7m6VJrivjBBr
PIyeZcw0UzftCZyBQFmDAgMBAAGjVzBVMA4GA1UdDwEB/wQEAwICpDATBgNVHSUE
DDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBREqIZCU4NC
rf6JGwrVlJS5AJLBZTANBgkqhkiG9w0BAQsFAAOCAQEAe4pT7qla+kcKnHT4DYyd
v0ifzQJhtTxNgS8Y7kFrC0/OAWIeYt/9lplwCZzl+X9ep7iIfZ3t+PlypBktb7gu
qnQaQxbi7UdgY0sJpepvilV08tA59o2pXMVWm0VIpGFYbDC5oTeSnp4riaQAPlUl
2FOVWGgpvG54KrEwrb6ha6gWuHpNESpsOR4iIj2LgSStGUbxZ2tYz4lkk5ZdOrlX
IGK2ayvoffXWLvSvFt6TAwhh8fjtp4QGtk3uPf2VoLO3PWjre3lP0MvsGrK32tYO
rvi1xUmOwefZCE4cEt9jFR2n/5Z7Ml7Q3upuAdfcJoKN2v2ZnxujiiWVE/Irz//T
Ew==
-----END CERTIFICATE-----
EOT
}

##########################
# MINIMAL E2E TEST RESOURCES
##########################

# 1. Basic account-scoped certificate
resource "cloudflare_zero_trust_access_mtls_certificate" "e2e_basic" {
  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-basic"
  certificate = local.cert1
}

# 2. Zone-scoped certificate
resource "cloudflare_zero_trust_access_mtls_certificate" "e2e_zone" {
  zone_id     = var.cloudflare_zone_id
  name        = "${local.name_prefix}-zone"
  certificate = local.cert2
}

# 3. for_each pattern with unique certificates
resource "cloudflare_zero_trust_access_mtls_certificate" "e2e_foreach" {
  for_each = {
    env1 = local.cert3
  }

  account_id  = var.cloudflare_account_id
  name        = "${local.name_prefix}-foreach-${each.key}"
  certificate = each.value
}
