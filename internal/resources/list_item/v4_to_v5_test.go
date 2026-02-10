package list_item

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		t.Run("IPListItem", testIPListItemConfig)
		t.Run("HostnameListItem", testHostnameListItemConfig)
		t.Run("RedirectListItem", testRedirectListItemConfig)
	})

	t.Run("StateTransformation_Removed", func(t *testing.T) {
		t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
	})
}

func testIPListItemConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "IP list item should be kept as-is",
			Input: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  ip         = "192.0.2.1"
  comment    = "Test IP"
}`,
			Expected: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  ip         = "192.0.2.1"
  comment    = "Test IP"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testHostnameListItemConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Hostname block should be converted to attribute",
			Input: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  hostname {
    url_hostname = "example.com"
  }
  comment = "Test hostname"
}`,
			Expected: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  hostname = {
    url_hostname = "example.com"
  }
  comment = "Test hostname"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testRedirectListItemConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Redirect block should be converted to attribute with boolean conversion",
			Input: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  redirect {
    source_url            = "https://example.com/old"
    target_url            = "https://example.com/new"
    status_code           = 301
    include_subdomains    = "enabled"
    subpath_matching      = "disabled"
    preserve_query_string = "enabled"
    preserve_path_suffix  = "disabled"
  }
  comment = "Test redirect"
}`,
			Expected: `resource "cloudflare_list_item" "test_item" {
  account_id = "abc123"
  list_id    = cloudflare_list.test.id
  redirect = {
    source_url            = "https://example.com/old"
    target_url            = "https://example.com/new"
    include_subdomains    = true
    subpath_matching      = false
    preserve_query_string = true
    preserve_path_suffix  = false
    status_code           = 301
  }
  comment = "Test redirect"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
