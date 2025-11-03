# Workers script with various binding types
resource "cloudflare_workers_script" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  content    = "addEventListener('fetch', event => { event.respondWith(new Response('Hello')) })"




  script_name = "example-script"
  bindings = [{
    type = "plain_text"
    name = "MY_TEXT"
    text = "example-value"
    },
    {
      type         = "kv_namespace"
      name         = "MY_KV"
      namespace_id = "kv-namespace-id"
    },
    {
      type = "d1"
      id   = "database-id"
      name = "MY_DB"
    },
    {
      type = "hyperdrive"
      name = "MY_HYPERDRIVE"
      id   = "hyperdrive-id"
  }]
}
