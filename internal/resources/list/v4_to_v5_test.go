package list

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		t.Run("IPListTransformations", testIPListConfig)
		t.Run("ASNListTransformations", testASNListConfig)
		t.Run("HostnameListTransformations", testHostnameListConfig)
		t.Run("RedirectListTransformations", testRedirectListConfig)
		t.Run("DynamicBlockTransformations", testDynamicBlockConfig)
		t.Run("EdgeCases", testConfigEdgeCases)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		t.Run("IPListState", testIPListState)
		t.Run("ASNListState", testASNListState)
		t.Run("HostnameListState", testHostnameListState)
		t.Run("RedirectListState", testRedirectListState)
	})
}

func testIPListConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Single IP item",
			Input: `resource "cloudflare_list" "ip_list" {
  account_id  = "abc123"
  name        = "ip_list"
  kind        = "ip"
  description = "List of IP addresses"

  item {
    comment = "First IP"
    value {
      ip = "1.1.1.1"
    }
  }
}`,
			Expected: `resource "cloudflare_list" "ip_list" {
  account_id  = "abc123"
  name        = "ip_list"
  kind        = "ip"
  description = "List of IP addresses"

  items = [{
    comment = "First IP"
    ip      = "1.1.1.1"
  }]
}`,
		},
		{
			Name: "Multiple IP items",
			Input: `resource "cloudflare_list" "ip_list" {
  account_id  = "abc123"
  name        = "ip_list"
  kind        = "ip"
  description = "List of IP addresses"

  item {
    comment = "First IP"
    value {
      ip = "1.1.1.1"
    }
  }

  item {
    comment = "Second IP"
    value {
      ip = "1.1.1.2"
    }
  }

  item {
    value {
      ip = "1.1.1.3"
    }
  }
}`,
			Expected: `resource "cloudflare_list" "ip_list" {
  account_id  = "abc123"
  name        = "ip_list"
  kind        = "ip"
  description = "List of IP addresses"

  items = [{
    comment = "First IP"
    ip      = "1.1.1.1"
  }, {
    comment = "Second IP"
    ip      = "1.1.1.2"
  }, {
    ip = "1.1.1.3"
  }]
}`,
		},
		{
			Name: "IP item without comment",
			Input: `resource "cloudflare_list" "ip_list" {
  account_id = "abc123"
  name       = "ip_list"
  kind       = "ip"

  item {
    value {
      ip = "10.0.0.1"
    }
  }
}`,
			Expected: `resource "cloudflare_list" "ip_list" {
  account_id = "abc123"
  name       = "ip_list"
  kind       = "ip"

  items = [{
    ip = "10.0.0.1"
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testASNListConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "ASN list migration",
			Input: `resource "cloudflare_list" "asn_list" {
  account_id = "abc123"
  name       = "asn_list"
  kind       = "asn"

  item {
    comment = "Google ASN"
    value {
      asn = 15169
    }
  }

  item {
    value {
      asn = 13335
    }
  }
}`,
			Expected: `resource "cloudflare_list" "asn_list" {
  account_id = "abc123"
  name       = "asn_list"
  kind       = "asn"

  items = [{
    asn     = 15169
    comment = "Google ASN"
  }, {
    asn = 13335
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testHostnameListConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Hostname list migration",
			Input: `resource "cloudflare_list" "hostname_list" {
  account_id = "abc123"
  name       = "hostname_list"
  kind       = "hostname"

  item {
    comment = "Example hostname"
    value {
      hostname {
        url_hostname = "example.com"
      }
    }
  }

  item {
    value {
      hostname {
        url_hostname = "test.example.com"
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "hostname_list" {
  account_id = "abc123"
  name       = "hostname_list"
  kind       = "hostname"

  items = [{
    comment = "Example hostname"
    hostname = {
      url_hostname = "example.com"
    }
  }, {
    hostname = {
      url_hostname = "test.example.com"
    }
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testRedirectListConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Redirect list migration with boolean conversions",
			Input: `resource "cloudflare_list" "redirect_list" {
  account_id = "abc123"
  name       = "redirect_list"
  kind       = "redirect"

  item {
    comment = "Main redirect"
    value {
      redirect {
        source_url            = "example.com/old"
        target_url            = "example.com/new"
        include_subdomains    = "enabled"
        subpath_matching      = "disabled"
        preserve_query_string = "enabled"
        preserve_path_suffix  = "disabled"
        status_code           = 301
      }
    }
  }

  item {
    value {
      redirect {
        source_url         = "test.com"
        target_url         = "newtest.com"
        include_subdomains = "disabled"
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "redirect_list" {
  account_id = "abc123"
  name       = "redirect_list"
  kind       = "redirect"

  items = [{
    comment = "Main redirect"
    redirect = {
      include_subdomains    = true
      preserve_path_suffix  = false
      preserve_query_string = true
      source_url            = "example.com/old"
      status_code           = 301
      subpath_matching      = false
      target_url            = "example.com/new"
    }
  }, {
    redirect = {
      include_subdomains = false
      source_url         = "test.com/"
      target_url         = "newtest.com"
    }
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testDynamicBlockConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "List with only dynamic blocks",
			Input: `resource "cloudflare_list" "dynamic_ip_list" {
  account_id = "abc123"
  name       = "dynamic_list"
  kind       = "ip"

  dynamic "item" {
    for_each = var.ip_list
    content {
      value {
        ip = item.value
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "dynamic_ip_list" {
  account_id = "abc123"
  name       = "dynamic_list"
  kind       = "ip"

  items = [for item in var.ip_list : {
    ip = item
  }]
}`,
		},
		{
			Name: "List with mixed static and dynamic items",
			Input: `resource "cloudflare_list" "mixed_list" {
  account_id = "abc123"
  name       = "mixed"
  kind       = "ip"

  item {
    value {
      ip = "1.1.1.1"
    }
  }

  dynamic "item" {
    for_each = var.additional_ips
    content {
      value {
        ip = item.value
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "mixed_list" {
  account_id = "abc123"
  name       = "mixed"
  kind       = "ip"

  items = concat([{ ip = "1.1.1.1" }], [for item in var.additional_ips : {
    ip = item
  }])
}`,
		},
		{
			Name: "Dynamic block with for_each and iterator",
			Input: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "ip"

  dynamic "item" {
    for_each = var.ip_addresses
    iterator = ip_item
    content {
      value {
        ip = ip_item.value
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "ip"

  items = [for ip_item in var.ip_addresses : {
    ip = ip_item
  }]
}`,
		},
		{
			Name: "Dynamic block with ASN and comment",
			Input: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "asn"

  dynamic "item" {
    for_each = var.asn_list
    content {
      value {
        asn = item.value.number
      }
      comment = item.value.description
    }
  }
}`,
			Expected: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "asn"

  items = [for item in var.asn_list : {
    asn     = item.number
    comment = item.description
  }]
}`,
		},
		{
			Name: "Dynamic block with hostname",
			Input: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "hostname"

  dynamic "item" {
    for_each = var.hostnames
    iterator = host
    content {
      value {
        hostname {
          url_hostname = host.value
        }
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "hostname"

  items = [for host in var.hostnames : {
    hostname = { url_hostname = host }
  }]
}`,
		},
		{
			Name: "Dynamic block with redirect",
			Input: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "redirect"

  dynamic "item" {
    for_each = var.redirects
    iterator = redir
    content {
      value {
        redirect {
          source_url         = redir.value.from
          target_url         = redir.value.to
          include_subdomains = "enabled"
          status_code        = 301
        }
      }
    }
  }
}`,
			Expected: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test"
  kind       = "redirect"

  items = [for redir in var.redirects : {
    redirect = {
      source_url         = redir.from
      target_url         = redir.to
      include_subdomains = true
      status_code        = 301
    }
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testConfigEdgeCases(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Empty list (no items)",
			Input: `resource "cloudflare_list" "empty_list" {
  account_id  = "abc123"
  name        = "empty_list"
  kind        = "ip"
  description = "Empty list"
}`,
			Expected: `resource "cloudflare_list" "empty_list" {
  account_id  = "abc123"
  name        = "empty_list"
  kind        = "ip"
  description = "Empty list"
}`,
		},
		{
			Name: "List without kind attribute (should not transform)",
			Input: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test_list"

  item {
    value {
      ip = "1.1.1.1"
    }
  }
}`,
			Expected: `resource "cloudflare_list" "test" {
  account_id = "abc123"
  name       = "test_list"

  item {
    value {
      ip = "1.1.1.1"
    }
  }
}`,
		},
		{
			Name: "Mixed list with comments and without",
			Input: `resource "cloudflare_list" "mixed_list" {
  account_id = "abc123"
  name       = "mixed_list"
  kind       = "ip"

  item {
    comment = "With comment"
    value {
      ip = "10.0.0.1"
    }
  }

  item {
    value {
      ip = "10.0.0.2"
    }
  }

  item {
    comment = "Another comment"
    value {
      ip = "10.0.0.3"
    }
  }
}`,
			Expected: `resource "cloudflare_list" "mixed_list" {
  account_id = "abc123"
  name       = "mixed_list"
  kind       = "ip"

  items = [{
    comment = "With comment"
    ip      = "10.0.0.1"
  }, {
    ip = "10.0.0.2"
  }, {
    comment = "Another comment"
    ip      = "10.0.0.3"
  }]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testIPListState(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "IP list state transformation",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-123",
    "kind": "ip",
    "name": "test-list",
    "item": [
      {
        "comment": "Test IP 1",
        "value": [{
          "ip": "192.0.2.1"
        }]
      },
      {
        "comment": "Test IP 2",
        "value": [{
          "ip": "192.0.2.2"
        }]
      }
    ],
    "num_items": 2
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-123",
    "kind": "ip",
    "name": "test-list",
    "items": [
      {
        "comment": "Test IP 1",
        "ip": "192.0.2.1"
      },
      {
        "comment": "Test IP 2",
        "ip": "192.0.2.2"
      }
    ],
    "num_items": 2
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func testASNListState(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "ASN list state transformation",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-456",
    "kind": "asn",
    "name": "test-asn-list",
    "item": [
      {
        "comment": "Google",
        "value": [{
          "asn": 15169
        }]
      }
    ],
    "num_items": 1
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-456",
    "kind": "asn",
    "name": "test-asn-list",
    "items": [
      {
        "comment": "Google",
        "asn": 15169
      }
    ],
    "num_items": 1
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func testHostnameListState(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "Hostname list state transformation",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-789",
    "kind": "hostname",
    "name": "test-hostname-list",
    "item": [
      {
        "comment": "Test hostname",
        "value": [{
          "hostname": [{
            "url_hostname": "example.com"
          }]
        }]
      }
    ],
    "num_items": 1
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-789",
    "kind": "hostname",
    "name": "test-hostname-list",
    "items": [
      {
        "comment": "Test hostname",
        "hostname": {
          "url_hostname": "example.com"
        }
      }
    ],
    "num_items": 1
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func testRedirectListState(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "Redirect list state transformation with boolean conversion",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-101",
    "kind": "redirect",
    "name": "test-redirect-list",
    "item": [
      {
        "value": [{
          "redirect": [{
            "source_url": "example.com/old",
            "target_url": "example.com/new",
            "status_code": 301,
            "include_subdomains": "enabled",
            "subpath_matching": "disabled",
            "preserve_query_string": "enabled",
            "preserve_path_suffix": "disabled"
          }]
        }]
      }
    ],
    "num_items": 1
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "abc123",
    "id": "list-101",
    "kind": "redirect",
    "name": "test-redirect-list",
    "items": [
      {
        "redirect": {
          "source_url": "example.com/old",
          "target_url": "example.com/new",
          "status_code": 301,
          "include_subdomains": true,
          "subpath_matching": false,
          "preserve_query_string": true,
          "preserve_path_suffix": false
        }
      }
    ],
    "num_items": 1
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func TestNormalizeIPAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "CIDR /8",
			input: "10.0.0.0/8",
			want:  "10.0.0.0",
		},
		{
			name:  "CIDR /16",
			input: "192.168.0.0/16",
			want:  "192.168.0.0",
		},
		{
			name:  "CIDR /24",
			input: "192.168.1.0/24",
			want:  "192.168.1.0",
		},
		{
			name:  "IPv4 without CIDR",
			input: "1.1.1.1",
			want:  "1.1.1.1",
		},
		{
			name:  "IPv6 without CIDR",
			input: "2001:db8::1",
			want:  "2001:db8::1",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "IPv6 with CIDR",
			input: "2001:db8::/32",
			want:  "2001:db8::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeIPAddress(tt.input)
			if got != tt.want {
				t.Errorf("normalizeIPAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeIPAddressInExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Quoted CIDR /8",
			input: `"10.0.0.0/8"`,
			want:  `"10.0.0.0"`,
		},
		{
			name:  "Quoted CIDR /24",
			input: `"192.168.1.0/24"`,
			want:  `"192.168.1.0"`,
		},
		{
			name:  "Quoted IPv4 without CIDR",
			input: `"1.1.1.1"`,
			want:  `"1.1.1.1"`,
		},
		{
			name:  "Variable expression (unchanged)",
			input: `var.ip_address`,
			want:  `var.ip_address`,
		},
		{
			name:  "each.value expression (unchanged)",
			input: `each.value`,
			want:  `each.value`,
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "Quoted IPv6 with CIDR",
			input: `"2001:db8::/32"`,
			want:  `"2001:db8::"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeIPAddressInExpr(tt.input)
			if got != tt.want {
				t.Errorf("normalizeIPAddressInExpr(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
