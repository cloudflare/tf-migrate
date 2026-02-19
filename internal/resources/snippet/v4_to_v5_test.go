package snippet

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource with single file",
				Input: `resource "cloudflare_snippet" "example" {
  zone_id     = "abc123"
  name        = "my_snippet"
  main_module = "main.js"

  files {
    name    = "main.js"
    content = "console.log('hello');"
  }
}`,
				Expected: `resource "cloudflare_snippet" "example" {
  zone_id = "abc123"

  snippet_name = "my_snippet"
  files = [{
    name    = "main.js"
    content = "console.log('hello');"
  }]
  metadata = {
    main_module = "main.js"
  }
}`,
			},
			{
				Name: "Resource with multiple files",
				Input: `resource "cloudflare_snippet" "multi" {
  zone_id     = "zone123"
  name        = "multi_file_snippet"
  main_module = "index.js"

  files {
    name    = "index.js"
    content = "export default { fetch() {} }"
  }

  files {
    name    = "utils.js"
    content = "export function helper() {}"
  }

  files {
    name    = "config.json"
    content = "{\"key\": \"value\"}"
  }
}`,
				Expected: `resource "cloudflare_snippet" "multi" {
  zone_id = "zone123"

  snippet_name = "multi_file_snippet"
  files = [{
    name    = "index.js"
    content = "export default { fetch() {} }"
  }, {
    name    = "utils.js"
    content = "export function helper() {}"
  }, {
    name    = "config.json"
    content = "{\"key\": \"value\"}"
  }]
  metadata = {
    main_module = "index.js"
  }
}`,
			},
			{
				Name: "Resource with file without content",
				Input: `resource "cloudflare_snippet" "optional_content" {
  zone_id     = "zone456"
  name        = "snippet_no_content"
  main_module = "empty.js"

  files {
    name = "empty.js"
  }
}`,
				Expected: `resource "cloudflare_snippet" "optional_content" {
  zone_id = "zone456"

  snippet_name = "snippet_no_content"
  files = [{
    name = "empty.js"
  }]
  metadata = {
    main_module = "empty.js"
  }
}`,
			},
			{
				Name: "Resource with variable references",
				Input: `resource "cloudflare_snippet" "vars" {
  zone_id     = var.zone_id
  name        = var.snippet_name
  main_module = var.main_module

  files {
    name    = var.file_name
    content = file("${path.module}/snippet.js")
  }
}`,
				Expected: `resource "cloudflare_snippet" "vars" {
  zone_id = var.zone_id

  snippet_name = var.snippet_name
  files = [{
    name    = var.file_name
    content = file("${path.module}/snippet.js")
  }]
  metadata = {
    main_module = var.main_module
  }
}`,
			},
			{
				Name: "Resource with special characters in content",
				Input: `resource "cloudflare_snippet" "special_chars" {
  zone_id     = "zone789"
  name        = "special"
  main_module = "app.js"

  files {
    name    = "app.js"
    content = "const msg = \"Hello, \\\"World\\\"!\"; console.log(msg);"
  }
}`,
				Expected: `resource "cloudflare_snippet" "special_chars" {
  zone_id = "zone789"

  snippet_name = "special"
  files = [{
    name    = "app.js"
    content = "const msg = \"Hello, \\\"World\\\"!\"; console.log(msg);"
  }]
  metadata = {
    main_module = "app.js"
  }
}`,
			},
			{
				Name: "Multiple resources in one config",
				Input: `resource "cloudflare_snippet" "first" {
  zone_id     = "zone1"
  name        = "first_snippet"
  main_module = "index.js"

  files {
    name    = "index.js"
    content = "// first"
  }
}

resource "cloudflare_snippet" "second" {
  zone_id     = "zone2"
  name        = "second_snippet"
  main_module = "app.js"

  files {
    name    = "app.js"
    content = "// second"
  }
}`,
				Expected: `resource "cloudflare_snippet" "first" {
  zone_id = "zone1"

  snippet_name = "first_snippet"
  files = [{
    name    = "index.js"
    content = "// first"
  }]
  metadata = {
    main_module = "index.js"
  }
}

resource "cloudflare_snippet" "second" {
  zone_id = "zone2"

  snippet_name = "second_snippet"
  files = [{
    name    = "app.js"
    content = "// second"
  }]
  metadata = {
    main_module = "app.js"
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}

func TestMigratorInterface(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("CanHandle accepts cloudflare_snippet", func(t *testing.T) {
		if !migrator.CanHandle("cloudflare_snippet") {
			t.Error("CanHandle(cloudflare_snippet) should return true")
		}
	})

	t.Run("CanHandle rejects other resources", func(t *testing.T) {
		if migrator.CanHandle("cloudflare_other_resource") {
			t.Error("CanHandle(cloudflare_other_resource) should return false")
		}
	})

	t.Run("Preprocess returns content unchanged", func(t *testing.T) {
		input := "test content"
		if got := migrator.Preprocess(input); got != input {
			t.Errorf("Preprocess() = %v, want %v", got, input)
		}
	})
}
