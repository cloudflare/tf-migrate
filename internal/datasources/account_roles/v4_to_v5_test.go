package account_roles

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/stretchr/testify/assert"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "account_id only",
			Input: `data "cloudflare_account_roles" "all" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			Expected: `data "cloudflare_account_roles" "all" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
		},
		{
			Name: "account_id as variable reference",
			Input: `data "cloudflare_account_roles" "all" {
  account_id = var.account_id
}`,
			Expected: `data "cloudflare_account_roles" "all" {
  account_id = var.account_id
}`,
		},
		{
			Name: "multiple datasources",
			Input: `data "cloudflare_account_roles" "staging" {
  account_id = var.staging_account_id
}

data "cloudflare_account_roles" "prod" {
  account_id = var.prod_account_id
}`,
			Expected: `data "cloudflare_account_roles" "staging" {
  account_id = var.staging_account_id
}

data "cloudflare_account_roles" "prod" {
  account_id = var.prod_account_id
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestGetAttributeRenames(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	renames := migrator.GetAttributeRenames()

	assert.Len(t, renames, 1)
	assert.Equal(t, "data.cloudflare_account_roles", renames[0].ResourceType)
	assert.Equal(t, "roles", renames[0].OldAttribute)
	assert.Equal(t, "result", renames[0].NewAttribute)
}

func TestCanHandle(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	assert.True(t, migrator.CanHandle("data.cloudflare_account_roles"))
	assert.False(t, migrator.CanHandle("cloudflare_account_roles"))
	assert.False(t, migrator.CanHandle("data.cloudflare_accounts"))
	assert.False(t, migrator.CanHandle("data.cloudflare_account_role"))
}

func TestGetResourceType(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	assert.Equal(t, "cloudflare_account_roles", migrator.GetResourceType())
}

func TestImplementsInterfaces(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	var _ transform.ResourceTransformer = migrator
	var _ transform.ResourceRenamer = migrator
	var _ transform.AttributeRenamer = migrator
}
