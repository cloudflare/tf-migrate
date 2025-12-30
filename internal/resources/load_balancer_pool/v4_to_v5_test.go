package load_balancer_pool

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
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
				Name: "Dynamic origins block with headers",
				Input: `locals {
  origin_configs = [
    {
      name    = "origin-0"
      address = "192.0.2.1"
      host    = "test0.example.com"
    },
    {
      name    = "origin-1"
      address = "192.0.2.2"
      host    = "test1.example.com"
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
      name    = "origin-0"
      address = "192.0.2.1"
      host    = "test0.example.com"
    },
    {
      name    = "origin-1"
      address = "192.0.2.2"
      host    = "test1.example.com"
    }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [for value in local.origin_configs : {
    name    = value.name
    address = value.address
    enabled = true
    header  = { host  =  [value.host]  }
  }]
}`,
			},
			{
				Name: "Dynamic origins block without headers",
				Input: `locals {
  origin_list = [
    { name = "origin-1", address = "192.0.2.1" },
    { name = "origin-2", address = "192.0.2.2" }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  dynamic "origins" {
    for_each = local.origin_list
    content {
      name    = origins.value.name
      address = origins.value.address
      weight  = 1.0
    }
  }
}`,
				Expected: `locals {
  origin_list = [
    { name = "origin-1", address = "192.0.2.1" },
    { name = "origin-2", address = "192.0.2.2" }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [for value in local.origin_list : {
    name    = value.name
    address = value.address
    weight  = 1.0
  }]
}`,
			},
			{
				Name: "Origins with header blocks",
				Input: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true

    header {
      header = "Host"
      values = ["test1.example.com"]
    }
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = false

    header {
      header = "Host"
      values = ["test2.example.com", "test2-alt.example.com"]
    }
  }
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true
    header  = { host  =  ["test1.example.com"]  }
  }, {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = false
    header  = { host  =  ["test2.example.com", "test2-alt.example.com"]  }
  }]
}`,
			},
			{
				Name: "Pool with origin_steering block",
				Input: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    weight  = 0.8
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    weight  = 0.2
  }

  origin_steering {
    policy = "least_connections"
  }
}`,
				Expected: `resource "cloudflare_load_balancer_pool" "example" {
  account_id = "abc123"
  name       = "example-pool"

  origins = [{
    name    = "origin-1"
    address = "192.0.2.1"
    weight  = 0.8
  }, {
    name    = "origin-2"
    address = "192.0.2.2"
    weight  = 0.2
  }]
  origin_steering = {
    policy = "least_connections"
  }
}`,
			},
			{
				Name: "Complex pool with all features",
				Input: `locals {
  backends = [
    { name = "backend-1", address = "192.0.2.1", host = "api1.example.com" },
    { name = "backend-2", address = "192.0.2.2", host = "api2.example.com" }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id      = "abc123"
  name            = "example-pool"
  enabled         = true
  minimum_origins = 1
  description     = "Example pool with all features"

  dynamic "origins" {
    for_each = local.backends
    content {
      name    = origins.value.name
      address = origins.value.address
      enabled = true
      weight  = 1.0

      header {
        header = "Host"
        values = [origins.value.host]
      }
    }
  }

  load_shedding {
    default_percent = 55
    default_policy  = "random"
    session_percent = 12
    session_policy  = "hash"
  }

  origin_steering {
    policy = "random"
  }

  check_regions = ["WEU", "ENAM"]
}`,
				Expected: `locals {
  backends = [
    { name = "backend-1", address = "192.0.2.1", host = "api1.example.com" },
    { name = "backend-2", address = "192.0.2.2", host = "api2.example.com" }
  ]
}

resource "cloudflare_load_balancer_pool" "example" {
  account_id      = "abc123"
  name            = "example-pool"
  enabled         = true
  minimum_origins = 1
  description     = "Example pool with all features"

  check_regions = ["WEU", "ENAM"]
  origins = [for value in local.backends : {
    name    = value.name
    address = value.address
    enabled = true
    weight  = 1.0
    header  = { host  =  [value.host]  }
  }]
  load_shedding = {
    default_percent = 55
    default_policy  = "random"
    session_percent = 12
    session_policy  = "hash"
  }
  origin_steering = {
    policy = "random"
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
			{
				Name: "Pool state with origins containing headers (array to object)",
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
        "enabled": true,
        "header": [
          {
            "header": "Host",
            "values": ["test1.example.com"]
          }
        ]
      },
      {
        "name": "origin-2",
        "address": "192.0.2.2",
        "enabled": false,
        "header": []
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
        "enabled": true,
        "header": {
          "header": "Host",
          "values": ["test1.example.com"]
        }
      },
      {
        "name": "origin-2",
        "address": "192.0.2.2",
        "enabled": false,
        "header": {}
      }
    ]
  }
}`,
			},
			{
				Name: "Pool state with load_shedding (array to object)",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "pool-123",
    "account_id": "account-123",
    "name": "example-pool",
    "origins": [
      {
        "name": "origin-1",
        "address": "192.0.2.1"
      }
    ],
    "load_shedding": [
      {
        "default_percent": 55,
        "default_policy": "random",
        "session_percent": 12,
        "session_policy": "hash"
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
        "address": "192.0.2.1"
      }
    ],
    "load_shedding": {
      "default_percent": 55,
      "default_policy": "random",
      "session_percent": 12,
      "session_policy": "hash"
    }
  }
}`,
			},
			{
				Name: "Pool state with origin_steering (array to object)",
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
        "weight": 0.8
      }
    ],
    "origin_steering": [
      {
        "policy": "least_connections"
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
        "weight": 0.8
      }
    ],
    "origin_steering": {
      "policy": "least_connections"
    }
  }
}`,
			},
			{
				Name: "Pool state with empty load_shedding (array to null)",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "pool-123",
    "account_id": "account-123",
    "name": "example-pool",
    "origins": [
      {
        "name": "origin-1",
        "address": "192.0.2.1"
      }
    ],
    "load_shedding": [],
    "origin_steering": []
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
        "address": "192.0.2.1"
      }
    ],
    "load_shedding": null,
    "origin_steering": null
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}

// TestDynamicOriginsBlockTransformation tests the transformation of dynamic origins blocks
// with detailed assertions on the output format
func TestDynamicOriginsBlockTransformation(t *testing.T) {
	input := `
locals {
  origin_configs = [
    {
      name    = "origin-0"
      address = "192.0.2.1"
      host    = "test0.example.com"
    },
    {
      name    = "origin-1"
      address = "192.0.2.2"
      host    = "test1.example.com"
    }
  ]
}

resource "cloudflare_load_balancer_pool" "test" {
  account_id = "abc123"
  name = "test"

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
}
`

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse input HCL")

	// Find the resource block
	var resourceBlock *hclwrite.Block
	for _, block := range file.Body().Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 && block.Labels()[0] == "cloudflare_load_balancer_pool" {
			resourceBlock = block
			break
		}
	}

	require.NotNil(t, resourceBlock, "Resource block not found")

	// Apply the migrator transformation
	migrator := &V4ToV5Migrator{}
	ctx := &transform.Context{}
	_, err := migrator.TransformConfig(ctx, resourceBlock)
	require.NoError(t, err, "TransformConfig failed")

	// Check the output
	output := string(file.Bytes())
	t.Logf("Output:\n%s", output)

	// Verify that:
	// 1. The dynamic block is gone
	assert.NotContains(t, output, `dynamic "origins"`, "Dynamic block should be removed")

	// 2. The origins attribute exists with a for expression
	assert.Contains(t, output, "origins = [for", "Should have a for expression for origins")

	// 3. The iterator reference is updated (origins.value -> value)
	assert.Contains(t, output, "value.name", "Iterator reference should be updated to value.name")
	assert.Contains(t, output, "value.address", "Iterator reference should be updated to value.address")
	assert.Contains(t, output, "value.host", "Iterator reference should be updated to value.host")

	// 4. The header is transformed to object format
	assert.Contains(t, output, "header  =", "Header should be transformed to object")
	assert.Contains(t, output, "host  =", "Header should have host attribute")

	// 5. The iterator name "origins.value" should not appear
	assert.NotContains(t, output, "origins.value", "Old iterator reference should be removed")
}

// TestStaticOriginsWithHeaderBlock tests the transformation of static origins blocks with headers
func TestStaticOriginsWithHeaderBlock(t *testing.T) {
	input := `
resource "cloudflare_load_balancer_pool" "test" {
  account_id = "abc123"
  name = "test"

  origins {
    name    = "origin-1"
    address = "192.0.2.1"
    enabled = true

    header {
      header = "Host"
      values = ["test1.example.com"]
    }
  }

  origins {
    name    = "origin-2"
    address = "192.0.2.2"
    enabled = false

    header {
      header = "Host"
      values = ["test2.example.com"]
    }
  }
}
`

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse input HCL")

	// Find the resource block
	var resourceBlock *hclwrite.Block
	for _, block := range file.Body().Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 && block.Labels()[0] == "cloudflare_load_balancer_pool" {
			resourceBlock = block
			break
		}
	}

	require.NotNil(t, resourceBlock, "Resource block not found")

	// Apply the migrator transformation
	migrator := &V4ToV5Migrator{}
	ctx := &transform.Context{}
	_, err := migrator.TransformConfig(ctx, resourceBlock)
	require.NoError(t, err, "TransformConfig failed")

	// Check the output
	output := string(file.Bytes())
	t.Logf("Output:\n%s", output)

	// Verify that:
	// 1. origins is an array attribute, not blocks
	assert.Contains(t, output, "origins = [", "Origins should be an array attribute")
	assert.NotContains(t, output, "origins {", "Origins blocks should be removed")

	// 2. The header is transformed to object format in each origin
	assert.Contains(t, output, "header  =", "Header should be transformed to object")
	assert.Contains(t, output, "host  =", "Header should have host attribute")

	// 3. No more header blocks
	assert.NotContains(t, output, "header {", "Header blocks should be removed")
}
