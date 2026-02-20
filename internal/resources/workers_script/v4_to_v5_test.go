package workers_script

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestWorkerScriptConfigTransform_BasicRename(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "resource type rename - singular to plural",
			Input: `resource "cloudflare_worker_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  script_name = "my-worker"
}

moved {
  from = cloudflare_worker_script.example
  to   = cloudflare_workers_script.example
}`,
		},
		{
			Name: "field rename - name to script_name",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "test-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  script_name = "test-worker"
}`,
		},
		{
			Name: "remove tags field",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  tags       = ["production", "api"]
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  script_name = "my-worker"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestWorkerScriptStateTransform_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

func _TestWorkerScriptStateTransform_Bindings(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "plain_text_binding array conversion",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "plain_text_binding": [
      {
        "name": "MY_VAR",
        "text": "my-value"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "plain_text",
        "name": "MY_VAR",
        "text": "my-value"
      }
    ]
  }
}`,
		},
		{
			Name: "kv_namespace_binding array conversion",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "kv_namespace_binding": [
      {
        "name": "MY_KV",
        "namespace_id": "abc123"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "kv_namespace",
        "name": "MY_KV",
        "namespace_id": "abc123"
      }
    ]
  }
}`,
		},
		{
			Name: "webassembly_binding to wasm_module with attribute rename",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "webassembly_binding": [
      {
        "name": "WASM",
        "module": "base64content"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "wasm_module",
        "name": "WASM",
        "part": "base64content"
      }
    ]
  }
}`,
		},
		{
			Name: "queue_binding with attribute renames",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "queue_binding": [
      {
        "binding": "MY_QUEUE",
        "queue": "my-queue"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "queue",
        "name": "MY_QUEUE",
        "queue_name": "my-queue"
      }
    ]
  }
}`,
		},
		{
			Name: "d1_database_binding with attribute rename",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "d1_database_binding": [
      {
        "name": "MY_DB",
        "database_id": "db-uuid-123"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "d1",
        "name": "MY_DB",
        "id": "db-uuid-123"
      }
    ]
  }
}`,
		},
		{
			Name: "hyperdrive_config_binding with attribute rename",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "hyperdrive_config_binding": [
      {
        "binding": "HYPERDRIVE",
        "id": "hyperdrive-uuid-123"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "hyperdrive",
        "name": "HYPERDRIVE",
        "id": "hyperdrive-uuid-123"
      }
    ]
  }
}`,
		},
		{
			Name: "dispatch_namespace attribute conversion",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "dispatch_namespace": "my-namespace"
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "dispatch_namespace",
        "namespace": "my-namespace"
      }
    ]
  }
}`,
		},
		{
			Name: "multiple bindings of different types",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "plain_text_binding": [
      {
        "name": "VAR1",
        "text": "value1"
      }
    ],
    "kv_namespace_binding": [
      {
        "name": "KV1",
        "namespace_id": "kv-123"
      }
    ],
    "d1_database_binding": [
      {
        "name": "DB1",
        "database_id": "db-456"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "bindings": [
      {
        "type": "plain_text",
        "name": "VAR1",
        "text": "value1"
      },
      {
        "type": "kv_namespace",
        "name": "KV1",
        "namespace_id": "kv-123"
      },
      {
        "type": "d1",
        "name": "DB1",
        "id": "db-456"
      }
    ]
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func _TestWorkerScriptStateTransform_Module(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "module = true becomes main_module",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "export default { fetch() { return new Response('Hello'); } };",
    "module": true
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "export default { fetch() { return new Response('Hello'); } };",
    "main_module": "worker.js"
  }
}`,
		},
		{
			Name: "module = false becomes body_part",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "module": false
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "my-worker",
    "content": "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });",
    "body_part": "worker.js"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func _TestWorkerScriptStateTransform_Comprehensive(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "comprehensive transformation with all features",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "comprehensive-worker",
    "content": "export default { fetch() { return new Response('Hello'); } };",
    "module": true,
    "dispatch_namespace": "my-namespace",
    "tags": ["production", "api"],
    "plain_text_binding": [
      {
        "name": "API_KEY",
        "text": "test-key"
      }
    ],
    "kv_namespace_binding": [
      {
        "name": "KV_STORE",
        "namespace_id": "kv-123"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 1,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "script_name": "comprehensive-worker",
    "content": "export default { fetch() { return new Response('Hello'); } };",
    "main_module": "worker.js",
    "bindings": [
      {
        "type": "plain_text",
        "name": "API_KEY",
        "text": "test-key"
      },
      {
        "type": "kv_namespace",
        "name": "KV_STORE",
        "namespace_id": "kv-123"
      },
      {
        "type": "dispatch_namespace",
        "namespace": "my-namespace"
      }
    ]
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}

func TestWorkerScriptConfigTransform_Bindings(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "plain_text_binding conversion",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  plain_text_binding {
    name = "MY_VAR"
    text = "my-value"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type = "plain_text"
      name = "MY_VAR"
      text = "my-value"
    }
  ]
}`,
		},
		{
			Name: "kv_namespace_binding conversion",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  kv_namespace_binding {
    name         = "MY_KV"
    namespace_id = "abc123"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type         = "kv_namespace"
      name         = "MY_KV"
      namespace_id = "abc123"
    }
  ]
}`,
		},
		{
			Name: "webassembly_binding to wasm_module with attribute rename",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  webassembly_binding {
    name   = "WASM"
    module = "base64content"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type = "wasm_module"
      name = "WASM"
      part = "base64content"
    }
  ]
}`,
		},
		{
			Name: "queue_binding with attribute renames",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  queue_binding {
    binding = "MY_QUEUE"
    queue   = "my-queue"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type       = "queue"
      name       = "MY_QUEUE"
      queue_name = "my-queue"
    }
  ]
}`,
		},
		{
			Name: "d1_database_binding with attribute rename",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  d1_database_binding {
    name        = "MY_DB"
    database_id = "db-uuid-123"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type = "d1"
      name = "MY_DB"
      id   = "db-uuid-123"
    }
  ]
}`,
		},
		{
			Name: "hyperdrive_config_binding with attribute rename",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  hyperdrive_config_binding {
    binding = "HYPERDRIVE"
    id      = "hyperdrive-uuid-123"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type = "hyperdrive"
      name = "HYPERDRIVE"
      id   = "hyperdrive-uuid-123"
    }
  ]
}`,
		},
		{
			Name: "dispatch_namespace attribute conversion",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name               = "my-worker"
  content            = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  dispatch_namespace = "my-namespace"
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  script_name = "my-worker"
  bindings = [
    {
      type      = "dispatch_namespace"
      namespace = "my-namespace"
    }
  ]
}`,
		},
		{
			Name: "multiple bindings of different types",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  plain_text_binding {
    name = "VAR1"
    text = "value1"
  }

  kv_namespace_binding {
    name         = "KV1"
    namespace_id = "kv-123"
  }

  d1_database_binding {
    name        = "DB1"
    database_id = "db-456"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  bindings = [
    {
      type = "plain_text"
      name = "VAR1"
      text = "value1"
    }, {
      type         = "kv_namespace"
      name         = "KV1"
      namespace_id = "kv-123"
    }, {
      type = "d1"
      name = "DB1"
      id   = "db-456"
    }
  ]
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestWorkerScriptConfigTransform_Module(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "module = true becomes main_module",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "export default { fetch() { return new Response('Hello'); } };"
  module     = true
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "export default { fetch() { return new Response('Hello'); } };"
  script_name = "my-worker"
  main_module = "worker.js"
}`,
		},
		{
			Name: "module = false becomes body_part",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  module     = false
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  content     = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"
  script_name = "my-worker"
  body_part   = "worker.js"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestWorkerScriptConfigTransform_Placement(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "placement block to object attribute",
			Input: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "my-worker"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  placement {
    mode = "smart"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')); });"

  script_name = "my-worker"
  placement = {
    mode = "smart"
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestWorkerScriptConfigTransform_Comprehensive(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "comprehensive transformation with all features",
			Input: `resource "cloudflare_workers_script" "comprehensive" {
  account_id         = "f037e56e89293a057740de681ac9abbe"
  name               = "comprehensive-worker"
  content            = "export default { fetch() { return new Response('Hello'); } };"
  module             = true
  dispatch_namespace = "my-namespace"
  tags               = ["production", "api"]

  plain_text_binding {
    name = "API_KEY"
    text = "test-key"
  }

  kv_namespace_binding {
    name         = "KV_STORE"
    namespace_id = "kv-123"
  }

  placement {
    mode = "smart"
  }
}`,
			Expected: `resource "cloudflare_workers_script" "comprehensive" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "export default { fetch() { return new Response('Hello'); } };"

  script_name = "comprehensive-worker"
  bindings = [
    {
      type = "plain_text"
      name = "API_KEY"
      text = "test-key"
    }, {
      type         = "kv_namespace"
      name         = "KV_STORE"
      namespace_id = "kv-123"
    }, {
      type      = "dispatch_namespace"
      namespace = "my-namespace"
    }
  ]
  main_module = "worker.js"
  placement = {
    mode = "smart"
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
