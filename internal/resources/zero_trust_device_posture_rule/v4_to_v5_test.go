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
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "non empty input block",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "os_version",
						"input": [{
							"active_threats": 0,
							"certificate_id": "",
							"check_disks": null,
							"check_private_key": false,
							"cn": "",
							"compliance_status": "",
							"connection_id": "",
							"count_operator": "",
							"domain": "",
							"eid_last_seen": "",
							"enabled": false,
							"exists": false,
							"extended_key_usage": null,
							"id": "",
							"infected": false,
							"is_active": false,
							"issue_count": "",
							"last_seen": "",
							"locations": [],
							"network_status": "",
							"operational_state": "",
							"operator": "",
							"os": "linux",
							"os_distro_name": "",
							"os_distro_revision": "",
							"os_version_extra": "",
							"overall": "",
							"path": "",
							"require_all": false,
							"risk_level": "",
							"running": false,
							"score": 0,
							"sensor_config": "",
							"sha256": "",
							"state": "",
							"thumbprint": "",
							"total_score": 0,
							"version": "",
							"version_operator": ""
  						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "os_version",
						"input": {
							"active_threats": null,
							"certificate_id": null,
							"check_disks": null,
							"check_private_key": null,
							"cn": null,
							"compliance_status": null,
							"connection_id": null,
							"count_operator": null,
							"domain": null,
							"eid_last_seen": null,
							"exists": null,
							"extended_key_usage": null,
							"id": null,
							"infected": null,
							"is_active": null,
							"issue_count": null,
							"last_seen": null,
							"locations": null,
							"network_status": null,
							"operational_state": null,
							"operator": null,
							"os": "linux",
							"os_distro_name": null,
							"os_distro_revision": null,
							"os_version_extra": null,
							"overall": null,
							"path": null,
							"require_all": null,
							"risk_level": null,
							"score": null,
							"sensor_config": null,
							"sha256": null,
							"state": null,
							"thumbprint": null,
							"total_score": null,
							"version": null,
							"version_operator": null
  						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "minimal resource - rename",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "serial_number"
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "serial_number"
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "minimal resource - no rename",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "serial_number"
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "serial_number"
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "resource with input block",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "os_version",
						"input": [{
							"active_threats": 0,
							"certificate_id": "",
							"check_disks": null,
							"check_private_key": false,
							"cn": "",
							"compliance_status": "",
							"connection_id": "",
							"count_operator": "",
							"domain": "",
							"eid_last_seen": "",
							"enabled": true,
							"exists": false,
							"extended_key_usage": null,
							"id": "",
							"infected": false,
							"is_active": false,
							"issue_count": "",
							"last_seen": "",
							"locations": [],
							"network_status": "",
							"operational_state": "",
							"operator": ">=",
							"os": "",
							"os_distro_name": "",
							"os_distro_revision": "",
							"os_version_extra": "",
							"overall": "",
							"path": "",
							"require_all": false,
							"risk_level": "",
							"running": false,
							"score": 0,
							"sensor_config": "",
							"sha256": "",
							"state": "",
							"thumbprint": "",
							"total_score": 0,
							"version": "10.0",
							"version_operator": ""
						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "os_version",
						"input": {
							"active_threats": null,
							"certificate_id": null,
							"check_disks": null,
							"check_private_key": null,
							"cn": null,
							"compliance_status": null,
							"connection_id": null,
							"count_operator": null,
							"domain": null,
							"eid_last_seen": null,
							"enabled": true,
							"exists": null,
							"extended_key_usage": null,
							"id": null,
							"infected": null,
							"is_active": null,
							"issue_count": null,
							"last_seen": null,
							"locations": null,
							"network_status": null,
							"operational_state": null,
							"operator": ">=",
							"os": null,
							"os_distro_name": null,
							"os_distro_revision": null,
							"os_version_extra": null,
							"overall": null,
							"path": null,
							"require_all": null,
							"risk_level": null,
							"score": null,
							"sensor_config": null,
							"sha256": null,
							"state": null,
							"thumbprint": null,
							"total_score": null,
							"version": "10.0",
							"version_operator": null
						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "resource with input block and nested locations block",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "client_certificate_v2",
						"input": [{
							"active_threats": 0,
							"certificate_id": "",
							"check_disks": null,
							"check_private_key": true,
							"cn": "device.example.com",
							"compliance_status": "",
							"connection_id": "",
							"count_operator": "",
							"domain": "",
							"eid_last_seen": "",
							"enabled": false,
							"exists": false,
							"extended_key_usage": null,
							"id": "",
							"infected": false,
							"is_active": false,
							"issue_count": "",
							"last_seen": "",
							"locations": [{
								"paths": ["/etc/ssl/certs", "/usr/local/share/certs"],
								"trust_stores": ["system", "user"]
							}],
							"network_status": "",
							"operational_state": "",
							"operator": "",
							"os": "",
							"os_distro_name": "",
							"os_distro_revision": "",
							"os_version_extra": "",
							"overall": "",
							"path": "",
							"require_all": false,
							"risk_level": "",
							"running": false,
							"score": 0,
							"sensor_config": "",
							"sha256": "",
							"state": "",
							"thumbprint": "",
							"total_score": 0,
							"version": "",
							"version_operator": ""
						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "client_certificate_v2",
						"input": {
							"active_threats": null,
							"certificate_id": null,
							"check_disks": null,
							"check_private_key": true,
							"cn": "device.example.com",
							"compliance_status": null,
							"connection_id": null,
							"count_operator": null,
							"domain": null,
							"eid_last_seen": null,
							"exists": null,
							"extended_key_usage": null,
							"id": null,
							"infected": null,
							"is_active": null,
							"issue_count": null,
							"last_seen": null,
							"locations": {
								"paths": ["/etc/ssl/certs", "/usr/local/share/certs"],
								"trust_stores": ["system", "user"]
							},
							"network_status": null,
							"operational_state": null,
							"operator": null,
							"os": null,
							"os_distro_name": null,
							"os_distro_revision": null,
							"os_version_extra": null,
							"overall": null,
							"path": null,
							"require_all": null,
							"risk_level": null,
							"score": null,
							"sensor_config": null,
							"sha256": null,
							"state": null,
							"thumbprint": null,
							"total_score": null,
							"version": null,
							"version_operator": null
						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "removed running attribute",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "application",
						"input": [{
							"active_threats": 0,
							"certificate_id": "",
							"check_disks": null,
							"check_private_key": false,
							"cn": "",
							"compliance_status": "",
							"connection_id": "",
							"count_operator": "",
							"domain": "",
							"eid_last_seen": "",
							"enabled": false,
							"exists": false,
							"extended_key_usage": null,
							"id": "",
							"infected": false,
							"is_active": false,
							"issue_count": "",
							"last_seen": "",
							"locations": [],
							"network_status": "",
							"operational_state": "",
							"operator": "",
							"os": "",
							"os_distro_name": "",
							"os_distro_revision": "",
							"os_version_extra": "",
							"overall": "",
							"path": "/usr/bin/app",
							"require_all": false,
							"risk_level": "",
							"running": true,
							"score": 0,
							"sensor_config": "",
							"sha256": "abc123",
							"state": "",
							"thumbprint": "",
							"total_score": 0,
							"version": "",
							"version_operator": ""
						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "application",
						"input": {
							"active_threats": null,
							"certificate_id": null,
							"check_disks": null,
							"check_private_key": null,
							"cn": null,
							"compliance_status": null,
							"connection_id": null,
							"count_operator": null,
							"domain": null,
							"eid_last_seen": null,
							"exists": null,
							"extended_key_usage": null,
							"id": null,
							"infected": null,
							"is_active": null,
							"issue_count": null,
							"last_seen": null,
							"locations": null,
							"network_status": null,
							"operational_state": null,
							"operator": null,
							"os": null,
							"os_distro_name": null,
							"os_distro_revision": null,
							"os_version_extra": null,
							"overall": null,
							"path": "/usr/bin/app",
							"require_all": null,
							"risk_level": null,
							"score": null,
							"sensor_config": null,
							"sha256": "abc123",
							"state": null,
							"thumbprint": null,
							"total_score": null,
							"version": null,
							"version_operator": null
						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "resource with int to float attributes",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "sentinelone_s2s",
						"input": [{
							"active_threats": 5,
							"certificate_id": "",
							"check_disks": null,
							"check_private_key": false,
							"cn": "",
							"compliance_status": "",
							"connection_id": "",
							"count_operator": "",
							"domain": "",
							"eid_last_seen": "",
							"enabled": false,
							"exists": false,
							"extended_key_usage": null,
							"id": "",
							"infected": false,
							"is_active": false,
							"issue_count": "",
							"last_seen": "",
							"locations": [],
							"network_status": "",
							"operational_state": "",
							"operator": "",
							"os": "",
							"os_distro_name": "",
							"os_distro_revision": "",
							"os_version_extra": "",
							"overall": "",
							"path": "",
							"require_all": false,
							"risk_level": "",
							"running": false,
							"score": 5,
							"sensor_config": "",
							"sha256": "",
							"state": "",
							"thumbprint": "",
							"total_score": 5,
							"version": "",
							"version_operator": ""
						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "sentinelone_s2s",
						"input": {
							"active_threats": 5.0,
							"certificate_id": null,
							"check_disks": null,
							"check_private_key": null,
							"cn": null,
							"compliance_status": null,
							"connection_id": null,
							"count_operator": null,
							"domain": null,
							"eid_last_seen": null,
							"exists": null,
							"extended_key_usage": null,
							"id": null,
							"infected": null,
							"is_active": null,
							"issue_count": null,
							"last_seen": null,
							"locations": null,
							"network_status": null,
							"operational_state": null,
							"operator": null,
							"os": null,
							"os_distro_name": null,
							"os_distro_revision": null,
							"os_version_extra": null,
							"overall": null,
							"path": null,
							"require_all": null,
							"risk_level": null,
							"score": 5.0,
							"sensor_config": null,
							"sha256": null,
							"state": null,
							"thumbprint": null,
							"total_score": 5.0,
							"version": null,
							"version_operator": null
						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "comprehensive resource with all field types",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"description": "Tests all field types and transformations",
						"type": "client_certificate_v2",
						"schedule": "5m",
						"expiration": "1h",
						"match": [
							{"platform": "windows"},
							{"platform": "mac"},
							{"platform": "linux"}
						],
						"input": [{
							"id": "test-id-123",
							"path": "/usr/bin/app",
							"exists": true,
							"thumbprint": "abc123def456",
							"sha256": "sha256hash",
							"running": true,
							"require_all": true,
							"check_disks": ["C:", "D:", "E:"],
							"enabled": true,
							"version": "1.0.0",
							"os_version_extra": "build-123",
							"operator": ">=",
							"domain": "example.com",
							"connection_id": "conn-123",
							"compliance_status": "compliant",
							"os_distro_name": "Ubuntu",
							"os_distro_revision": "22.04",
							"os": "linux",
							"overall": "pass",
							"sensor_config": "sensor-config-123",
							"version_operator": ">=",
							"last_seen": "2024-01-01T00:00:00Z",
							"state": "active",
							"count_operator": ">",
							"issue_count": "5",
							"certificate_id": "cert-123",
							"cn": "device.example.com",
							"active_threats": 1,
							"operational_state": "active",
							"network_status": "connected",
							"infected": false,
							"is_active": true,
							"eid_last_seen": "2024-01-01T00:00:00Z",
							"risk_level": "low",
							"total_score": 85,
							"check_private_key": true,
							"extended_key_usage": ["clientAuth", "serverAuth"],
							"score": 100,
							"locations": [{
								"paths": ["/etc/ssl/certs", "/usr/local/share/certs", "/opt/certs"],
								"trust_stores": ["system", "user", "custom"]
							}]
						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"description": "Tests all field types and transformations",
						"type": "client_certificate_v2",
						"schedule": "5m",
						"expiration": "1h",
						"match": [
							{"platform": "windows"},
							{"platform": "mac"},
							{"platform": "linux"}
						],
						"input": {
							"id": "test-id-123",
							"path": "/usr/bin/app",
							"exists": true,
							"thumbprint": "abc123def456",
							"sha256": "sha256hash",
							"require_all": true,
							"check_disks": ["C:", "D:", "E:"],
							"enabled": true,
							"version": "1.0.0",
							"os_version_extra": "build-123",
							"operator": ">=",
							"domain": "example.com",
							"connection_id": "conn-123",
							"compliance_status": "compliant",
							"os_distro_name": "Ubuntu",
							"os_distro_revision": "22.04",
							"os": "linux",
							"overall": "pass",
							"sensor_config": "sensor-config-123",
							"version_operator": ">=",
							"last_seen": "2024-01-01T00:00:00Z",
							"state": "active",
							"count_operator": ">",
							"issue_count": "5",
							"certificate_id": "cert-123",
							"cn": "device.example.com",
							"active_threats": 1.0,
							"operational_state": "active",
							"network_status": "connected",
							"infected": null,
							"is_active": true,
							"eid_last_seen": "2024-01-01T00:00:00Z",
							"risk_level": "low",
							"total_score": 85.0,
							"check_private_key": true,
							"extended_key_usage": ["clientAuth", "serverAuth"],
							"score": 100.0,
							"locations": {
								"paths": ["/etc/ssl/certs", "/usr/local/share/certs", "/opt/certs"],
								"trust_stores": ["system", "user", "custom"]
							}
						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "resource with missing attributes",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "resource with empty array input",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "serial_number",
						"input": []
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "serial_number"
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
		{
			Name: "resource with empty locations array",
			Input: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "client_certificate_v2",
						"input": [{
							"active_threats": 0,
							"certificate_id": "",
							"check_disks": null,
							"check_private_key": true,
							"cn": "device.example.com",
							"compliance_status": "",
							"connection_id": "",
							"count_operator": "",
							"domain": "",
							"eid_last_seen": "",
							"enabled": false,
							"exists": false,
							"extended_key_usage": null,
							"id": "",
							"infected": false,
							"is_active": false,
							"issue_count": "",
							"last_seen": "",
							"locations": [],
							"network_status": "",
							"operational_state": "",
							"operator": "",
							"os": "",
							"os_distro_name": "",
							"os_distro_revision": "",
							"os_version_extra": "",
							"overall": "",
							"path": "",
							"require_all": false,
							"risk_level": "",
							"running": false,
							"score": 0,
							"sensor_config": "",
							"sha256": "",
							"state": "",
							"thumbprint": "",
							"total_score": 0,
							"version": "",
							"version_operator": ""
						}]
					},
					"schema_version": 1
				}]
			}]
		}`,
			Expected: `{
			"version": 4,
			"terraform_version": "1.5.0",
			"resources": [{
				"type": "cloudflare_zero_trust_device_posture_rule",
				"name": "name",
				"instances": [{
					"attributes": {
						"id": "test-rule-id",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "name",
						"type": "client_certificate_v2",
						"input": {
							"active_threats": null,
							"certificate_id": null,
							"check_disks": null,
							"check_private_key": true,
							"cn": "device.example.com",
							"compliance_status": null,
							"connection_id": null,
							"count_operator": null,
							"domain": null,
							"eid_last_seen": null,
							"exists": null,
							"extended_key_usage": null,
							"id": null,
							"infected": null,
							"is_active": null,
							"issue_count": null,
							"last_seen": null,
							"locations": null,
							"network_status": null,
							"operational_state": null,
							"operator": null,
							"os": null,
							"os_distro_name": null,
							"os_distro_revision": null,
							"os_version_extra": null,
							"overall": null,
							"path": null,
							"require_all": null,
							"risk_level": null,
							"score": null,
							"sensor_config": null,
							"sha256": null,
							"state": null,
							"thumbprint": null,
							"total_score": null,
							"version": null,
							"version_operator": null
						}
					},
					"schema_version": 0
				}]
			}]
		}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
