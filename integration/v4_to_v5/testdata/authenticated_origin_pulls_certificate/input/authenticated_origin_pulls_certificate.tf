# v4 configuration with both per-zone and per-hostname certificates

variable "cloudflare_zone_id" {
  type = string
}

# Per-zone certificate - will migrate to cloudflare_authenticated_origin_pulls_certificate
resource "cloudflare_authenticated_origin_pulls_certificate" "per_zone_1" {
  zone_id     = var.cloudflare_zone_id
  certificate = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIIDxzCCAq+gAwIBAgIUOhbAEzf7if9lHC0qsFUxWftvZxYwDQYJKoZIhvcNAQEL
    BQAwXTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMRYwFAYDVQQHDA1TYW4gRnJh
    bmNpc2NvMQ0wCwYDVQQKDARUZXN0MRowGAYDVQQDDBF6b25lMS5leGFtcGxlLmNv
    bTAeFw0yNjAyMTEyMzAwNDlaFw0yNzAyMTEyMzAwNDlaMF0xCzAJBgNVBAYTAlVT
    MQswCQYDVQQIDAJDQTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwE
    VGVzdDEaMBgGA1UEAwwRem9uZTEuZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEB
    AQUAA4IBDwAwggEKAoIBAQCwqFhMZimGfQgig2uRKzxoM8xON0aQUUjYmX/4l5p6
    Yg45FnZHwO9KCAojClAE9Cy+tAVdYlUvfanQgMUQuzy6mBpBRXeS3QnGGtbGm5TX
    6AIKL513UdydEucAel1VoXHZb+4dXmrOCE53Hj4KJ9cJA8PG68uqGMSrTLY61eF3
    acqlvyUd+2wXOD4fZaY2/KMk3xMVdvLJaTW21n4NZCRin3RTY/hD22YMckJK+WL9
    8/21kUXUt1/AZA/VzOchUkVpZNRgPsuBt+xJoqzqKN7dfBy517scGbQo0GyUwzOX
    h0dvsiwsFLtETEhuGNZCj44xQqUPnRAeLWodGBVWh/DHAgMBAAGjfzB9MAwGA1Ud
    EwEB/wQCMAAwDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMCBggr
    BgEFBQcDATAdBgNVHQ4EFgQUU0cbmyKjM7depvi+ZuasVxJ8pe4wHwYDVR0jBBgw
    FoAUU0cbmyKjM7depvi+ZuasVxJ8pe4wDQYJKoZIhvcNAQELBQADggEBAJ5+VM4U
    aETiw3zfiigZYCxeybW4QigDxvPgSOOiz8zPJuNbpLQAgbmTK1JsbU1eAs0AZvXz
    AnpX2OnLR3Gf2eiP5YMq5rhfdAjjA83oJh9nd4pFEgzLJGsp4BlEmtgMK7h7HFhf
    WOEhXN3cCBOLkzu5Z/JcNvs338gICAM2LpKJrlYWvyynZWdY7xrrqBSi4wsz5vVU
    x4ezirXchNLctv+wjoyY0ogGr8vAiWpQjZpsowiYCuR61CXBOUlDs48kPOvfG7sx
    tjQFMlZkoIFZy+wse67pzF3uPdQmOvCKpwI32vGUEPzYJrVvd8W87n8rQaUWuFjb
    hTchNNmoJJCg2NU=
    -----END CERTIFICATE-----
  EOT
  private_key = <<-EOT
    -----BEGIN PRIVATE KEY-----
    MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCwqFhMZimGfQgi
    g2uRKzxoM8xON0aQUUjYmX/4l5p6Yg45FnZHwO9KCAojClAE9Cy+tAVdYlUvfanQ
    gMUQuzy6mBpBRXeS3QnGGtbGm5TX6AIKL513UdydEucAel1VoXHZb+4dXmrOCE53
    Hj4KJ9cJA8PG68uqGMSrTLY61eF3acqlvyUd+2wXOD4fZaY2/KMk3xMVdvLJaTW2
    1n4NZCRin3RTY/hD22YMckJK+WL98/21kUXUt1/AZA/VzOchUkVpZNRgPsuBt+xJ
    oqzqKN7dfBy517scGbQo0GyUwzOXh0dvsiwsFLtETEhuGNZCj44xQqUPnRAeLWod
    GBVWh/DHAgMBAAECggEAKgM/TJwXUBa4Mo0SremcaiO3ePqIW5YZPvnyh0p2wJhF
    Tapb4uCth+u1jXPMaAEyCwCBLh5OqAa4tg+JzlrZLH8z70X4FANhaa3EWmNx2I8i
    vQ1p45CiaPCv41s2i0Dj9JQ8CtwDhpBPKOEWXA/xggFVNB+rxf4x95M82202O9GV
    is1PlolZlpjg1UB42rNJxd/ZBx0QEbyZvNzIw/G7n07X9DKE8fw/FZ4YSGhSJqq8
    HxgdDmCXpgkM4rjGFPGRY+U8xvtHBlLEbTX2pUM6ZAIHAPuGvtidS93BIzvdsvGR
    IM+ua9gyyM5jY3dZsZBsamBMn9e3I2nX7BShgvWM4QKBgQDn2JdyR6iOdUNn54C1
    ZwTrdGZXd1U6hHMCdjCPrIPZ7vevzJnHunojJkDv27IgDgDxI3oDIpsK5T1XbtzT
    dkdCK8kAAXUI50EUd0v6NvCqNNIygXChul9hjRqxkmtVdYsvM7CeP9lVAePnnDd/
    1X0xClFUgj52/xiY+gDrQR9+zwKBgQDDD9wBe4szkbAfvAdEZXczWbiUFVfFfqsM
    TR9iOndgBGJ/yYiA8fRpZtBZuBjNaqmVZMlvuZsynbXVJktrYdvmeQ6QRMUg0t7/
    gTnCcqvgUAcPlkeAfeQKMu8aW0BuKqtQuuzWsXxunqGZhXWBP1e15lCg/+EHgv6v
    3Hyco2asiQKBgH9Z8v6cLBNsiEUn3gRG/WXUf27mJtPI81/TyiLxcU+huz4+1e3n
    GbX7Ckp21GZVKuFKSng0ZxPaDhLb28LwQn4vjO5K3p2wYYg7a2mbCiGEeD2z6kl8
    FW6BUrtdoUXFFlosO4UBr4DJVAXiQn4ep/DrKPeRv3wf7cQB98VB9WnzAoGBAJep
    4RmWAWmbQSGrhMr9SW03uXgKEDCSiFQMMvahFuglAKDzBZuchLjfI+heZ4pwAGMT
    9jtUSQNV9GdCWymm8N+GCHjLv6oByzlGNK6nklPaZWMNKZMSTxhO+fG4OaRusL0Y
    WcWkQmeQF33ScsaHhZ788Hv99+1rQLNj78+qjM5hAoGBAKVICj/Sbo7twdbG99rB
    XKIBYJCmMBNCWTyQvE1e9Uh3yo0UTfkzH1DHhCRtmAtBPo7ZujZQhlnz3x/xdXmi
    jF+u83OrQe5iNTzm883zsp0AwfEYw0BhBVvFM47clZqmsY9tcfXGGuN/si/2o9x0
    Zfgmf6fJYEk/3o9TauQL6mrm
    -----END PRIVATE KEY-----
  EOT
  type        = "per-zone"
}

# Per-hostname certificate - will migrate to cloudflare_authenticated_origin_pulls_hostname_certificate
resource "cloudflare_authenticated_origin_pulls_certificate" "per_hostname_1" {
  zone_id     = var.cloudflare_zone_id
  certificate = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIIDzzCCAregAwIBAgIUAwo1wahcI52qxAQfKAt1PbxZyegwDQYJKoZIhvcNAQEL
    BQAwYTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMRYwFAYDVQQHDA1TYW4gRnJh
    bmNpc2NvMQ0wCwYDVQQKDARUZXN0MR4wHAYDVQQDDBVob3N0bmFtZTEuZXhhbXBs
    ZS5jb20wHhcNMjYwMjExMjI1MDU0WhcNMjcwMjExMjI1MDU0WjBhMQswCQYDVQQG
    EwJVUzELMAkGA1UECAwCQ0ExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDTALBgNV
    BAoMBFRlc3QxHjAcBgNVBAMMFWhvc3RuYW1lMS5leGFtcGxlLmNvbTCCASIwDQYJ
    KoZIhvcNAQEBBQADggEPADCCAQoCggEBALsXqqrB7RfX/SjdS/qClLivTEuyVio7
    Nu2GoORc4v0CFn5oiMqXztgnRv0AjUfmoOb+vXatPalE4YE/OnekPn9RqXuGgaZT
    lca0E9/I9Lx/2+6JsJU5XXRop8eFvev4o99xN611DybI2SgWg/cljGOk8j4x2rpj
    v5vzQTI92msbTiXdh/3jYIvqI8mzDKnkAK6jHcy/LWx4WhFgt1ugWJ9xRFFroddP
    upjhKJHTrT1GVeaO59IKMwBwjAaRtwy8zQoeioDZNZ4zUXz/abcuLOdyRndhWUci
    MATueYoRxdmIzWN7gZtIejC68RfuXoHssf8LWSQUkLzgnmTBSC99Qm8CAwEAAaN/
    MH0wDAYDVR0TAQH/BAIwADAOBgNVHQ8BAf8EBAMCBaAwHQYDVR0lBBYwFAYIKwYB
    BQUHAwIGCCsGAQUFBwMBMB0GA1UdDgQWBBQfznVNtzb3KblWwBzHkwI0gkRkKjAf
    BgNVHSMEGDAWgBQfznVNtzb3KblWwBzHkwI0gkRkKjANBgkqhkiG9w0BAQsFAAOC
    AQEAEODKbnurjns8+FPxfDMU0JoiRAlDR4lrlGh+CpkA3AGxaz/TLYelGZaj5bxn
    mjEfEfh6IKZhG3DtOJkFR/O56efApFOx25ctXR7Q/xRoflxxdw+aOlzmLe/cq1ug
    iujb0lupfmImboFYk/ff5ksD0oYsBJUH/JGwAbiJ2/ZA9ZuXmh8UD+pMa57C88Ov
    DaggDzelMywt2tNo4b6ukEozhhnnsU9FRvP8nPBB+ay+/+J1QATc+c+yrQYcgrgl
    yXj85N81RsNmrehIJBm4wKUuXT72RLhiPH/g/RAqVTgkS0KZitopHAOf8TyJSyCo
    DwHWLIjKek0vRjN7xpp+CGA7RA==
    -----END CERTIFICATE-----
  EOT
  private_key = <<-EOT
    -----BEGIN PRIVATE KEY-----
    MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7F6qqwe0X1/0o
    3Uv6gpS4r0xLslYqOzbthqDkXOL9AhZ+aIjKl87YJ0b9AI1H5qDm/r12rT2pROGB
    Pzp3pD5/Ual7hoGmU5XGtBPfyPS8f9vuibCVOV10aKfHhb3r+KPfcTetdQ8myNko
    FoP3JYxjpPI+Mdq6Y7+b80EyPdprG04l3Yf942CL6iPJswyp5ACuox3Mvy1seFoR
    YLdboFifcURRa6HXT7qY4SiR0609RlXmjufSCjMAcIwGkbcMvM0KHoqA2TWeM1F8
    /2m3LiznckZ3YVlHIjAE7nmKEcXZiM1je4GbSHowuvEX7l6B7LH/C1kkFJC84J5k
    wUgvfUJvAgMBAAECggEAApAhCWfQsxPrzVFPk7+RzZnkX+EYjXzA+uE7EQXPIj75
    RrZxF7CbgWbzi2gkVCJuFkJrhaJ7IF3nzPJ4/yyXCP16AE3OahRMQZJnKn8OEAsC
    w0v/L3wl5WaNoQ3m+48tB4f9U3kxqCSP1pzzjKfNOlkHH7nLg+QoutH+FNpRRK68
    Es3gHdfJWHy8u+aaC1TcqA8RaR3UrZapIlGByuE38B3BpdHqBpsjd/pRlti9VWXK
    XQdh7hSBS4tuYp9I0N0FIT5b1ObYIGm5Kf2hOXwhvLvvWQoUIxe5stWwjpBxvhe9
    01RXhAQ7lS6UyaFFb30wDPcftQy2TAAM/G1zzie+AQKBgQDmloKSxg+KFvZD5xTS
    zkZsd1gYd0+AuGB83hMq/+UgZP9/TSgW+z8dx3BigViRGR9QghKwYJmRRGBJ46pj
    rvFZUe/TP8/YvYKCVjb215jO41EMzMqFNpn4A2a7fQSTUd+ZS8AZaJJgM0WqUKNg
    fwEK2WNS58DaEgEFJvhk3Z3nbwKBgQDPtgj5SlURLMnIF/U1tr63QQJnbUMjFrZs
    vx/k3rjgXcZo7DVxgTqfAlE/FppskascFZLPv3HWISN/Msttjqhhri1h+sGOXKfr
    WRVuvBE59Jj2GFxwKMBWTKPNfYLQ+PPDRT/8JC8PCVSNCL6gfxCCqZHvCF5koPcn
    u5QDbLDVAQKBgQDiCeDt6GILR/8ZCUmMbND0OvmM4kh5MkTDox6/JCKD4v3i2MvX
    22s/0eYFai5b7niX/yo65DcmBBUv2ZGKLlBA8uVZ/E/Pc9af1cwDpc0R4hvtpENS
    2veL/CmU2TTHBZdfOraRMcVrsFc2Yd4GFfn7nKaU+sI+AzAk0NLmbakA2QKBgATB
    2ZDEGBCtou13RwF07wdJcOGnifsawRDai8N1KmzRGQM8LbksyYfsyKmWPfEwoOei
    wtsJOnU6CxMVub0HoGmkUJvG33oAO0RTpP8FRau7I2m3gx56gHU5iiLhtgZNPWAC
    jQWcWouQniQgyCTq5BjqA1KjMW5ClYaOcERnz+EBAoGATm32+nbH+xprM/ygkTFu
    p6U9GmqGUcI9MFlGhpg6oJOc0wVwb7mi8c9AahI8Ex5CmZPmkF0HuNQEJRO0DLHi
    KdrEsiJhFSecWLVKyR4Ri6sDflyYrcQgGpwgSLb8gB1VJi/Fxd/dQoY97SJGPh7S
    vXML71sZeCAnZF+gSRquldM=
    -----END PRIVATE KEY-----
  EOT
  type        = "per-hostname"
}

# Output references to test that they are preserved
output "per_zone_1_status" {
  value = cloudflare_authenticated_origin_pulls_certificate.per_zone_1.status
}
