package custom_hostname

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

var migrator = NewV4ToV5Migrator()

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "convert ssl/settings blocks and remove wait flag",
				Input: `
resource "cloudflare_custom_hostname" "example" {
  zone_id = "deadbeefdeadbeefdeadbeefdeadbeef"
  hostname = "app.example.com"

  ssl {
    method = "txt"
    type = "dv"
    settings {
      tls13 = "on"
      http2 = "on"
    }
  }

  wait_for_ssl_pending_validation = true
}`,
				Expected: `resource "cloudflare_custom_hostname" "example" {
  zone_id  = "deadbeefdeadbeefdeadbeefdeadbeef"
  hostname = "app.example.com"
  ssl = {
    method   = "txt"
    type     = "dv"
    wildcard = false
    settings = {
      tls_1_3 = "on"
      http2   = "on"
    }
  }
}`,
			},
			{
				Name: "preserve resource type and simple attributes",
				Input: `
resource "cloudflare_custom_hostname" "minimal" {
  zone_id = "deadbeefdeadbeefdeadbeefdeadbeef"
  hostname = "minimal.example.com"
  ssl {
    method = "txt"
  }
}`,
				Expected: `resource "cloudflare_custom_hostname" "minimal" {
  zone_id  = "deadbeefdeadbeefdeadbeefdeadbeef"
  hostname = "minimal.example.com"
  ssl = {
    method   = "txt"
    wildcard = false
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
