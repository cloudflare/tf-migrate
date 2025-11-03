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
				Name: "basic snippet migration",
				Input: `resource "cloudflare_snippet" "test" {
  zone_id     = "abc123"
  name        = "test_snippet"
  main_module = "main.js"
  files {
    name    = "main.js"
    content = "export default {async fetch(request) {return fetch(request)}};"
  }
}`,
				Expected: `resource "cloudflare_snippet" "test" {
  zone_id      = "abc123"
  snippet_name = "test_snippet"
  metadata = {
    main_module = "main.js"
  }
  files = [
    {
      name    = "main.js"
      content = "export default {async fetch(request) {return fetch(request)}};"
    }
  ]
}`,
			},
			{
				Name: "multiple files migration",
				Input: `resource "cloudflare_snippet" "test" {
  zone_id     = "abc123"
  name        = "test_snippet"
  main_module = "main.js"
  files {
    name    = "main.js"
    content = "import {helper} from './helper.js';"
  }
  files {
    name    = "helper.js"
    content = "export function helper() {}"
  }
}`,
				Expected: `resource "cloudflare_snippet" "test" {
  zone_id      = "abc123"
  snippet_name = "test_snippet"
  metadata = {
    main_module = "main.js"
  }
  files = [
    {
      name    = "main.js"
      content = "import {helper} from './helper.js';"
    },
    {
      name    = "helper.js"
      content = "export function helper() {}"
    }
  ]
}`,
			},
			{
				Name: "snippet with heredoc content",
				Input: `resource "cloudflare_snippet" "test" {
  zone_id     = "abc123"
  name        = "test_snippet"
  main_module = "main.js"
  files {
    name    = "main.js"
    content = <<-EOF
      export default {
        async fetch(request) {
          return fetch(request);
        }
      }
    EOF
  }
}`,
				Expected: `resource "cloudflare_snippet" "test" {
  zone_id      = "abc123"
  snippet_name = "test_snippet"
  metadata = {
    main_module = "main.js"
  }
  files = [
    {
      name    = "main.js"
      content = <<-EOF
      export default {
        async fetch(request) {
          return fetch(request);
        }
      }
    EOF
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "rename name to snippet_name",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "name": "test_snippet",
    "main_module": "main.js"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "snippet_name": "test_snippet",
    "metadata": {
      "main_module": "main.js"
    }
  }
}`,
			},
			{
				Name: "move main_module to metadata",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "snippet_name": "test_snippet",
    "main_module": "index.js"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "snippet_name": "test_snippet",
    "metadata": {
      "main_module": "index.js"
    }
  }
}`,
			},
			{
				Name: "handle both transformations",
				Input: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "name": "my_snippet",
    "main_module": "worker.js"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "zone_id": "abc123",
    "snippet_name": "my_snippet",
    "metadata": {
      "main_module": "worker.js"
    }
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
