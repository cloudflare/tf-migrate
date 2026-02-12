package dns_record

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

var migrator = NewV4ToV5Migrator()

func TestV4ToV5Transformation(t *testing.T) {
	// Test configuration transformations (automatically handles preprocessing when needed)
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "CAA record with numeric flags in data block - content renamed to value",
				Input: `
resource "cloudflare_record" "caa_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "CAA"
  ttl     = 3600

  data {
    flags   = 0
    tag     = "issue"
    content = "letsencrypt.org"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "caa_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "CAA"
  ttl     = 3600

  data = {
    flags = 0
    tag   = "issue"
    value = "letsencrypt.org"
  }
}

moved {
  from = cloudflare_record.caa_test
  to   = cloudflare_dns_record.caa_test
}`,
			},
			{
				Name: "CAA record with numeric flags in data attribute map - content renamed to value",
				Input: `
resource "cloudflare_record" "caa_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "CAA"
  ttl     = 3600
  data    {
    flags   = 0
    tag     = "issue"
    content = "letsencrypt.org"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "caa_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "CAA"
  ttl     = 3600
  data = {
    flags = 0
    tag   = "issue"
    value = "letsencrypt.org"
  }
}

moved {
  from = cloudflare_record.caa_test
  to   = cloudflare_dns_record.caa_test
}`,
			},
			{
				Name: "CAA record with flags numeric string",
				Input: `
resource "cloudflare_record" "caa" {
  zone_id = "abc123"
  name    = "test"
  type    = "CAA"
  data = {
    flags   = "128"
    tag     = "issue"
    content = "ca.example.com"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "caa" {
  zone_id = "abc123"
  name    = "test"
  type    = "CAA"
  data = {
    flags = "128"
    tag   = "issue"
    value = "ca.example.com"
  }
  ttl = 1
}

moved {
  from = cloudflare_record.caa
  to   = cloudflare_dns_record.caa
}`,
			},
			{
				Name: "CAA record with content field in middle of data attribute",
				Input: `
resource "cloudflare_record" "caa" {
  zone_id = "abc123"
  name    = "test"
  type    = "CAA"
  data = {
    tag     = "issue"
    content = "ca.example.com"
    flags   = "critical"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "caa" {
  zone_id = "abc123"
  name    = "test"
  type    = "CAA"
  data = {
    tag   = "issue"
    value = "ca.example.com"
    flags = "critical"
  }
  ttl = 1
}

moved {
  from = cloudflare_record.caa
  to   = cloudflare_dns_record.caa
}`,
			},
			{
				Name: "Non-CAA record should not be modified",
				Input: `
resource "cloudflare_record" "a_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "A"
  ttl     = 3600
  content = "192.168.1.1"
}`,
				Expected: `resource "cloudflare_dns_record" "a_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "A"
  ttl     = 3600
  content = "192.168.1.1"
}

moved {
  from = cloudflare_record.a_test
  to   = cloudflare_dns_record.a_test
}`,
			},
			{
				Name: "cloudflare_record (legacy) with CAA type - content renamed to value",
				Input: `
resource "cloudflare_record" "caa_legacy" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "CAA"
  ttl     = 3600

  data {
    flags   = 128
    tag     = "issuewild"
    content = "pki.goog"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "caa_legacy" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "CAA"
  ttl     = 3600

  data = {
    flags = 128
    tag   = "issuewild"
    value = "pki.goog"
  }
}

moved {
  from = cloudflare_record.caa_legacy
  to   = cloudflare_dns_record.caa_legacy
}`,
			},
			{
				Name: "DNS record without TTL - should add TTL with default value",
				Input: `
resource "cloudflare_record" "mx_test" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "test.example.com"
  type     = "MX"
  content  = "mx.sendgrid.net"
  priority = 10
}`,
				Expected: `resource "cloudflare_dns_record" "mx_test" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  name     = "test.example.com"
  type     = "MX"
  content  = "mx.sendgrid.net"
  priority = 10
  ttl      = 1
}

moved {
  from = cloudflare_record.mx_test
  to   = cloudflare_dns_record.mx_test
}`,
			},
			{
				Name: "DNS record with existing TTL - should keep existing value",
				Input: `
resource "cloudflare_record" "a_test_ttl" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "A"
  ttl     = 3600
  content = "192.168.1.1"
}`,
				Expected: `resource "cloudflare_dns_record" "a_test_ttl" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "A"
  ttl     = 3600
  content = "192.168.1.1"
}

moved {
  from = cloudflare_record.a_test_ttl
  to   = cloudflare_dns_record.a_test_ttl
}`,
			},
			{
				Name: "Multiple CAA records in same file - content renamed to value and TTL added",
				Input: `
resource "cloudflare_record" "caa_test1" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test1.example.com"
  type    = "CAA"
  data {
    flags   = 0
    tag     = "issue"
    content = "letsencrypt.org"
  }
}

resource "cloudflare_record" "caa_test2" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test2.example.com"
  type    = "CAA"
  data {
    flags   = 128
    tag     = "issuewild"
    content = "pki.goog"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "caa_test1" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test1.example.com"
  type    = "CAA"
  ttl     = 1
  data = {
    flags = 0
    tag   = "issue"
    value = "letsencrypt.org"
  }
}

moved {
  from = cloudflare_record.caa_test1
  to   = cloudflare_dns_record.caa_test1
}

resource "cloudflare_dns_record" "caa_test2" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test2.example.com"
  type    = "CAA"
  ttl     = 1
  data = {
    flags = 128
    tag   = "issuewild"
    value = "pki.goog"
  }
}

moved {
  from = cloudflare_record.caa_test2
  to   = cloudflare_dns_record.caa_test2
}`,
			},
			{
				Name: "DNS record with value field should rename to content",
				Input: `
resource "cloudflare_record" "a_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "A"
  value   = "192.168.1.1"
}`,
				Expected: `resource "cloudflare_dns_record" "a_test" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
  name    = "test.example.com"
  type    = "A"
  ttl     = 1
  content = "192.168.1.1"
}

moved {
  from = cloudflare_record.a_test
  to   = cloudflare_dns_record.a_test
}`,
			},
			// Additional test cases for better coverage
			{
				Name: "MX record with data block - priority hoisted",
				Input: `
resource "cloudflare_record" "mx" {
  zone_id = "abc123"
  name    = "@"
  type    = "MX"

  data {
    priority = 10
    target   = "mail.example.com"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "mx" {
  zone_id = "abc123"
  name    = "@"
  type    = "MX"

  ttl      = 1
  priority = 10
  data = {
    target = "mail.example.com"
  }
}

moved {
  from = cloudflare_record.mx
  to   = cloudflare_dns_record.mx
}`,
			},
			{
				Name: "URI record with data block - priority hoisted",
				Input: `
resource "cloudflare_record" "uri" {
  zone_id = "abc123"
  name    = "_http._tcp"
  type    = "URI"

  data {
    priority = 10
    weight   = 1
    target   = "http://example.com"
  }
}`,
				Expected: `resource "cloudflare_dns_record" "uri" {
  zone_id = "abc123"
  name    = "_http._tcp"
  type    = "URI"

  ttl      = 1
  priority = 10
  data = {
    weight = 1
    target = "http://example.com"
  }
}

moved {
  from = cloudflare_record.uri
  to   = cloudflare_dns_record.uri
}`,
			},
			{
				Name: "Record without type attribute",
				Input: `
resource "cloudflare_record" "no_type" {
  zone_id = "abc123"
  name    = "test"
  value   = "192.0.2.1"
}`,
				Expected: `resource "cloudflare_dns_record" "no_type" {
  zone_id = "abc123"
  name    = "test"
  ttl     = 1
  content = "192.0.2.1"
}

moved {
  from = cloudflare_record.no_type
  to   = cloudflare_dns_record.no_type
}`,
			},
			{
				Name: "OPENPGPKEY record value renamed to content",
				Input: `
resource "cloudflare_record" "pgp" {
  zone_id = "abc123"
  name    = "test"
  type    = "OPENPGPKEY"
  value   = "base64encodedkey"
}`,
				Expected: `resource "cloudflare_dns_record" "pgp" {
  zone_id = "abc123"
  name    = "test"
  type    = "OPENPGPKEY"
  ttl     = 1
  content = "base64encodedkey"
}

moved {
  from = cloudflare_record.pgp
  to   = cloudflare_dns_record.pgp
}`,
			},
			{
				Name: "AAAA record with compressed IPv6",
				Input: `
resource "cloudflare_record" "ipv6" {
  zone_id = "abc123"
  name    = "test"
  type    = "AAAA"
  value   = "2001:db8::1"
}`,
				Expected: `resource "cloudflare_dns_record" "ipv6" {
  zone_id = "abc123"
  name    = "test"
  type    = "AAAA"
  ttl     = 1
  content = "2001:db8::1"
}

moved {
  from = cloudflare_record.ipv6
  to   = cloudflare_dns_record.ipv6
}`,
			},
			{
				Name: "AAAA record with full IPv6 address",
				Input: `
resource "cloudflare_record" "ipv6_full" {
  zone_id = "abc123"
  name    = "test"
  type    = "AAAA"
  value   = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
}`,
				Expected: `resource "cloudflare_dns_record" "ipv6_full" {
  zone_id = "abc123"
  name    = "test"
  type    = "AAAA"
  ttl     = 1
  content = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
}

moved {
  from = cloudflare_record.ipv6_full
  to   = cloudflare_dns_record.ipv6_full
}`,
			},
			{
				Name: "AAAA record with existing content field",
				Input: `
resource "cloudflare_record" "ipv6_content" {
  zone_id = "abc123"
  name    = "ipv6"
  type    = "AAAA"
  content = "2001:db8::1"
  ttl     = 3600
}`,
				Expected: `resource "cloudflare_dns_record" "ipv6_content" {
  zone_id = "abc123"
  name    = "ipv6"
  type    = "AAAA"
  content = "2001:db8::1"
  ttl     = 3600
}

moved {
  from = cloudflare_record.ipv6_content
  to   = cloudflare_dns_record.ipv6_content
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
