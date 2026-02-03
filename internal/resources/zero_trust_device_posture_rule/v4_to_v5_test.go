package zero_trust_device_posture_rule

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "minimal resource - rename",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "serial_number"
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "serial_number"
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "minimal resource - no rename",
			Input: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "serial_number"
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "serial_number"
}
`,
		},
		{
			Name: "resource with input block",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input {
    version  = "10.0"
    operator = ">="
    enabled  = true
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input = {
    version  = "10.0"
    operator = ">="
    enabled  = true
  }
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "resource with input block and nested locations block",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input {
    version  = "10.0"
    operator = ">="
    enabled  = true
    locations {
      paths        = ["/etc/ssl/certs"]
      trust_stores = ["system"]
    }
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input = {
    version  = "10.0"
    operator = ">="
    enabled  = true
    locations = {
      paths        = ["/etc/ssl/certs"]
      trust_stores = ["system"]
    }
  }
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "resource with match blocks",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  match {
    platform = "windows"
  }
  match {
    platform = "mac"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  match = [{
    platform = "windows"
    }, {
    platform = "mac"
  }]
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "resource with removed running attribute",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "serial_number"
  input {
    exists  = true
    running = 1
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "serial_number"
  input      = {
    exists = true
  }
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "comprehensive resource with all field types",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "name"
  description = "Tests all field types and transformations"
  type        = "client_certificate_v2"
  schedule    = "5m"
  expiration  = "1h"
  input {
    id                   = "test-id-123"
    path                 = "/usr/bin/app"
    exists               = true
    thumbprint           = "abc123def456"
    sha256               = "sha256hash"
    running              = true
    require_all          = true
    check_disks          = ["C:", "D:", "E:"]
    enabled              = true
    version              = "1.0.0"
    os_version_extra     = "build-123"
    operator             = ">="
    domain               = "example.com"
    connection_id        = "conn-123"
    compliance_status    = "compliant"
    os_distro_name       = "Ubuntu"
    os_distro_revision   = "22.04"
    os                   = "linux"
    overall              = "pass"
    sensor_config        = "sensor-config-123"
    version_operator     = ">="
    last_seen            = "2024-01-01T00:00:00Z"
    state                = "active"
    count_operator       = ">"
    issue_count          = "5"
    certificate_id       = "cert-123"
    cn                   = "device.example.com"
    active_threats       = 1
    operational_state    = "active"
    network_status       = "connected"
    infected             = false
    is_active            = true
    eid_last_seen        = "2024-01-01T00:00:00Z"
    risk_level           = "low"
    total_score          = 85
    check_private_key    = true
    extended_key_usage   = ["clientAuth", "serverAuth"]
    score                = 100
    locations {
      paths        = ["/etc/ssl/certs", "/usr/local/share/certs", "/opt/certs"]
      trust_stores = ["system", "user", "custom"]
    }
  }
  match {
    platform = "windows"
  }
  match {
    platform = "mac"
  }
  match {
    platform = "linux"
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "name"
  description = "Tests all field types and transformations"
  type        = "client_certificate_v2"
  schedule    = "5m"
  expiration  = "1h"
  input = {
    id                   = "test-id-123"
    path                 = "/usr/bin/app"
    exists               = true
    thumbprint           = "abc123def456"
    sha256               = "sha256hash"
    require_all          = true
    check_disks          = ["C:", "D:", "E:"]
    enabled              = true
    version              = "1.0.0"
    os_version_extra     = "build-123"
    operator             = ">="
    domain               = "example.com"
    connection_id        = "conn-123"
    compliance_status    = "compliant"
    os_distro_name       = "Ubuntu"
    os_distro_revision   = "22.04"
    os                   = "linux"
    overall              = "pass"
    sensor_config        = "sensor-config-123"
    version_operator     = ">="
    last_seen            = "2024-01-01T00:00:00Z"
    state                = "active"
    count_operator       = ">"
    issue_count          = "5"
    certificate_id       = "cert-123"
    cn                   = "device.example.com"
    active_threats       = 1
    operational_state    = "active"
    network_status       = "connected"
    infected             = false
    is_active            = true
    eid_last_seen        = "2024-01-01T00:00:00Z"
    risk_level           = "low"
    total_score          = 85
    check_private_key    = true
    extended_key_usage   = ["clientAuth", "serverAuth"]
    score                = 100
    locations = {
      paths        = ["/etc/ssl/certs", "/usr/local/share/certs", "/opt/certs"]
      trust_stores = ["system", "user", "custom"]
    }
  }
  match = [{
    platform = "windows"
    }, {
    platform = "mac"
    }, {
    platform = "linux"
  }]
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "match block with dynamic reference",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  match {
    platform = each.value.platform
  }
  input {
    version  = each.value.version
    operator = ">="
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input = {
    version  = each.value.version
    operator = ">="
  }
  match = [
    {
      platform = each.value.platform
    }
  ]
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "multiple match blocks with dynamic references",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  match {
    platform = var.platform1
  }
  match {
    platform = var.platform2
  }
  input {
    version  = "1.0.0"
    operator = ">="
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input = {
    version  = "1.0.0"
    operator = ">="
  }
  match = [
    {
      platform = var.platform1
    },
    {
      platform = var.platform2
    }
  ]
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
		{
			Name: "match blocks with mixed static and dynamic values",
			Input: `
resource "cloudflare_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  match {
    platform = "windows"
  }
  match {
    platform = each.value.platform
  }
  input {
    version  = "1.0.0"
    operator = ">="
  }
}
`,
			Expected: `
resource "cloudflare_zero_trust_device_posture_rule" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "name"
  type       = "os_version"
  input = {
    version  = "1.0.0"
    operator = ">="
  }
  match = [
    {
      platform = each.value.platform
    },
    {
      platform = "windows"
    }
  ]
}
moved {
  from = cloudflare_device_posture_rule.test
  to   = cloudflare_zero_trust_device_posture_rule.test
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
