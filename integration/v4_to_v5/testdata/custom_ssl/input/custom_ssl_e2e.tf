# E2E override file for custom_ssl.
#
# Supplies a real self-signed cert/key pair (required by the Cloudflare API)
# and trims the for_each/count locals to keep concurrent creates within
# the API rate limit for certificate inserts.

variable "cert_pem" {
  type = string
  default = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIICKjCCAdACCQC6+rpIwJqjGjAKBggqhkjOPQQDAjAjMSEwHwYDVQQDDBhjb3J0
    LnRlcnJhZm9ybS5jZmFwaS5uZXQwHhcNMjYwMzA2MjM0MjQyWhcNMjcwMzA2MjM0
    MjQyWjAjMSEwHwYDVQQDDBhjb3J0LnRlcnJhZm9ybS5jZmFwaS5uZXQwggFLMIIB
    AwYHKoZIzj0CATCB9wIBATAsBgcqhkjOPQEBAiEA/////wAAAAEAAAAAAAAAAAAA
    AAD///////////////8wWwQg/////wAAAAEAAAAAAAAAAAAAAAD/////////////
    //wEIFrGNdiqOpPns+u9VXaYhrxlHQawzFOw9jvOPD4n0mBLAxUAxJ02CIbnBJNq
    ZnjhE50mt4GffpAEQQRrF9Hy4SxCR/i85uVjpEDydwN9gS3rM6D0oTlF2JjClk/j
    QuL+Gn+bjufrSnwPnhYrzjNXazFezsu2QGg3v1H1AiEA/////wAAAAD/////////
    /7zm+q2nF56E87nKwvxjJVECAQEDQgAEu+nqW7L4WI4RMUaYaZpmIuff6FNSc0GF
    AOLcXkNPAhrLB/hq7Crxu9y6HJHq2aIT+KHwd2OMKn4llWHcggq0qTAKBggqhkjO
    PQQDAgNIADBFAiEA4KZWMgwpCXOvusQ5h7SF9ZHJvWLzn/N+bJGLgSPY8lcCICIl
    JXpidot9bn4iDccBRIj9ZsdSvPdGa7nH2nrSzmx1
    -----END CERTIFICATE-----
  EOT
}

variable "key_pem" {
  type = string
  default = <<-EOT
    -----BEGIN PRIVATE KEY-----
    MIIBeQIBADCCAQMGByqGSM49AgEwgfcCAQEwLAYHKoZIzj0BAQIhAP////8AAAAB
    AAAAAAAAAAAAAAAA////////////////MFsEIP////8AAAABAAAAAAAAAAAAAAAA
    ///////////////8BCBaxjXYqjqT57PrvVV2mIa8ZR0GsMxTsPY7zjw+J9JgSwMV
    AMSdNgiG5wSTamZ44ROdJreBn36QBEEEaxfR8uEsQkf4vOblY6RA8ncDfYEt6zOg
    9KE5RdiYwpZP40Li/hp/m47n60p8D54WK84zV2sxXs7LtkBoN79R9QIhAP////8A
    AAAA//////////+85vqtpxeehPO5ysL8YyVRAgEBBG0wawIBAQQggriwkh31Gjoq
    D1cHvb4lllSpGqAyjXtlQi2yXvEG13+hRANCAAS76epbsvhYjhExRphpmmYi59/o
    U1JzQYUA4txeQ08CGssH+GrsKvG73LockerZohP4ofB3Y4wqfiWVYdyCCrSp
    -----END PRIVATE KEY-----
  EOT
}

# Trim for_each/count locals to limit concurrent API creates and
# stay within the certificate insert rate limit.
locals {
  geo_regions = ["us", "eu"]

  ssl_configs = {
    alpha = {
      bundle_method = "ubiquitous"
      type          = "legacy_custom"
      geo           = "us"
    }
    beta = {
      bundle_method = "force"
      type          = "legacy_custom"
      geo           = "eu"
    }
  }
}
