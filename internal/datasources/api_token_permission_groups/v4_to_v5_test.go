package api_token_permission_groups

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/stretchr/testify/assert"
)

func TestConfigTransformation(t *testing.T) {
	testhelpers.RunConfigTransformTests(t, []testhelpers.ConfigTestCase{
		{Name: "empty datasource", Input: `data "cloudflare_api_token_permission_groups" "all" {}`, Expected: `data "cloudflare_api_token_permission_groups_list" "all" {}`},
		{Name: "retain meta arguments", Input: `
data "cloudflare_api_token_permission_groups" "scoped" {
  provider = cloudflare.alternate
  for_each = var.accounts
  depends_on = [cloudflare_account.example]
}`, Expected: `
data "cloudflare_api_token_permission_groups_list" "scoped" {
  provider = cloudflare.alternate
  for_each = var.accounts
  depends_on = [cloudflare_account.example]
}`},
	}, NewV4ToV5Migrator())
}

func TestReferenceMetadata(t *testing.T) {
	m := &V4ToV5Migrator{}
	assert.True(t, m.CanHandle("data.cloudflare_api_token_permission_groups"))
	assert.False(t, m.CanHandle("cloudflare_api_token_permission_groups"))
	old, newType := m.GetResourceRename()
	assert.Equal(t, []string{"data.cloudflare_api_token_permission_groups"}, old)
	assert.Equal(t, "data.cloudflare_api_token_permission_groups_list", newType)
	var attributes []string
	for _, ref := range m.GetInvalidAttributeReferences() {
		attributes = append(attributes, ref.Attribute)
		assert.Equal(t, newType, ref.ResourceType)
		assert.Contains(t, ref.Suggestion, "scope")
	}
	assert.ElementsMatch(t, []string{"permissions", "zone", "account", "user", "r2", "id"}, attributes)
	assert.Implements(t, (*transform.InvalidAttributeReferenceDetector)(nil), m)
}
