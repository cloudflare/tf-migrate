# Snippet Migration Guide (v4 → v5)

This guide explains how `cloudflare_snippet` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_snippet` | `cloudflare_snippet` | No change |
| `name` | `name = "..."` | `snippet_name = "..."` | Field renamed |
| `main_module` | Root-level attribute | `metadata.main_module` | Nested in metadata |
| `files` | Multiple blocks | Array attribute | Structure change |


---

## Migration Examples

### Example 1: Basic Snippet

**v4 Configuration:**
```hcl
resource "cloudflare_snippet" "example" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "my_snippet"
  main_module = "main.js"

  files {
    name    = "main.js"
    content = "export default { async fetch(request) { return new Response('Hello!') } }"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_snippet" "example" {
  zone_id      = "0da42c8d2132a9ddaf714f9e7c920711"
  snippet_name = "my_snippet"

  metadata = {
    main_module = "main.js"
  }

  files = [{
    name    = "main.js"
    content = "export default { async fetch(request) { return new Response('Hello!') } }"
  }]
}
```

**What Changed:**
- `name` → `snippet_name`
- `main_module` moved to `metadata.main_module`
- `files` blocks → array attribute

---

### Example 2: Multiple Files

**v4 Configuration:**
```hcl
resource "cloudflare_snippet" "multi_file" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  name        = "complex_snippet"
  main_module = "index.js"

  files {
    name    = "index.js"
    content = "import { helper } from './utils.js'; export default { async fetch(request) { return helper(request) } }"
  }

  files {
    name    = "utils.js"
    content = "export function helper(request) { return new Response('Processed') }"
  }

  files {
    name    = "config.json"
    content = "{\"version\": \"1.0.0\"}"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_snippet" "multi_file" {
  zone_id      = "0da42c8d2132a9ddaf714f9e7c920711"
  snippet_name = "complex_snippet"

  metadata = {
    main_module = "index.js"
  }

  files = [
    {
      name    = "index.js"
      content = "import { helper } from './utils.js'; export default { async fetch(request) { return helper(request) } }"
    },
    {
      name    = "utils.js"
      content = "export function helper(request) { return new Response('Processed') }"
    },
    {
      name    = "config.json"
      content = "{\"version\": \"1.0.0\"}"
    }
  ]
}
```

**What Changed:**
- Field renames applied
- Multiple `files` blocks → single array

---

