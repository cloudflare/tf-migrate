package dns_record

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

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
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "CAA record v4 format with array data and numeric flags",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_record",
					"name": "caa_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "CAA",
							"content": "0 issue letsencrypt.org",
							"data": [{
								"flags": 0,
								"tag": "issue",
								"content": "letsencrypt.org"
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_dns_record",
					"name": "caa_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "CAA",
							"ttl": 1,
							"content": "0 issue letsencrypt.org",
							"created_on": "2024-01-01T00:00:00Z",
							"modified_on": "2024-01-01T00:00:00Z",
							"data": {
								"flags": {
									"type": "string",
									"value": "0"
								},
								"tag": "issue",
								"value": "letsencrypt.org"
							}
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "Simple A record should set data field to null",
				Input: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_record",
					"name": "a_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "A",
							"content": "192.168.1.1",
							"data": []
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_dns_record",
					"name": "a_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "A",
							"ttl": 1,
							"content": "192.168.1.1",
							"created_on": "2024-01-01T00:00:00Z",
							"modified_on": "2024-01-01T00:00:00Z"
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "Schema version should be updated from v4 (3) to v5 (0)",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_record",
					"name": "test",
					"instances": [{
						"schema_version": 3,
						"attributes": {
							"id": "test-id",
							"zone_id": "test-zone",
							"name": "test",
							"type": "A",
							"value": "192.168.1.1",
							"ttl": 300
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_dns_record",
					"name": "test",
					"instances": [{
						"schema_version": 0,
						"attributes": {
							"id": "test-id",
							"zone_id": "test-zone",
							"name": "test",
							"type": "A",
							"content": "192.168.1.1",
							"ttl": 300,
							"created_on": "2024-01-01T00:00:00Z",
							"modified_on": "2024-01-01T00:00:00Z"
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "cloudflare_record (legacy) renamed to cloudflare_dns_record",
				Input: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_record",
					"name": "caa_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "CAA",
							"content": "0 issue letsencrypt.org",
							"data": [{
								"flags": 0,
								"tag": "issue",
								"content": "letsencrypt.org"
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_dns_record",
					"name": "caa_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "CAA",
							"ttl": 1,
							"content": "0 issue letsencrypt.org",
							"created_on": "2024-01-01T00:00:00Z",
							"modified_on": "2024-01-01T00:00:00Z",
							"data": {
								"flags": {
									"type": "string",
									"value": "0"
								},
								"tag": "issue",
								"value": "letsencrypt.org"
							}
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "Record with value field renamed to content",
				Input: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_record",
					"name": "a_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "A",
							"value": "192.168.1.1",
							"hostname": "test.example.com",
							"allow_overwrite": true
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_dns_record",
					"name": "a_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "test.example.com",
							"type": "A",
							"ttl": 1,
							"content": "192.168.1.1",
							"created_on": "2024-01-01T00:00:00Z",
							"modified_on": "2024-01-01T00:00:00Z"
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "SRV record with array data migration",
				Input: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_record",
					"name": "srv_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "_sip._tcp.example.com",
							"type": "SRV",
							"data": [{
								"priority": 10,
								"weight": 60,
								"port": 5060,
								"target": "sipserver.example.com"
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"resources": [{
					"type": "cloudflare_dns_record",
					"name": "srv_test",
					"instances": [{
						"attributes": {
							"id": "test-id",
							"zone_id": "0da42c8d2132a9ddaf714f9e7c920711",
							"name": "_sip._tcp.example.com",
							"type": "SRV",
							"priority": 10,
							"ttl": 1,
							"created_on": "2024-01-01T00:00:00Z",
							"modified_on": "2024-01-01T00:00:00Z",
							"data": {
								"priority": 10,
								"weight": 60,
								"port": 5060,
								"target": "sipserver.example.com"
							}
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			// Additional state test cases for better coverage
			{
				Name: "State with missing attributes - should skip",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "invalid",
						"instances": [{
							"attributes": {}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "invalid",
						"instances": [{
							"attributes": {},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "State without instances",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "empty",
						"instances": []
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "empty",
						"instances": []
					}]
				}`,
			},
			{
				Name: "MX record state with data array",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "mx",
						"instances": [{
							"attributes": {
								"id": "mx123",
								"zone_id": "zone123",
								"name": "@",
								"type": "MX",
								"data": [{
									"priority": 10,
									"target": "mail.example.com"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "mx",
						"instances": [{
							"attributes": {
								"id": "mx123",
								"zone_id": "zone123",
								"name": "@",
								"type": "MX",
								"priority": 10,
								"content": "10 mail.example.com",
								"data": {
									"target": "mail.example.com"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "URI record state with data array",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "uri",
						"instances": [{
							"attributes": {
								"id": "uri123",
								"zone_id": "zone123",
								"name": "_http._tcp",
								"type": "URI",
								"data": [{
									"priority": 10,
									"weight": 1,
									"target": "http://example.com"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "uri",
						"instances": [{
							"attributes": {
								"id": "uri123",
								"zone_id": "zone123",
								"name": "_http._tcp",
								"type": "URI",
								"priority": 10,
								"content": "10 1 http://example.com",
								"data": {
									"weight": 1,
									"target": "http://example.com"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "CAA record with numeric string flags - should convert to number type",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"data": [{
									"flags": "128",
									"tag": "issue",
									"content": "letsencrypt.org"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"content": "128 issue letsencrypt.org",
								"data": {
									"flags": {
										"type": "string",
										"value": "128"
									},
									"tag": "issue",
									"value": "letsencrypt.org"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "CAA record with null flags",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"data": [{
									"flags": null,
									"tag": "issue",
									"content": "letsencrypt.org"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"content": "0 issue letsencrypt.org",
								"data": {
									"flags": null,
									"tag": "issue",
									"value": "letsencrypt.org"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "CAA record with non-numeric string flags",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"data": [{
									"flags": "critical",
									"tag": "issue",
									"content": "letsencrypt.org"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"content": "critical issue letsencrypt.org",
								"data": {
									"flags": {
										"type": "string",
										"value": "critical"
									},
									"tag": "issue",
									"value": "letsencrypt.org"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "CAA record with empty string flags",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"data": [{
									"flags": "",
									"tag": "issue",
									"content": "letsencrypt.org"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "caa",
						"instances": [{
							"attributes": {
								"id": "caa123",
								"zone_id": "zone123",
								"name": "example.com",
								"type": "CAA",
								"content": "0 issue letsencrypt.org",
								"data": {
									"flags": null,
									"tag": "issue",
									"value": "letsencrypt.org"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "State with empty resources",
				Input: `{
					"resources": []
				}`,
				Expected: `{
					"resources": []
				}`,
			},
			{
				Name: "Record with created_on but no modified_on",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "test",
								"type": "A",
								"value": "192.0.2.1",
								"created_on": "2023-01-01T00:00:00Z"
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "test",
								"type": "A",
								"content": "192.0.2.1",
								"ttl": 1,
								"created_on": "2023-01-01T00:00:00Z",
								"modified_on": "2023-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "Record with both value and content fields",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "test",
								"type": "A",
								"value": "192.0.2.1",
								"content": "192.0.2.2"
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "test",
								"type": "A",
								"content": "192.0.2.2",
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "MX record with integer priority - converts to float64",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "mail",
								"type": "MX",
								"priority": 10,
								"content": "10 mail.example.com"
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "mail",
								"type": "MX",
								"priority": 10,
								"content": "10 mail.example.com",
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "SRV record with integer fields in data - converts to float64",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "_sip._tcp",
								"type": "SRV",
								"data": [{
									"priority": 10,
									"weight": 60,
									"port": 5060,
									"target": "sipserver.example.com"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "_sip._tcp",
								"type": "SRV",
								"priority": 10,
								"data": {
									"priority": 10,
									"weight": 60,
									"port": 5060,
									"target": "sipserver.example.com"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "TLSA record with integer fields in data - converts to float64",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "_443._tcp",
								"type": "TLSA",
								"data": [{
									"usage": 3,
									"selector": 1,
									"matching_type": 1,
									"certificate": "abcdef123456"
								}]
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "test",
						"instances": [{
							"attributes": {
								"zone_id": "zone123",
								"name": "_443._tcp",
								"type": "TLSA",
								"data": {
									"usage": 3,
									"selector": 1,
									"matching_type": 1,
									"certificate": "abcdef123456"
								},
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
			{
				Name: "AAAA record with compressed IPv6 - preserved as-is",
				Input: `{
					"resources": [{
						"type": "cloudflare_record",
						"name": "ipv6_test",
						"instances": [{
							"attributes": {
								"id": "ipv6-id",
								"zone_id": "zone123",
								"name": "ipv6.example.com",
								"type": "AAAA",
								"value": "2001:db8::1"
							}
						}]
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_dns_record",
						"name": "ipv6_test",
						"instances": [{
							"attributes": {
								"id": "ipv6-id",
								"zone_id": "zone123",
								"name": "ipv6.example.com",
								"type": "AAAA",
								"content": "2001:db8::1",
								"ttl": 1,
								"created_on": "2024-01-01T00:00:00Z",
								"modified_on": "2024-01-01T00:00:00Z"
							},
							"schema_version": 0
						}]
					}]
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

}
