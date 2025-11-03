# Snippet example - no transformation needed
resource "cloudflare_snippet" "example" {
  zone_id      = "0da42c8d2132a9ddaf714f9e7c920711"
  content      = "console.log('Hello from snippet');"
  snippet_name = "example_snippet"
}
