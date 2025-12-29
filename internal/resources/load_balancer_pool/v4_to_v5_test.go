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
				Name: "Basic pool with origins block",
				Input: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }]
}`,
			},
			{
				Name: "Pool with multiple origins",
				Input: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
  }
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
  }, {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = true
  }]
}`,
			},
			{
				Name: "Pool with load_shedding block",
				Input: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
  }

  load_shedding {
    default_percent = 50
    default_policy  = "random"
    session_percent = 25
    session_policy  = "hash"
  }
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
  }]
  load_shedding = {
    default_percent = 50
    default_policy  = "random"
    session_percent = 25
    session_policy  = "hash"
  }
}`,
			},
			{
				Name: "Pool with dynamic origins block (converted to for expression)",
				Input: `locals {
  origin_configs = [
    {
      name    = "origin-0"
      address = "192.0.2.1"
    },
    {
      name    = "origin-1"
      address = "192.0.2.2"
    }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  dynamic "origins" {
    for_each = local.origin_configs
    content {
      name    = origins.value.name
      address = origins.value.address
      enabled = true
    }
  }
}`,
				Expected: `locals {
  origin_configs = [
    {
      name    = "origin-0"
      address = "192.0.2.1"
    },
    {
      name    = "origin-1"
      address = "192.0.2.2"
    }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [for origins in local.origin_configs : {
    name    = origins.name
    address = origins.address
    enabled = true
  }]
}`,
			},
			{
				Name: "Pool with dynamic origins and static load_shedding",
				Input: `locals {
  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
  }]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  dynamic "origins" {
    for_each = local.origins
    content {
      name    = origins.value.name
      address = origins.value.address
    }
  }

  load_shedding {
    default_percent = 50
    default_policy  = "random"
  }
}`,
				Expected: `locals {
  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
  }]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [for origins in local.origins : {
    name    = origins.name
    address = origins.address
  }]
  load_shedding = {
    default_percent = 50
    default_policy  = "random"
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Basic pool state",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "pool-123",
    "account_id": "account-123",
    "name": "example-pool",
    "origins": [
      {
        "name": "origin-1",
        "address": "192.0.2.1",
        "enabled": true
      }
    ]
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "pool-123",
    "account_id": "account-123",
    "name": "example-pool",
    "origins": [
      {
        "name": "origin-1",
        "address": "192.0.2.1",
        "enabled": true
      }
    ]
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
