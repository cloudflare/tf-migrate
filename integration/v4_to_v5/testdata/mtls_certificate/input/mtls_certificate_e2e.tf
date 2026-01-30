variable "cloudflare_account_id" {
  type    = string
  default = "f037e56e89293a057740de681ac9abbe"
}

variable "cloudflare_zone_id" {
  type    = string
  default = "0da42c8d2132a9ddaf714f9e7c920711"
}

variable "cloudflare_domain" {
  type        = string
  description = "Domain for testing (not used by this module but accepted for consistency)"
}

locals {
  name_prefix = "cftftest"

  cert = <<EOT
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
}

resource "cloudflare_mtls_certificate" "basic_ca" {
  account_id   = var.cloudflare_account_id
  ca           = true
  certificates = local.cert
  name         = "${local.name_prefix}-basic-ca"
}
