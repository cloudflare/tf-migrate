package leaked_credential_check_rule

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic resource with all fields",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}`,
				// Import block is generated because v4 provider had a bug where detection_id
				// was not stored in state. The zone_id is a literal 32-char hex, so it is
				// embedded in the import block ID.
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.example
  id = "0da42c8d2132a9ddaf714f9e7c920711/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "example" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}`,
			},
			{
				Name: "Resource with only zone_id (both optional)",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.minimal
  id = "0da42c8d2132a9ddaf714f9e7c920711/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "minimal" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			},
			{
				Name: "Resource with only username",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "username_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
}`,
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.username_only
  id = "0da42c8d2132a9ddaf714f9e7c920711/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "username_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
}`,
			},
			{
				Name: "Resource with only password",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "password_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  password = "http.request.body.form.password"
}`,
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.password_only
  id = "0da42c8d2132a9ddaf714f9e7c920711/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "password_only" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  password = "http.request.body.form.password"
}`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "first" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}

resource "cloudflare_leaked_credential_check_rule" "second" {
  zone_id  = "28fea702d1075b10ba9c8620b86218ec"
  username = "http.request.body.user"
}`,
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.first
  id = "0da42c8d2132a9ddaf714f9e7c920711/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "first" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}
import {
  to = cloudflare_leaked_credential_check_rule.second
  id = "28fea702d1075b10ba9c8620b86218ec/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "second" {
  zone_id  = "28fea702d1075b10ba9c8620b86218ec"
  username = "http.request.body.user"
}`,
			},
			{
				Name: "Resource with complex ruleset expressions",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "complex" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "lookup_json_string(http.request.body.raw, \"credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"credentials.password\")"
}`,
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.complex
  id = "0da42c8d2132a9ddaf714f9e7c920711/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "complex" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "lookup_json_string(http.request.body.raw, \"credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"credentials.password\")"
}`,
			},
			{
				Name: "Resource with variable zone_id (placeholder fallback)",
				Input: `
resource "cloudflare_leaked_credential_check_rule" "var_zone" {
  zone_id  = var.zone_id
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}`,
				// When zone_id is a variable reference, placeholder <zone_id> is used
				Expected: `import {
  to = cloudflare_leaked_credential_check_rule.var_zone
  id = "<zone_id>/<detection_id>"
}
resource "cloudflare_leaked_credential_check_rule" "var_zone" {
  zone_id  = var.zone_id
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
