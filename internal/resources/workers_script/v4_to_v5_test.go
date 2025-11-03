package workers_script

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "name to script_name",
				Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"
}`,
				Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"
  script_name = "my-worker"
}`,
			},
			{
				Name: "worker_script (singular) with name",
				Input: `resource "cloudflare_worker_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"
}`,
				Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"
  script_name = "my-worker"
}`,
			},
			{
				Name: "plain_text_binding",
				Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  plain_text_binding {
    name = "MY_VAR"
    text = "my-value"
  }
}`,
				Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  script_name = "my-worker"
  bindings = [{
    type = "plain_text"
    name = "MY_VAR"
    text = "my-value"
  }]
}`,
			},
			{
				Name: "multiple binding types",
				Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  plain_text_binding {
    name = "MY_VAR"
    text = "my-value"
  }

  kv_namespace_binding {
    name         = "MY_KV"
    namespace_id = "abc123"
  }
}`,
				Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"


  script_name = "my-worker"
  bindings = [{
    type = "plain_text"
    name = "MY_VAR"
    text = "my-value"
  },
  {
    type         = "kv_namespace"
    name         = "MY_KV"
    namespace_id = "abc123"
  }]
}`,
			},
			{
				Name: "d1_database_binding with database_id to id rename",
				Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  d1_database_binding {
    name        = "MY_DB"
    database_id = "db123"
  }
}`,
				Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  script_name = "my-worker"
  bindings = [{
    type = "d1"
    id   = "db123"
    name = "MY_DB"
  }]
}`,
			},
			{
				Name: "hyperdrive_config_binding with binding to name rename",
				Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  hyperdrive_config_binding {
    binding = "HYPERDRIVE"
    id      = "hyperdrive123"
  }
}`,
				Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello World')); });"

  script_name = "my-worker"
  bindings = [{
    type = "hyperdrive"
    name = "HYPERDRIVE"
    id   = "hyperdrive123"
  }]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "name to script_name in state",
				Input: `{
  "attributes": {
    "name": "my-worker",
    "content": "worker code"
  }
}`,
				Expected: `{
  "attributes": {
    "script_name": "my-worker",
    "content": "worker code"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
