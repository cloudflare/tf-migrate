package managed_transforms

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		t.Run("BasicTransformations", testBasicConfig)
		t.Run("MultipleBlocks", testMultipleBlocks)
		t.Run("EdgeCases", testConfigEdgeCases)
		t.Run("MultipleResources", testMultipleResources)
	})
}

func TestV4ToV5TransformationState_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

func TestUsesProviderStateUpgrader(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	if got := migrator.(*V4ToV5Migrator).UsesProviderStateUpgrader(); !got {
		t.Errorf("UsesProviderStateUpgrader() = %v, want true", got)
	}
}

func testBasicConfig(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Minimal resource - no headers",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id                  = "abc123"
  managed_request_headers  = []
  managed_response_headers = []
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
		{
			Name: "Single request header",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_request_headers {
    id      = "add_true_client_ip_headers"
    enabled = true
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = []
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
		{
			Name: "Single response header",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_response_headers {
    id      = "remove_x-powered-by_header"
    enabled = true
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = []
  managed_response_headers = [{
    id      = "remove_x-powered-by_header"
    enabled = true
  }]
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
		{
			Name: "One of each header type",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_request_headers {
    id      = "add_true_client_ip_headers"
    enabled = true
  }

  managed_response_headers {
    id      = "add_security_headers"
    enabled = false
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = [{
    id      = "add_security_headers"
    enabled = false
  }]
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testMultipleBlocks(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Multiple request headers",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_request_headers {
    id      = "add_true_client_ip_headers"
    enabled = true
  }

  managed_request_headers {
    id      = "add_visitor_location_headers"
    enabled = false
  }

  managed_request_headers {
    id      = "add_bot_protection_headers"
    enabled = true
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }, {
    id      = "add_visitor_location_headers"
    enabled = false
  }, {
    id      = "add_bot_protection_headers"
    enabled = true
  }]
  managed_response_headers = []
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
		{
			Name: "Multiple response headers",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_response_headers {
    id      = "remove_x-powered-by_header"
    enabled = true
  }

  managed_response_headers {
    id      = "add_security_headers"
    enabled = false
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = []
  managed_response_headers = [{
    id      = "remove_x-powered-by_header"
    enabled = true
  }, {
    id      = "add_security_headers"
    enabled = false
  }]
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
		{
			Name: "Multiple headers of both types with mixed enabled states",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_request_headers {
    id      = "add_true_client_ip_headers"
    enabled = true
  }

  managed_request_headers {
    id      = "add_visitor_location_headers"
    enabled = false
  }

  managed_response_headers {
    id      = "remove_x-powered-by_header"
    enabled = true
  }

  managed_response_headers {
    id      = "add_security_headers"
    enabled = false
  }

  managed_response_headers {
    id      = "remove_server_header"
    enabled = true
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }, {
    id      = "add_visitor_location_headers"
    enabled = false
  }]
  managed_response_headers = [{
    id      = "remove_x-powered-by_header"
    enabled = true
  }, {
    id      = "add_security_headers"
    enabled = false
  }, {
    id      = "remove_server_header"
    enabled = true
  }]
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testConfigEdgeCases(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Boolean false values preserved",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_request_headers {
    id      = "header_1"
    enabled = false
  }

  managed_request_headers {
    id      = "header_2"
    enabled = false
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = [{
    id      = "header_1"
    enabled = false
  }, {
    id      = "header_2"
    enabled = false
  }]
  managed_response_headers = []
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
		{
			Name: "Headers with special characters in IDs",
			Input: `resource "cloudflare_managed_headers" "example" {
  zone_id = "abc123"

  managed_request_headers {
    id      = "add_x-custom-header"
    enabled = true
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "example" {
  zone_id = "abc123"

  managed_request_headers = [{
    id      = "add_x-custom-header"
    enabled = true
  }]
  managed_response_headers = []
}

moved {
  from = cloudflare_managed_headers.example
  to   = cloudflare_managed_transforms.example
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testMultipleResources(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "Two resources in same file",
			Input: `resource "cloudflare_managed_headers" "zone1" {
  zone_id = "zone-1"

  managed_request_headers {
    id      = "add_true_client_ip_headers"
    enabled = true
  }
}

resource "cloudflare_managed_headers" "zone2" {
  zone_id = "zone-2"

  managed_response_headers {
    id      = "remove_x-powered-by_header"
    enabled = true
  }
}`,
			Expected: `resource "cloudflare_managed_transforms" "zone1" {
  zone_id = "zone-1"

  managed_request_headers = [{
    id      = "add_true_client_ip_headers"
    enabled = true
  }]
  managed_response_headers = []
}

moved {
  from = cloudflare_managed_headers.zone1
  to   = cloudflare_managed_transforms.zone1
}

resource "cloudflare_managed_transforms" "zone2" {
  zone_id = "zone-2"

  managed_request_headers = []
  managed_response_headers = [{
    id      = "remove_x-powered-by_header"
    enabled = true
  }]
}

moved {
  from = cloudflare_managed_headers.zone2
  to   = cloudflare_managed_transforms.zone2
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

