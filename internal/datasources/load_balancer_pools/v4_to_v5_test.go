package load_balancer_pools

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic datasource - no filter",
			Input: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			Expected: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
		},
		{
			Name: "with filter block - removed in v5",
			Input: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  filter {
    name = "example-.*"
  }
}`,
			Expected: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

}`,
		},
		{
			Name: "with empty filter block",
			Input: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  filter {
  }
}`,
			Expected: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"

}`,
		},
		{
			Name: "with variable references",
			Input: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = var.cloudflare_account_id

  filter {
    name = "prod-.*"
  }
}`,
			Expected: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = var.cloudflare_account_id

}`,
		},
		{
			Name: "multiple datasources in one file",
			Input: `
data "cloudflare_load_balancer_pools" "all" {
  account_id = "abc123"
}

data "cloudflare_load_balancer_pools" "filtered" {
  account_id = "abc123"

  filter {
    name = "example-.*"
  }
}`,
			Expected: `
data "cloudflare_load_balancer_pools" "all" {
  account_id = "abc123"
}

data "cloudflare_load_balancer_pools" "filtered" {
  account_id = "abc123"

}`,
		},
		{
			Name: "with local reference",
			Input: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = local.account_id

  filter {
    name = local.pool_name_pattern
  }
}`,
			Expected: `
data "cloudflare_load_balancer_pools" "example" {
  account_id = local.account_id

}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// TestStateTransformation_Removed is a marker indicating state transformation tests were removed.
// State migration is now handled by the provider's StateUpgraders.
func TestStateTransformation_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

func TestMigratorMethods(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("GetResourceType returns correct type", func(t *testing.T) {
		resourceType := migrator.GetResourceType()
		assert.Equal(t, "cloudflare_load_balancer_pools", resourceType)
	})

	t.Run("CanHandle matches correct datasource type", func(t *testing.T) {
		assert.True(t, migrator.CanHandle("data.cloudflare_load_balancer_pools"))
		assert.False(t, migrator.CanHandle("cloudflare_load_balancer_pools"))
		assert.False(t, migrator.CanHandle("data.cloudflare_zone"))
		assert.False(t, migrator.CanHandle("cloudflare_load_balancer_pool"))
	})

	t.Run("GetAttributeRenames returns correct mapping", func(t *testing.T) {
		// Cast to concrete type to access GetAttributeRenames
		concreteMigrator := migrator.(*V4ToV5Migrator)
		renames := concreteMigrator.GetAttributeRenames()
		assert.Len(t, renames, 1)
		assert.Equal(t, "data.cloudflare_load_balancer_pools", renames[0].ResourceType)
		assert.Equal(t, "pools", renames[0].OldAttribute)
		assert.Equal(t, "result", renames[0].NewAttribute)
	})

	t.Run("Preprocess returns content unchanged", func(t *testing.T) {
		input := "some terraform config content"
		output := migrator.Preprocess(input)
		assert.Equal(t, input, output)
	})
}

