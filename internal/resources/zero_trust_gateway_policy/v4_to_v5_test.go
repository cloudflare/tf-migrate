package zero_trust_gateway_policy

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "minimal policy",
				Input: `
resource "cloudflare_teams_rule" "minimal" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Minimal policy"
  description = "Test policy"
  precedence  = 100
  action      = "allow"
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "minimal" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Minimal policy"
  description = "Test policy"
  precedence  = 100
  action      = "allow"
}`,
			},
			{
				Name: "policy with rule_settings field renames",
				Input: `
resource "cloudflare_teams_rule" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Policy with settings"
  description = "Test"
  precedence  = 200
  action      = "block"
  
  rule_settings {
    block_page_enabled = true
    block_page_reason  = "Access denied"
    
    notification_settings {
      enabled = true
      message = "Policy violation"
      support_url = "https://example.com"
    }
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Policy with settings"
  description = "Test"
  precedence  = 200
  action      = "block"
  
  rule_settings {
    block_page_enabled = true
    block_reason = "Access denied"
    
    notification_settings {
      enabled = true
      msg = "Policy violation"
      support_url = "https://example.com"
    }
  }
}`,
			},
			{
				Name: "policy with BISO admin controls",
				Input: `
resource "cloudflare_teams_rule" "with_biso" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "BISO policy"
  description = "Test"
  precedence  = 300
  action      = "isolate"
  
  rule_settings {
    biso_admin_controls {
      disable_printing = true
      disable_copy_paste = true
      disable_download = false
      disable_keyboard = false
      disable_upload = true
      disable_clipboard_redirection = true
    }
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "with_biso" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "BISO policy"
  description = "Test"
  precedence  = 300
  action      = "isolate"
  
  rule_settings {
    biso_admin_controls {
      dp = true
      dcp = true
      dd = false
      dk = false
      du = true
    }
  }
}`,
			},
			{
				Name: "multiple policies in single file",
				Input: `
resource "cloudflare_teams_rule" "first" {
  account_id  = "abc123"
  name        = "First"
  description = "First policy"
  precedence  = 100
  action      = "allow"
  
  rule_settings {
    block_page_reason = "First reason"
  }
}

resource "cloudflare_teams_rule" "second" {
  account_id  = "abc123"
  name        = "Second"
  description = "Second policy"
  precedence  = 200
  action      = "block"
  
  rule_settings {
    block_page_reason = "Second reason"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "first" {
  account_id  = "abc123"
  name        = "First"
  description = "First policy"
  precedence  = 100
  action      = "allow"
  
  rule_settings {
    block_reason = "First reason"
  }
}

resource "cloudflare_zero_trust_gateway_policy" "second" {
  account_id  = "abc123"
  name        = "Second"
  description = "Second policy"
  precedence  = 200
  action      = "block"
  
  rule_settings {
    block_reason = "Second reason"
  }
}`,
			},
			{
				Name: "empty description field",
				Input: `
resource "cloudflare_teams_rule" "empty_desc" {
  account_id  = "abc123"
  name        = "Empty desc"
  description = ""
  precedence  = 0
  action      = "allow"
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "empty_desc" {
  account_id  = "abc123"
  name        = "Empty desc"
  description = ""
  precedence  = 0
  action      = "allow"
}`,
			},
			{
				Name: "policy with DNS resolvers",
				Input: `
resource "cloudflare_teams_rule" "dns_policy" {
  account_id  = "abc123"
  name        = "DNS"
  description = "DNS policy"
  precedence  = 100
  action      = "allow"
  
  rule_settings {
    dns_resolvers {
      ipv4 {
        ip   = "1.1.1.1"
        port = 53
      }
      ipv4 {
        ip   = "1.0.0.1"
        port = 5053
      }
    }
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "dns_policy" {
  account_id  = "abc123"
  name        = "DNS"
  description = "DNS policy"
  precedence  = 100
  action      = "allow"
  
  rule_settings {
    dns_resolvers {
      ipv4 {
        ip   = "1.1.1.1"
        port = 53
      }
      ipv4 {
        ip   = "1.0.0.1"
        port = 5053
      }
    }
  }
}`,
			},
			{
				Name: "policy with filters and conditions",
				Input: `
resource "cloudflare_teams_rule" "complex" {
  account_id     = "abc123"
  name           = "Complex"
  description    = "Complex policy"
  precedence     = 100
  action         = "block"
  enabled        = true
  filters        = ["dns", "http", "l4"]
  traffic        = "any(http.request.uri.path contains \"/admin\")"
  identity       = "any(identity.groups.name == \"admins\")"
  device_posture = "any(device_posture.checks.passed == true)"
}`,
				Expected: `
resource "cloudflare_zero_trust_gateway_policy" "complex" {
  account_id     = "abc123"
  name           = "Complex"
  description    = "Complex policy"
  precedence     = 100
  action         = "block"
  enabled        = true
  filters        = ["dns", "http", "l4"]
  traffic        = "any(http.request.uri.path contains \"/admin\")"
  identity       = "any(identity.groups.name == \"admins\")"
  device_posture = "any(device_posture.checks.passed == true)"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "minimal state",
				Input: `{
					"attributes": {
						"id": "test-id-123",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "Minimal policy",
						"description": "Test policy",
						"precedence": 100,
						"action": "allow",
						"version": 1
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id-123",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "Minimal policy",
						"description": "Test policy",
						"precedence": 100.0,
						"action": "allow",
						"version": 1.0
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with rule_settings field renames",
				Input: `{
					"attributes": {
						"id": "test-id-456",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "Policy with settings",
						"description": "Test",
						"precedence": 200,
						"action": "block",
						"rule_settings": [{
							"block_page_enabled": true,
							"block_page_reason": "Access denied",
							"notification_settings": [{
								"enabled": true,
								"message": "Policy violation",
								"support_url": "https://example.com"
							}]
						}]
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id-456",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "Policy with settings",
						"description": "Test",
						"precedence": 200.0,
						"action": "block",
						"rule_settings": {
							"block_page_enabled": true,
							"block_reason": "Access denied",
							"notification_settings": {
								"enabled": true,
								"msg": "Policy violation",
								"support_url": "https://example.com"
							}
						}
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with BISO admin controls",
				Input: `{
					"attributes": {
						"id": "test-id-789",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "BISO policy",
						"description": "Test",
						"precedence": 300,
						"action": "isolate",
						"rule_settings": [{
							"biso_admin_controls": [{
								"disable_printing": true,
								"disable_copy_paste": true,
								"disable_download": false,
								"disable_keyboard": false,
								"disable_upload": true,
								"disable_clipboard_redirection": true
							}]
						}]
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id-789",
						"account_id": "f037e56e89293a057740de681ac9abbe",
						"name": "BISO policy",
						"description": "Test",
						"precedence": 300.0,
						"action": "isolate",
						"rule_settings": {
							"biso_admin_controls": {
								"dp": true,
								"dcp": true,
								"dd": false,
								"dk": false,
								"du": true
							}
						}
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with DNS resolver ports",
				Input: `{
					"attributes": {
						"id": "test-dns",
						"account_id": "abc123",
						"name": "DNS policy",
						"description": "Test",
						"precedence": 400,
						"action": "allow",
						"rule_settings": [{
							"dns_resolvers": [{
								"ipv4": [
									{"ip": "1.1.1.1", "port": 53},
									{"ip": "1.0.0.1", "port": 5053}
								],
								"ipv6": [
									{"ip": "2606:4700:4700::1111", "port": 53}
								]
							}]
						}]
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-dns",
						"account_id": "abc123",
						"name": "DNS policy",
						"description": "Test",
						"precedence": 400.0,
						"action": "allow",
						"rule_settings": {
							"dns_resolvers": {
								"ipv4": [
									{"ip": "1.1.1.1", "port": 53.0},
									{"ip": "1.0.0.1", "port": 5053.0}
								],
								"ipv6": [
									{"ip": "2606:4700:4700::1111", "port": 53.0}
								]
							}
						}
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with L4 override and check_session",
				Input: `{
					"attributes": {
						"id": "test-l4",
						"account_id": "abc123",
						"name": "L4 policy",
						"description": "Test",
						"precedence": 500,
						"action": "allow",
						"rule_settings": [{
							"l4override": [{
								"ip": "10.0.0.1",
								"port": 8080
							}],
							"check_session": [{
								"enforce": true,
								"duration": 300
							}]
						}]
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-l4",
						"account_id": "abc123",
						"name": "L4 policy",
						"description": "Test",
						"precedence": 500.0,
						"action": "allow",
						"rule_settings": {
							"l4override": {
								"ip": "10.0.0.1",
								"port": 8080.0
							},
							"check_session": {
								"enforce": true,
								"duration": 300.0
							}
						}
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "empty state with no attributes",
				Input: `{
					"attributes": {}
				}`,
				Expected: `{
					"attributes": {},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with empty filters array",
				Input: `{
					"attributes": {
						"id": "test-empty-filters",
						"account_id": "abc123",
						"name": "Empty filters",
						"description": "Test",
						"precedence": 600,
						"action": "allow",
						"filters": []
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-empty-filters",
						"account_id": "abc123",
						"name": "Empty filters",
						"description": "Test",
						"precedence": 600.0,
						"action": "allow",
						"filters": []
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with null values",
				Input: `{
					"attributes": {
						"id": "test-nulls",
						"account_id": "abc123",
						"name": "Null values",
						"description": "Test",
						"precedence": 700,
						"action": "allow",
						"filters": null,
						"traffic": null,
						"identity": null,
						"device_posture": null
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-nulls",
						"account_id": "abc123",
						"name": "Null values",
						"description": "Test",
						"precedence": 700.0,
						"action": "allow",
						"filters": null,
						"traffic": null,
						"identity": null,
						"device_posture": null
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "state with conditions",
				Input: `{
					"attributes": {
						"id": "test-conditions",
						"account_id": "abc123",
						"name": "With conditions",
						"description": "Test",
						"precedence": 800,
						"action": "block",
						"enabled": true,
						"filters": ["dns", "http"],
						"traffic": "any(http.request.uri.path contains \"/admin\")",
						"identity": "any(identity.groups.name == \"admins\")",
						"device_posture": "any(device_posture.checks.passed == true)"
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-conditions",
						"account_id": "abc123",
						"name": "With conditions",
						"description": "Test",
						"precedence": 800.0,
						"action": "block",
						"enabled": true,
						"filters": ["dns", "http"],
						"traffic": "any(http.request.uri.path contains \"/admin\")",
						"identity": "any(identity.groups.name == \"admins\")",
						"device_posture": "any(device_posture.checks.passed == true)"
					},
					"schema_version": 0
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}