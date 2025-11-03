package load_balancer_pool

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "header block transformation",
				Input: `resource "cloudflare_load_balancer_pool" "test" {
  origins = [{
    name    = "test"
    address = "192.0.2.1"
    header {
      header = "Host"
      values = ["example.com"]
    }
  }]
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "test" {
  origins = [{
    name    = "test"
    address = "192.0.2.1"
    header = {
      host = ["example.com"]
    }
  }]
}`,
			},
			{
				Name: "no header block",
				Input: `resource "cloudflare_load_balancer_pool" "test" {
  origins = [{
    name    = "test"
    address = "192.0.2.1"
  }]
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "test" {
  origins = [{
    name    = "test"
    address = "192.0.2.1"
  }]
}`,
			},
			{
				Name: "multiple origins",
				Input: `resource "cloudflare_load_balancer_pool" "test" {
  origins = [{
    name    = "test1"
    address = "192.0.2.1"
  }, {
    name    = "test2"
    address = "192.0.2.2"
  }]
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "test" {
  origins = [{
    name    = "test1"
    address = "192.0.2.1"
  }, {
    name    = "test2"
    address = "192.0.2.2"
  }]
}`,
			},
			{
				Name: "dynamic origins block simple",
				Input: `locals {
  origin_list = ["192.0.2.1", "192.0.2.2"]
}

resource "cloudflare_load_balancer_pool" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-pool"

  dynamic "origins" {
    for_each = local.origin_list
    content {
      name    = "origin-${origins.key}"
      address = origins.value
      enabled = true
    }
  }
}`,
				Expected: `locals {
  origin_list = ["192.0.2.1", "192.0.2.2"]
}

resource "cloudflare_load_balancer_pool" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-pool"

  origins = [for key, value in local.origin_list : {
    name    = "origin-${key}"
    address = value
    enabled = true
  }]
}`,
			},
			{
				Name: "dynamic origins with header block",
				Input: `locals {
  origin_configs = [
    {
      name    = "origin1"
      address = "192.0.2.1"
      host    = "example1.com"
    },
    {
      name    = "origin2"
      address = "192.0.2.2"
      host    = "example2.com"
    }
  ]
}

resource "cloudflare_load_balancer_pool" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-pool"

  dynamic "origins" {
    for_each = local.origin_configs
    content {
      name    = origins.value.name
      address = origins.value.address
      enabled = true

      header {
        header = "Host"
        values = [origins.value.host]
      }
    }
  }
}`,
				Expected: `locals {
  origin_configs = [
    {
      name    = "origin1"
      address = "192.0.2.1"
      host    = "example1.com"
    },
    {
      name    = "origin2"
      address = "192.0.2.2"
      host    = "example2.com"
    }
  ]
}

resource "cloudflare_load_balancer_pool" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-pool"

  origins = [for key, value in local.origin_configs : {
    name    = value.name
    address = value.address
    enabled = true
    header = {
      host = [value.host]
    }
  }]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
