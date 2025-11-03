# Workers script with various binding types
resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-script"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')) })"

  plain_text_binding {
    name = "MY_TEXT"
    text = "example-value"
  }

  kv_namespace_binding {
    name         = "MY_KV"
    namespace_id = "kv-namespace-id"
  }

  d1_database_binding {
    name        = "MY_DB"
    database_id = "database-id"
  }

  hyperdrive_config_binding {
    binding = "MY_HYPERDRIVE"
    id      = "hyperdrive-id"
  }
}
