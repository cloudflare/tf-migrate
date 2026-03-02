package pages_domain

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic field rename domain to name",
				Input: `resource "cloudflare_pages_domain" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "example.com"
}`,
				Expected: `resource "cloudflare_pages_domain" "example" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  name         = "example.com"
}`,
			},
			{
				Name: "Multiple resources",
				Input: `resource "cloudflare_pages_domain" "prod" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "prod.example.com"
}

resource "cloudflare_pages_domain" "staging" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  domain       = "staging.example.com"
}`,
				Expected: `resource "cloudflare_pages_domain" "prod" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  name         = "prod.example.com"
}

resource "cloudflare_pages_domain" "staging" {
  account_id   = "f037e56e89293a057740de681ac9abbe"
  project_name = "my-project"
  name         = "staging.example.com"
}`,
			},
			{
				Name: "With variable reference",
				Input: `resource "cloudflare_pages_domain" "example" {
  account_id   = var.cloudflare_account_id
  project_name = var.project_name
  domain       = var.domain_name
}`,
				Expected: `resource "cloudflare_pages_domain" "example" {
  account_id   = var.cloudflare_account_id
  project_name = var.project_name
  name         = var.domain_name
}`,
			},
			{
				Name: "With project reference",
				Input: `resource "cloudflare_pages_domain" "example" {
  account_id   = cloudflare_account.example.id
  project_name = cloudflare_pages_project.example.name
  domain       = "example.com"
}`,
				Expected: `resource "cloudflare_pages_domain" "example" {
  account_id   = cloudflare_account.example.id
  project_name = cloudflare_pages_project.example.name
  name         = "example.com"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"cloudflare_pages_domain", true},
		{"cloudflare_pages_project", false},
		{"cloudflare_custom_pages", false},
		{"cloudflare_something_else", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	expected := "cloudflare_pages_domain"
	result := migrator.GetResourceType()
	if result != expected {
		t.Errorf("GetResourceType() = %q, expected %q", result, expected)
	}
}
