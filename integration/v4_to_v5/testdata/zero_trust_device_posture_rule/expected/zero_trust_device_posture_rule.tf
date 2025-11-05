# Test case 1: Basic os_version rule with input and match
resource "cloudflare_zero_trust_device_posture_rule" "basic" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "tf-test-posture-basic"
  type        = "os_version"
  description = "Device posture rule for corporate devices."
  schedule    = "24h"
  expiration  = "24h"


  input = {
    version            = "1.0.0"
    operator           = "<"
    os_distro_name     = "ubuntu"
    os_distro_revision = "1.0.0"
    os_version_extra   = "(a)"
  }
  match = [
    { platform = "linux" }
  ]
}

# Test case 2: Firewall rule with enabled input
resource "cloudflare_zero_trust_device_posture_rule" "firewall" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "tf-test-firewall"
  type       = "firewall"
  schedule   = "5m"


  input = {
    enabled = true
  }
  match = [
    { platform = "windows" }
  ]
}

# Test case 3: Disk encryption with check_disks (Set->List conversion)
resource "cloudflare_zero_trust_device_posture_rule" "disk_encryption" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "tf-test-disk"
  type       = "disk_encryption"
  schedule   = "5m"


  input = {
    check_disks = ["C:", "D:"]
    require_all = true
  }
  match = [
    { platform = "windows" }
  ]
}

# Test case 4: Multiple platforms (multiple match blocks)
resource "cloudflare_zero_trust_device_posture_rule" "multi_platform" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "tf-test-multi"
  type       = "firewall"
  schedule   = "5m"




  input = {
    enabled = true
  }
  match = [
    { platform = "windows" },
    { platform = "mac" },
    { platform = "linux" }
  ]
}
