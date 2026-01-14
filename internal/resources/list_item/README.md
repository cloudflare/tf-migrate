# List Item Migration Guide (v4 → v5)

This guide explains how `cloudflare_list_item` resources migrate from v4 to v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource type | `cloudflare_list_item` | **Removed** - merged into parent | Cross-resource merge |
| Items location | Separate resources | Parent `cloudflare_list.items` | Consolidated |
| Dynamic patterns | `for_each` / `count` | For expressions in parent | Pattern conversion |


---

## Migration Overview

**Important:** The `cloudflare_list_item` resource no longer exists in v5. All list items are now embedded in the parent `cloudflare_list` resource's `items` attribute.

During migration:
1. All `cloudflare_list_item` resources are identified
2. Parent `cloudflare_list` is located via `list_id` reference
3. Items are merged into parent's `items` array
4. `cloudflare_list_item` resources are removed from configuration
5. State is updated to reflect the merge

---

## Migration Examples

### Example 1: Static List Items

**v4 Configuration:**
```hcl
resource "cloudflare_list" "blocked_ips" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "blocked_ips"
  kind        = "ip"
  description = "Blocked IP addresses"
}

resource "cloudflare_list_item" "ip1" {
  list_id = cloudflare_list.blocked_ips.id
  ip      = "192.0.2.1"
  comment = "Server 1"
}

resource "cloudflare_list_item" "ip2" {
  list_id = cloudflare_list.blocked_ips.id
  ip      = "192.0.2.2"
  comment = "Server 2"
}

resource "cloudflare_list_item" "ip3" {
  list_id = cloudflare_list.blocked_ips.id
  ip      = "192.0.2.3"
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "blocked_ips" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "blocked_ips"
  kind        = "ip"
  description = "Blocked IP addresses"

  items = [
    {
      ip      = "192.0.2.1"
      comment = "Server 1"
    },
    {
      ip      = "192.0.2.2"
      comment = "Server 2"
    },
    {
      ip = "192.0.2.3"
    }
  ]
}
```

**What Changed:**
- Three separate `cloudflare_list_item` resources merged into one `items` array
- `list_id` references removed (no longer needed)
- All `cloudflare_list_item` resources deleted

---

### Example 2: Dynamic List Items with for_each

**v4 Configuration:**
```hcl
variable "blocked_ips" {
  default = {
    server1 = "10.0.0.1"
    server2 = "10.0.0.2"
    server3 = "10.0.0.3"
  }
}

resource "cloudflare_list" "dynamic_ips" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "dynamic_blocked_ips"
  kind       = "ip"
}

resource "cloudflare_list_item" "dynamic" {
  for_each = var.blocked_ips

  list_id = cloudflare_list.dynamic_ips.id
  ip      = each.value
  comment = "Blocked IP ${each.key}"
}
```

**v5 Configuration (After Migration):**
```hcl
variable "blocked_ips" {
  default = {
    server1 = "10.0.0.1"
    server2 = "10.0.0.2"
    server3 = "10.0.0.3"
  }
}

resource "cloudflare_list" "dynamic_ips" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "dynamic_blocked_ips"
  kind       = "ip"

  items = [for k, v in var.blocked_ips : {
    ip      = v
    comment = "Blocked IP ${k}"
  }]
}
```

**What Changed:**
- `for_each` on list_item resource → for expression in parent's `items`
- `each.key` → `k`, `each.value` → `v` (iterator variables)
- Separate resource consolidated into parent

---

### Example 3: Redirect List Items with Boolean Conversions

**v4 Configuration:**
```hcl
resource "cloudflare_list" "redirects" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "site_redirects"
  kind       = "redirect"
}

resource "cloudflare_list_item" "redirect1" {
  list_id = cloudflare_list.redirects.id

  redirect {
    source_url             = "old.example.com"
    target_url             = "new.example.com"
    include_subdomains     = "enabled"
    preserve_query_string  = "disabled"
    status_code            = 301
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "redirects" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "site_redirects"
  kind       = "redirect"

  items = [{
    redirect = {
      source_url            = "old.example.com"
      target_url            = "new.example.com"
      include_subdomains    = true   # ← Boolean conversion
      preserve_query_string = false  # ← Boolean conversion
      status_code           = 301
    }
  }]
}
```

**What Changed:**
- Redirect block structure preserved but moved to parent
- String booleans converted to actual booleans
- `"enabled"` → `true`, `"disabled"` → `false`

---

### Example 4: List Items with count

**v4 Configuration:**
```hcl
variable "asn_count" {
  default = 3
}

variable "asns" {
  default = [13335, 209242, 394536]
}

resource "cloudflare_list" "asns" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_asns"
  kind       = "asn"
}

resource "cloudflare_list_item" "asn" {
  count = var.asn_count

  list_id = cloudflare_list.asns.id
  asn     = var.asns[count.index]
}
```

**v5 Configuration (After Migration):**
```hcl
variable "asn_count" {
  default = 3
}

variable "asns" {
  default = [13335, 209242, 394536]
}

resource "cloudflare_list" "asns" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_asns"
  kind       = "asn"

  items = [for i in range(var.asn_count) : {
    asn = var.asns[i]
  }]
}
```

**What Changed:**
- `count` meta-argument → for expression with `range()`
- `count.index` → `i` (loop variable)

---

### Example 5: Hostname List Items

**v4 Configuration:**
```hcl
resource "cloudflare_list" "hosts" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_hosts"
  kind       = "hostname"
}

resource "cloudflare_list_item" "host1" {
  list_id = cloudflare_list.hosts.id

  hostname {
    url_hostname = "malicious.com"
  }
  comment = "Known malicious domain"
}

resource "cloudflare_list_item" "host2" {
  list_id = cloudflare_list.hosts.id

  hostname {
    url_hostname = "phishing.example"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_list" "hosts" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "blocked_hosts"
  kind       = "hostname"

  items = [
    {
      hostname = {
        url_hostname = "malicious.com"
      }
      comment = "Known malicious domain"
    },
    {
      hostname = {
        url_hostname = "phishing.example"
      }
    }
  ]
}
```

---

