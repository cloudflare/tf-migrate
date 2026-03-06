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
				Expected: `resource "cloudflare_leaked_credential_check_rule" "example" {
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
				Expected: `resource "cloudflare_leaked_credential_check_rule" "minimal" {
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
				Expected: `resource "cloudflare_leaked_credential_check_rule" "username_only" {
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
				Expected: `resource "cloudflare_leaked_credential_check_rule" "password_only" {
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
				Expected: `resource "cloudflare_leaked_credential_check_rule" "first" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "http.request.body.form.username"
  password = "http.request.body.form.password"
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
				Expected: `resource "cloudflare_leaked_credential_check_rule" "complex" {
  zone_id  = "0da42c8d2132a9ddaf714f9e7c920711"
  username = "lookup_json_string(http.request.body.raw, \"credentials.username\")"
  password = "lookup_json_string(http.request.body.raw, \"credentials.password\")"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
