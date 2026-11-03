# Zero Trust Tunnel Cloudflared Migration Guide (v4 → v5)

This guide explains how `cloudflare_tunnel` / `cloudflare_zero_trust_tunnel_cloudflared` resources migrate to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_tunnel` | `cloudflare_zero_trust_tunnel_cloudflared` | Renamed (deprecated) + `moved {}` block |
| Alt resource name | `cloudflare_zero_trust_tunnel_cloudflared` | `cloudflare_zero_trust_tunnel_cloudflared` | No change, no `moved {}` block |
| `secret` | Field name | `tunnel_secret` | Renamed (config + cross-file references) |
| `config_src` | Optional, ForceNew | Optional + Computed, defaults to `"local"` | **Left untouched** — see below |
| `tunnel_token` | Computed output | Not an attribute | Removed — emits a `DiagWarning` on reference |

Both v4 names are registered, so a config using either one migrates correctly. Only
`cloudflare_tunnel` gets a type rename and a `moved {}` block; the preferred name is
updated in place.

---

## `config_src` is deliberately not written

tf-migrate does **not** add `config_src` to the migrated config when it is absent from
your v4 configuration. This is intentional and must stay that way.

In v5, `config_src` is `Optional + Computed` with `Default: stringdefault.StaticString("local")`,
and it carries `RequiresReplaceIfConfigured()`. That plan modifier only fires when the
attribute is **explicitly set in configuration**. Writing it — even to the value tf-migrate
believes is correct — arms the trigger.

A tunnel managed through `cloudflare_tunnel_config` has a real `config_src` of
`"cloudflare"`. The v4 provider only refreshed `config_src` into state if it was already
non-empty, so imported or dashboard-created tunnels commonly sit at `""` in v4 state while
actually being remotely managed. Synthesizing `config_src = "local"` for those produces:

```
~ config_src = "cloudflare" -> "local" # forces replacement
```

which destroys and recreates the tunnel **and** its configuration. Leaving the attribute
out plans cleanly, because the provider's own default covers the genuinely-local case and
the computed value is preserved for the remote one.

This regressed once already (BUGS-2158). The regression fixture lives in
`integration/v4_to_v5/testdata/zero_trust_tunnel_cloudflared_config/` — a tunnel that omits
`config_src` alongside a `tunnel_config` sibling. Every other tunnel fixture sets
`config_src` explicitly, which is why the original bug was invisible to the suite.

If you need to change `config_src`, do it as a separate, deliberate apply after the
migration — not as part of it.

---

## Migration Examples

### Example 1: Deprecated resource name

**v4 Configuration:**
```hcl
resource "cloudflare_tunnel" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-tunnel"
  secret     = base64encode("a-secret-that-is-at-least-32-bytes-long")
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "example-tunnel"
  tunnel_secret = base64encode("a-secret-that-is-at-least-32-bytes-long")
}

moved {
  from = cloudflare_tunnel.example
  to   = cloudflare_zero_trust_tunnel_cloudflared.example
}
```

**What Changed:**
- Resource type renamed from the deprecated name
- `secret` → `tunnel_secret`
- `moved {}` block generated so the provider's `MoveState` hook migrates state in place
- No `config_src` added

---

### Example 2: Preferred v4 name (in-place)

**v4 Configuration:**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "example-tunnel"
  secret     = base64encode("a-secret-that-is-at-least-32-bytes-long")
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared" "example" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "example-tunnel"
  tunnel_secret = base64encode("a-secret-that-is-at-least-32-bytes-long")
}
```

**What Changed:**
- `secret` → `tunnel_secret`
- No type rename, so no `moved {}` block

---

### Example 3: Remotely-managed tunnel

**v4 Configuration:**
```hcl
resource "cloudflare_tunnel" "remote" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "remote-tunnel"
  secret     = base64encode("a-secret-that-is-at-least-32-bytes-long")
  # config_src not set, but this tunnel is managed via cloudflare_tunnel_config
}

resource "cloudflare_tunnel_config" "remote" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_tunnel.remote.id

  config {
    ingress_rule {
      service = "http_status:404"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_tunnel_cloudflared" "remote" {
  account_id    = "f037e56e89293a057740de681ac9abbe"
  name          = "remote-tunnel"
  tunnel_secret = base64encode("a-secret-that-is-at-least-32-bytes-long")
}

moved {
  from = cloudflare_tunnel.remote
  to   = cloudflare_zero_trust_tunnel_cloudflared.remote
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "remote" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.remote.id

  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
  }
}

moved {
  from = cloudflare_tunnel_config.remote
  to   = cloudflare_zero_trust_tunnel_cloudflared_config.remote
}
```

**What Changed:**
- Standard rename and `secret` → `tunnel_secret`
- `config_src` still absent — the tunnel's real value (`"cloudflare"`) is preserved as a
  computed value instead of being overwritten with `"local"`

---

## Cross-file references

`secret` → `tunnel_secret` is also rewritten in other files, including non-Cloudflare
resources, via `GetComputedAttributeMappings()`. Both v4 type names are covered:

**Before:**
```hcl
resource "vault_generic_secret" "tunnel" {
  data_json = jsonencode({
    secret = cloudflare_tunnel.example.secret
  })
}
```

**After:**
```hcl
resource "vault_generic_secret" "tunnel" {
  data_json = jsonencode({
    secret = cloudflare_zero_trust_tunnel_cloudflared.example.tunnel_secret
  })
}
```

---

## Manual action: `tunnel_token`

`tunnel_token` is not a valid attribute on `cloudflare_zero_trust_tunnel_cloudflared` in v5.
Any reference to it produces a `DiagWarning` during migration and must be resolved by hand.

Valid computed attributes are `tunnel_secret`, `cname`, and `id`.
