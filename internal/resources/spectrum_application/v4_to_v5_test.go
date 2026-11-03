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
}

func testConfigTransformations(t *testing.T, migrator transform.ResourceTransformer) {
	tests := []testhelpers.ConfigTestCase{
		{
			Name: "origin_port_range with literal bounds",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["tcp://203.0.113.1:22"]
  origin_port_range {
    start = 1000
    end   = 2000
  }
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["tcp://203.0.113.1:22"]
  origin_port    = "1000-2000"
}`,
		},
		{
			// Reading only the first token collapsed each side to its leading
			// identifier, producing "var-var": well formed, accepted by
			// terraform validate, and meaningless.
			Name: "origin_port_range with expression bounds is interpolated",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["tcp://203.0.113.1:22"]
  origin_port_range {
    start = var.port_start
    end   = var.port_end
  }
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["tcp://203.0.113.1:22"]
  origin_port    = "${var.port_start}-${var.port_end}"
}`,
		},
		{
			Name: "origin_port_range with one literal and one expression",
			Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["tcp://203.0.113.1:22"]
  origin_port_range {
    start = 1000
    end   = local.port_base + 5
  }
}`,
			Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id       = "example.com"
  protocol      = "tcp/22"
  origin_direct = ["tcp://203.0.113.1:22"]
  origin_port    = "1000-${local.port_base + 5}"
}`,
		},
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
