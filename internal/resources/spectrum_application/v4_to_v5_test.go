package spectrum_application

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		testConfigTransformations(t, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		testStateTransformations(t, migrator)
	})
}

func testConfigTransformations(t *testing.T, migrator transform.ResourceTransformer) {
	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Basic resource with dns block",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "example.com"
  protocol = "tcp/443"
  dns {
    type = "CNAME"
    name = "test.example.com"
  }
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/443"
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "CNAME"
    name = "test.example.com"
  }
}`,
		},
		{
			Name: "Remove optional id attribute",
			Input: `resource "cloudflare_spectrum_application" "example" {
  id           = "some-user-provided-id"
  zone_id      = "example.com"
  protocol     = "tcp/443"
  dns {
    type = "CNAME"
    name = "secure.example.com"
  }
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/443"
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "CNAME"
    name = "secure.example.com"
  }
}`,
		},
		{
			Name: "Convert origin_dns block to attribute",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "example.com"
  protocol = "tcp/3306"
  dns {
    type = "ADDRESS"
    name = "db.example.com"
  }
  origin_dns {
    name = "origin.example.com"
  }
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "example.com"
  protocol = "tcp/3306"
  dns = {
    type = "ADDRESS"
    name = "db.example.com"
  }
  origin_dns = {
    name = "origin.example.com"
  }
}`,
		},
		{
			Name: "Convert edge_ips block to attribute",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "example.com"
  protocol = "tcp/443"
  dns {
    type = "CNAME"
    name = "app.example.com"
  }
  edge_ips {
    type         = "dynamic"
    connectivity = "all"
  }
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/443"
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "CNAME"
    name = "app.example.com"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "all"
  }
}`,
		},
		{
			Name: "Convert origin_port_range to origin_port string",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "example.com"
  protocol = "tcp/3306-3310"
  dns {
    type = "ADDRESS"
    name = "db.example.com"
  }
  origin_port_range {
    start = 3306
    end   = 3310
  }
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/3306-3310"
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "ADDRESS"
    name = "db.example.com"
  }
  origin_port = "3306-3310"
}`,
		},
		{
			Name: "Preserve existing origin_port attribute",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/3306"
  dns {
    type = "ADDRESS"
    name = "db.example.com"
  }
  origin_port   = 3306
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/3306"
  origin_port   = 3306
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "ADDRESS"
    name = "db.example.com"
  }
}`,
		},
		{
			Name: "Complex resource with all optional fields",
			Input: `resource "cloudflare_spectrum_application" "example" {
  id            = "user-specified-id"
  zone_id       = "example.com"
  protocol      = "tcp/443"
  dns {
    type = "CNAME"
    name = "app.example.com"
  }
  edge_ips {
    type         = "dynamic"
    connectivity = "all"
  }
  origin_dns {
    name = "backend.example.com"
  }
  tls                = "flexible"
  argo_smart_routing = true
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id            = "example.com"
  protocol           = "tcp/443"
  tls                = "flexible"
  argo_smart_routing = true
  dns = {
    type = "CNAME"
    name = "app.example.com"
  }
  origin_dns = {
    name = "backend.example.com"
  }
  edge_ips = {
    type         = "dynamic"
    connectivity = "all"
  }
}`,
		},
		{
			Name: "Remove id and convert origin_port_range simultaneously",
			Input: `resource "cloudflare_spectrum_application" "example" {
  id       = "user-provided-id"
  zone_id  = "example.com"
  protocol = "tcp/8080"
  dns {
    type = "CNAME"
    name = "app.example.com"
  }
  origin_port_range {
    start = 8080
    end   = 8090
  }
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/8080"
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "CNAME"
    name = "app.example.com"
  }
  origin_port = "8080-8090"
}`,
		},
		{
			Name: "No transformation needed when id not present and no blocks",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  dns {
    type = "CNAME"
    name = "ssh.example.com"
  }
  origin_direct = ["203.0.113.1"]
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["203.0.113.1"]
  dns = {
    type = "CNAME"
    name = "ssh.example.com"
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testStateTransformations(t *testing.T, migrator transform.ResourceTransformer) {
	tests := []testhelpers.StateTestCase{
		{
			Name: "Basic state with dns array to object",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "example.com",
    "protocol": "tcp/443",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "origin_direct": ["203.0.113.1"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "example.com",
    "protocol": "tcp/443",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "origin_direct": ["203.0.113.1"]
  }
}`,
		},
		{
			Name: "origin_port integer value preserved",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/3306",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "origin_direct": ["tcp://128.66.0.2:3306"],
    "origin_port": 3306
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/3306",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "origin_direct": ["tcp://128.66.0.2:3306"],
    "origin_port": {
      "type": "number",
      "value": 3306
    }
  }
}`,
		},
		{
			Name: "origin_port_range array converted to origin_port string",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/3306",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "origin_direct": ["tcp://128.66.0.1:23"],
    "origin_port_range": [
      {
        "start": 3306,
        "end": 3310
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/3306",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "origin_direct": ["tcp://128.66.0.1:23"],
    "origin_port": {
      "type": "string",
      "value": "3306-3310"
    }
  }
}`,
		},
		{
			Name: "origin_dns array to object conversion",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "origin_dns": [
      {
        "name": "origin.example.com"
      }
    ],
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "origin_dns": {
      "name": "origin.example.com"
    },
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
		},
		{
			Name: "edge_ips array to object conversion",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "edge_ips": [
      {
        "type": "dynamic",
        "connectivity": "all"
      }
    ],
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "edge_ips": {
      "type": "dynamic",
      "connectivity": "all"
    },
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
		},
		{
			Name: "All nested objects converted from arrays",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "edge_ips": [
      {
        "type": "dynamic",
        "connectivity": "all"
      }
    ],
    "origin_dns": [
      {
        "name": "origin.example.com",
        "type": "A"
      }
    ],
    "origin_direct": ["tcp://128.66.0.3:443"],
    "tls": "flexible",
    "argo_smart_routing": true,
    "proxy_protocol": "v1",
    "ip_firewall": true,
    "traffic_type": "direct"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "edge_ips": {
      "type": "dynamic",
      "connectivity": "all"
    },
    "origin_dns": {
      "name": "origin.example.com",
      "type": "A"
    },
    "origin_direct": ["tcp://128.66.0.3:443"],
    "tls": "flexible",
    "argo_smart_routing": true,
    "proxy_protocol": "v1",
    "ip_firewall": true,
    "traffic_type": "direct"
  }
}`,
		},
		{
			Name: "Empty dns array is deleted",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": [],
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
		},
		{
			Name: "Empty origin_dns array is deleted",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "origin_dns": [],
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
		},
		{
			Name: "Empty edge_ips array is deleted",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": [
      {
        "type": "CNAME",
        "name": "test.example.com"
      }
    ],
    "edge_ips": [],
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "test-zone-id",
    "protocol": "tcp/443",
    "dns": {
      "type": "CNAME",
      "name": "test.example.com"
    },
    "origin_direct": ["tcp://128.66.0.3:443"]
  }
}`,
		},
		{
			Name: "State with no attributes still sets schema_version",
			Input: `{
  "schema_version": 1
}`,
			Expected: `{
  "schema_version": 0
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
