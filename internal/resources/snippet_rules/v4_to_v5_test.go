package snippet_rules

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "single rule with all fields",
			Input: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = "zone123"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/test\""
    snippet_name = "example_snippet"
    description  = "Test rule"
  }
}`,
			Expected: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = "zone123"

  rules = [
    {
      enabled      = true
      expression   = "http.request.uri.path eq \"/test\""
      snippet_name = "example_snippet"
      description  = "Test rule"
    }
  ]
}`,
		},
		{
			Name: "multiple rules",
			Input: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = "zone123"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/api\""
    snippet_name = "api_snippet"
  }

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/admin\""
    snippet_name = "admin_snippet"
    description  = "Admin rule"
  }
}`,
			Expected: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = "zone123"

  rules = [
    {
      enabled      = true
      expression   = "http.request.uri.path eq \"/api\""
      snippet_name = "api_snippet"
    },
    {
      enabled      = false
      expression   = "http.request.uri.path eq \"/admin\""
      snippet_name = "admin_snippet"
      description  = "Admin rule"
    }
  ]
}`,
		},
		{
			Name: "rule with minimal fields",
			Input: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = "zone123"

  rules {
    expression   = "http.request.uri.path eq \"/test\""
    snippet_name = "test_snippet"
  }
}`,
			Expected: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = "zone123"

  rules = [
    {
      expression   = "http.request.uri.path eq \"/test\""
      snippet_name = "test_snippet"
      enabled      = true
    }
  ]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestConfigTransformation_MultipleResources(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "multiple snippet_rules resources in one file",
			Input: `
resource "cloudflare_snippet_rules" "first" {
  zone_id = "zone123"

  rules {
    enabled      = true
    expression   = "http.request.uri.path eq \"/first\""
    snippet_name = "first_snippet"
  }
}

resource "cloudflare_snippet_rules" "second" {
  zone_id = "zone456"

  rules {
    enabled      = false
    expression   = "http.request.uri.path eq \"/second\""
    snippet_name = "second_snippet"
  }
}`,
			Expected: `
resource "cloudflare_snippet_rules" "first" {
  zone_id = "zone123"

  rules = [
    {
      enabled      = true
      expression   = "http.request.uri.path eq \"/first\""
      snippet_name = "first_snippet"
    }
  ]
}

resource "cloudflare_snippet_rules" "second" {
  zone_id = "zone456"

  rules = [
    {
      enabled      = false
      expression   = "http.request.uri.path eq \"/second\""
      snippet_name = "second_snippet"
    }
  ]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestConfigTransformation_WithVariables(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "rules with variable references",
			Input: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = var.zone_id

  rules {
    enabled      = var.enable_rule
    expression   = var.rule_expression
    snippet_name = var.snippet_name
    description  = var.description
  }
}`,
			Expected: `
resource "cloudflare_snippet_rules" "example" {
  zone_id = var.zone_id

  rules = [
    {
      enabled      = var.enable_rule
      expression   = var.rule_expression
      snippet_name = var.snippet_name
      description  = var.description
    }
  ]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "single rule with all fields",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "zone123",
    "rules": [
      {
        "enabled": true,
        "expression": "http.request.uri.path eq \"/test\"",
        "snippet_name": "example_snippet",
        "description": "Test rule"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone123",
    "rules": [
      {
        "enabled": true,
        "expression": "http.request.uri.path eq \"/test\"",
        "snippet_name": "example_snippet",
        "description": "Test rule"
      }
    ]
  }
}`,
		},
		{
			Name: "multiple rules",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "zone123",
    "rules": [
      {
        "enabled": true,
        "expression": "http.request.uri.path eq \"/api\"",
        "snippet_name": "api_snippet"
      },
      {
        "enabled": false,
        "expression": "http.request.uri.path eq \"/admin\"",
        "snippet_name": "admin_snippet",
        "description": "Admin rule"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone123",
    "rules": [
      {
        "enabled": true,
        "expression": "http.request.uri.path eq \"/api\"",
        "snippet_name": "api_snippet"
      },
      {
        "enabled": false,
        "expression": "http.request.uri.path eq \"/admin\"",
        "snippet_name": "admin_snippet",
        "description": "Admin rule"
      }
    ]
  }
}`,
		},
		{
			Name: "empty rules array",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "zone123",
    "rules": []
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone123"
  }
}`,
		},
		{
			Name: "minimal state without rules",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "zone123"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone123"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}

func TestStateTransformation_EdgeCases(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "state without attributes (invalid instance)",
			Input: `{
  "schema_version": 1
}`,
			Expected: `{
  "schema_version": 0
}`,
		},
		{
			Name: "state with null rules",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "zone_id": "zone123",
    "rules": null
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "zone123",
    "rules": null
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}

func TestMigratorMethods(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test CanHandle
	if !migrator.CanHandle("cloudflare_snippet_rules") {
		t.Error("CanHandle() should return true for cloudflare_snippet_rules")
	}
	if migrator.CanHandle("cloudflare_other_resource") {
		t.Error("CanHandle() should return false for other resources")
	}

	// Test Preprocess (should return input unchanged)
	input := "some content"
	if output := migrator.Preprocess(input); output != input {
		t.Errorf("Preprocess() modified content: got %v, want %v", output, input)
	}

	// Test GetResourceType (accessed via cast)
	if m, ok := migrator.(*V4ToV5Migrator); ok {
		if resourceType := m.GetResourceType(); resourceType != "cloudflare_snippet_rules" {
			t.Errorf("GetResourceType() = %v, want cloudflare_snippet_rules", resourceType)
		}

		oldName, newName := m.GetResourceRename()
		if oldName != "cloudflare_snippet_rules" || newName != "cloudflare_snippet_rules" {
			t.Errorf("GetResourceRename() = (%v, %v), want (cloudflare_snippet_rules, cloudflare_snippet_rules)", oldName, newName)
		}
	}
}
