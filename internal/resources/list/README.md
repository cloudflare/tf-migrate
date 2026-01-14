# List Migration Guide (v4 → v5)

This guide explains how `cloudflare_list` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_list` | `cloudflare_list` | No change |
| Items structure | `item { }` blocks | `items = [...]` attribute | Block → array |
| Item values | `item { value { ... } }` | Flattened in array | Structure simplified |
| IP addresses | CIDR notation `"10.0.0.0/8"` | Plain IP `"10.0.0.0"` | CIDR removed |
| Redirect booleans | String `"enabled"/"disabled"` | Boolean `true/false` | Type change |
| Redirect URLs | No path required | Path required `"test.com/"` | Validation added |
| Dynamic blocks | `dynamic "item"` | `items = [for...]` | For expression |
| `cloudflare_list_item` | Separate resource | Merged into parent | Cross-resource merge |


---

## Migration Examples

### Example 1: Simple IP List

**v4 Configuration:**
```hcl
resource "cloudflare_list" "ips" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "blocked_ips"
  kind        = "ip"
  description = "Blocked IP addresses"

  item {
    comment = "First IP"
    value {
      ip = "10.0.0.0/8"
    }
  }

  item {
    value {
      ip = "192.168.1.1"
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "ips" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "blocked_ips"
  kind        = "ip"
  description = "Blocked IP addresses"

  items = [
    {
      comment = "First IP"
      ip      = "10.0.0.0"  # ← CIDR notation removed
    },
    {
      ip = "192.168.1.1"
    }
  ]
}
```

**What Changed:**
- `item { value { ip } }` → `items = [{ ip }]`
- CIDR notation stripped from IPs
- Structure flattened

---

### Example 2: Redirect List with Boolean Conversions

**v4 Configuration:**
```hcl
resource "cloudflare_list" "redirects" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "redirects"
  kind       = "redirect"

  item {
    comment = "Old domain redirect"
    value {
      redirect {
        source_url              = "old.com"
        target_url              = "new.com"
        include_subdomains      = "enabled"
        subpath_matching        = "disabled"
        preserve_query_string   = "enabled"
        preserve_path_suffix    = "disabled"
        status_code             = 301
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "redirects" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "redirects"
  kind       = "redirect"

  items = [{
    comment = "Old domain redirect"
    redirect = {
      source_url             = "old.com/"  # ← Path added
      target_url             = "new.com"
      include_subdomains     = true        # ← Boolean conversion
      subpath_matching       = false       # ← Boolean conversion
      preserve_query_string  = true        # ← Boolean conversion
      preserve_path_suffix   = false       # ← Boolean conversion
      status_code            = 301
    }
  }]
}
```

**What Changed:**
- String booleans → actual booleans
- `"enabled"` → `true`, `"disabled"` → `false`
- Path added to source_url for validation

---

### Example 3: ASN List

**v4 Configuration:**
```hcl
resource "cloudflare_list" "asns" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_asns"
  kind       = "asn"

  item {
    comment = "Example ASN"
    value {
      asn = 13335
    }
  }

  item {
    value {
      asn = 209242
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "asns" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_asns"
  kind       = "asn"

  items = [
    {
      comment = "Example ASN"
      asn     = 13335
    },
    {
      asn = 209242
    }
  ]
}
```

---

### Example 4: Hostname List

**v4 Configuration:**
```hcl
resource "cloudflare_list" "hostnames" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_hosts"
  kind       = "hostname"

  item {
    value {
      hostname {
        url_hostname = "malicious.com"
      }
    }
  }

  item {
    comment = "Another bad host"
    value {
      hostname {
        url_hostname = "phishing.com"
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "hostnames" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_hosts"
  kind       = "hostname"

  items = [
    {
      hostname = {
        url_hostname = "malicious.com"
      }
    },
    {
      comment = "Another bad host"
      hostname = {
        url_hostname = "phishing.com"
      }
    }
  ]
}
```

---

### Example 5: Dynamic List with for_each

**v4 Configuration:**
```hcl
variable "blocked_ips" {
  default = ["10.0.0.1", "10.0.0.2", "10.0.0.3"]
}

resource "cloudflare_list" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "dynamic_ips"
  kind       = "ip"

  dynamic "item" {
    for_each = var.blocked_ips
    iterator = ip_item
    content {
      value {
        ip = ip_item.value
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
variable "blocked_ips" {
  default = ["10.0.0.1", "10.0.0.2", "10.0.0.3"]
}

resource "cloudflare_list" "dynamic" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "dynamic_ips"
  kind       = "ip"

  items = [for ip_item in var.blocked_ips : {
    ip = ip_item  # ← .value suffix stripped
  }]
}
```

**What Changed:**
- `dynamic "item"` → `items = [for...]` comprehension
- Iterator references simplified (`.value` suffix removed)

---

### Example 6: Mixed Static and Dynamic Items

**v4 Configuration:**
```hcl
resource "cloudflare_list" "mixed" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "mixed_ips"
  kind       = "ip"

  item {
    comment = "Static IP"
    value {
      ip = "192.168.1.1"
    }
  }

  dynamic "item" {
    for_each = var.dynamic_ips
    content {
      value {
        ip = item.value
      }
    }
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "mixed" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "mixed_ips"
  kind       = "ip"

  items = concat([
    {
      comment = "Static IP"
      ip      = "192.168.1.1"
    }
  ], [for item in var.dynamic_ips : {
    ip = item
  }])
}
```

**What Changed:**
- Static and dynamic combined using `concat()`
- Static items in first array, dynamic in second

---

