package accounts

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
			Name: "name filter only",
			Input: `data "cloudflare_accounts" "test" {
  name = "My Company"
}`,
			Expected: `data "cloudflare_accounts" "test" {
  name = "My Company"
}`,
		},
		{
			Name: "empty block (no filter)",
			Input: `data "cloudflare_accounts" "test" {
}`,
			Expected: `data "cloudflare_accounts" "test" {
}`,
		},
		{
			Name: "variable reference preserved",
			Input: `data "cloudflare_accounts" "test" {
  name = var.account_name
}`,
			Expected: `data "cloudflare_accounts" "test" {
  name = var.account_name
}`,
		},
		{
			Name: "local reference preserved",
			Input: `data "cloudflare_accounts" "test" {
  name = local.account_name
}`,
			Expected: `data "cloudflare_accounts" "test" {
  name = local.account_name
}`,
		},
		{
			Name: "multiple datasources",
			Input: `data "cloudflare_accounts" "all" {
}

data "cloudflare_accounts" "filtered" {
  name = "Production"
}`,
			Expected: `data "cloudflare_accounts" "all" {
}

data "cloudflare_accounts" "filtered" {
  name = "Production"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestGetAttributeRenames(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	renames := migrator.GetAttributeRenames()

	assert.Len(t, renames, 1)
	assert.Equal(t, "data.cloudflare_accounts", renames[0].ResourceType)
	assert.Equal(t, "accounts", renames[0].OldAttribute)
	assert.Equal(t, "result", renames[0].NewAttribute)
}

func TestGetResourceRename(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	oldType, newType := migrator.GetResourceRename()

	assert.Equal(t, "data.cloudflare_accounts", oldType)
	assert.Equal(t, "data.cloudflare_accounts", newType)
}

func TestCanHandle(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	assert.True(t, migrator.CanHandle("data.cloudflare_accounts"))
	assert.False(t, migrator.CanHandle("cloudflare_accounts"))
	assert.False(t, migrator.CanHandle("data.cloudflare_account"))
	assert.False(t, migrator.CanHandle("data.cloudflare_zones"))
}

func TestGetResourceType(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	assert.Equal(t, "cloudflare_accounts", migrator.GetResourceType())
}

func TestImplementsInterfaces(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	// Verify ResourceTransformer interface
	var _ transform.ResourceTransformer = migrator

	// Verify ResourceRenamer interface
	var _ transform.ResourceRenamer = migrator

	// Verify AttributeRenamer interface
	var _ transform.AttributeRenamer = migrator
}
