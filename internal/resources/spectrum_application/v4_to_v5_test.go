package spectrum_application

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "remove id attribute",
				Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  id       = "123"
  protocol = "tcp/22"
}`,
				Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/22"
}`,
			},
			{
				Name: "convert origin_port_range block to origin_port string",
				Input: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/22"

  origin_port_range {
    start = 80
    end   = 85
  }
}`,
				Expected: `resource "cloudflare_spectrum_application" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  protocol = "tcp/22"

  origin_port = "80-85"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
