# Zero Trust Device Posture Rule Migration Guide (v4 → v5)

This guide explains how `cloudflare_device_posture_rule` resources migrate to `cloudflare_zero_trust_device_posture_rule` in v5.

## Quick Reference

| Aspect | v4 | v5 | Change |
|--------|----|----|--------|
| Resource name | `cloudflare_device_posture_rule` | `cloudflare_zero_trust_device_posture_rule` | Renamed |
| `input` | Block (array) | Attribute object | Syntax change |
| `match` | Multiple blocks | Array attribute | Structure change |
| `input.enabled = false` | Configurable | Removed | Deprecated |
| `input.running` | Supported | Removed | Deprecated field |
| Numeric fields | Int | Int64 (float64 in state) | Type conversion |


---

## Migration Examples

### Example 1: File Check Rule

**v4 Configuration:**
```hcl
resource "cloudflare_device_posture_rule" "file_check" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Check antivirus file"
  type        = "file"
  description = "Verify antivirus signature file exists"

  match {
    platform = "windows"
  }

  match {
    platform = "mac"
  }

  input {
    path = "C:\\Program Files\\Antivirus\\signatures.dat"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_posture_rule" "file_check" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Check antivirus file"
  type        = "file"
  description = "Verify antivirus signature file exists"

  match = [
    { platform = "windows" },
    { platform = "mac" }
  ]

  input = {
    path = "C:\\Program Files\\Antivirus\\signatures.dat"
  }
}
```

**What Changed:**
- Resource type renamed
- Multiple `match` blocks → single `match` array
- `input` block → `input` attribute object

---

### Example 2: Application Check with Version

**v4 Configuration:**
```hcl
resource "cloudflare_device_posture_rule" "app_version" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Require Chrome v100+"
  type       = "application"

  match {
    platform = "windows"
  }

  input {
    id               = "chrome-app-id"
    version          = "100.0.0.0"
    operator         = ">="
    check_disks      = ["C"]
    require_all      = true
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_posture_rule" "app_version" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Require Chrome v100+"
  type       = "application"

  match = [{
    platform = "windows"
  }]

  input = {
    id          = "chrome-app-id"
    version     = "100.0.0.0"
    operator    = ">="
    check_disks = ["C"]
    require_all = true
  }
}
```

**What Changed:**
- Blocks → attributes
- All fields preserved

---

### Example 3: Client Certificate with Thumbprint

**v4 Configuration:**
```hcl
resource "cloudflare_device_posture_rule" "client_cert" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Valid Client Certificate"
  type       = "client_certificate"

  match {
    platform = "windows"
  }

  input {
    certificate_id = "cert-id-123"
    cn             = "device.example.com"
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_posture_rule" "client_cert" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Valid Client Certificate"
  type       = "client_certificate"

  match = [{
    platform = "windows"
  }]

  input = {
    certificate_id = "cert-id-123"
    cn             = "device.example.com"
  }
}
```

---

### Example 4: Disk Encryption Check

**v4 Configuration:**
```hcl
resource "cloudflare_device_posture_rule" "disk_encryption" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Require Disk Encryption"
  type       = "disk_encryption"

  match {
    platform = "mac"
  }

  input {
    require_all = true
    check_disks = ["/"]
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_posture_rule" "disk_encryption" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Require Disk Encryption"
  type       = "disk_encryption"

  match = [{
    platform = "mac"
  }]

  input = {
    require_all = true
    check_disks = ["/"]
  }
}
```

---

### Example 5: OS Version Check

**v4 Configuration:**
```hcl
resource "cloudflare_device_posture_rule" "os_version" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Minimum OS Version"
  type       = "os_version"

  match {
    platform = "windows"
  }

  input {
    version  = "10.0.19041"
    operator = ">="
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_posture_rule" "os_version" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Minimum OS Version"
  type       = "os_version"

  match = [{
    platform = "windows"
  }]

  input = {
    version  = "10.0.19041"
    operator = ">="
  }
}
```

---

### Example 6: Sentinel One with Threat Scores

**v4 Configuration:**
```hcl
resource "cloudflare_device_posture_rule" "sentinelone" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SentinelOne Check"
  type       = "sentinelone"

  match {
    platform = "linux"
  }

  input {
    connection_id   = "s1-connection-id"
    active_threats  = 0
    network_status  = "connected"
    operator        = "<="
  }
}
```

**v5 Configuration (After Migration):**
```hcl
resource "cloudflare_zero_trust_device_posture_rule" "sentinelone" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "SentinelOne Check"
  type       = "sentinelone"

  match = [{
    platform = "linux"
  }]

  input = {
    connection_id  = "s1-connection-id"
    active_threats = 0
    network_status = "connected"
    operator       = "<="
  }
}
```

**What Changed:**
- `active_threats` converted to float64 in state

---

