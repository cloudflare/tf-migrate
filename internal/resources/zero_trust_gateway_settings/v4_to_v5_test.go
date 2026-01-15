package zero_trust_gateway_settings

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "minimal config",
			Input: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "minimal config - rename",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "url_browser_isolation_enabled & non_identity_browser_isolation_enabled not defined - defaults to false",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "block_page",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  block_page {
    enabled          = true
    name             = "Custom Block Page"
    footer_text      = "Contact your IT department"
    header_text      = "Access Blocked"
    logo_path        = "https://example.com/logo.png"
    background_color = "#FF0000"
    mailto_address   = "it@example.com"
    mailto_subject   = "Access Request"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    block_page = {
      enabled          = true
      name             = "Custom Block Page"
      footer_text      = "Contact your IT department"
      header_text      = "Access Blocked"
      logo_path        = "https://example.com/logo.png"
      background_color = "#FF0000"
      mailto_address   = "it@example.com"
      mailto_subject   = "Access Request"
    }
  }
}`,
		},
		{
			Name: "body_scanning",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  body_scanning {
    inspection_mode = "deep"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    body_scanning = {
      inspection_mode = "deep"
    }
  }
}`,
		},
		{
			Name: "fips",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  fips {
    tls = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    fips = {
      tls = true
    }
  }
}`,
		},
		{
			Name: "antivirus with notification_settings",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = false
    fail_closed            = true

    notification_settings {
      enabled     = true
      message     = "File is being scanned for threats"
      support_url = "https://support.example.com/antivirus"
    }
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = false
      fail_closed            = true
      notification_settings = {
        enabled     = true
        support_url = "https://support.example.com/antivirus"
        msg         = "File is being scanned for threats"
      }
    }
  }
}`,
		},
		{
			Name: "tls_decrypt_enabled",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id          = "f037e56e89293a057740de681ac9abbe"
  tls_decrypt_enabled = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    tls_decrypt = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "protocol_detection_enabled",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id                 = "f037e56e89293a057740de681ac9abbe"
  protocol_detection_enabled = false
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    protocol_detection = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "activity_log_enabled",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  activity_log_enabled = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    activity_log = {
      enabled = true
    }
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "url_browser_isolation_enabled",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id                    = "f037e56e89293a057740de681ac9abbe"
  url_browser_isolation_enabled = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = true
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "non_identity_browser_isolation_enabled",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id                             = "f037e56e89293a057740de681ac9abbe"
  non_identity_browser_isolation_enabled = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = true
    }
  }
}`,
		},
		{
			Name: "both url_browser_isolation_enabled and non_identity_browser_isolation_enabled",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id                             = "f037e56e89293a057740de681ac9abbe"
  url_browser_isolation_enabled          = true
  non_identity_browser_isolation_enabled = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = true
      non_identity_enabled          = true
    }
  }
}`,
		},
		{
			Name: "logging block creates separate resource",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  logging {
    settings_by_rule_type {
      dns {
        log_all    = true
        log_blocks = false
      }
      http {
        log_all    = true
        log_blocks = false
      }
      l4 {
        log_all    = true
        log_blocks = false
      }
    }
    redact_pii = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}
resource "cloudflare_zero_trust_gateway_logging" "test_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings_by_rule_type = {
    dns = {
      log_all    = true
      log_blocks = false
    }
    http = {
      log_all    = true
      log_blocks = false
    }
    l4 = {
      log_all    = true
      log_blocks = false
    }
  }
  redact_pii = true
}`,
		},
		{
			Name: "extended_email_matching",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  extended_email_matching {
    enabled = true
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    extended_email_matching = {
      enabled = true
    }
  }
}`,
		},
		{
			Name: "custom_certificate",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  custom_certificate {
    enabled = true
    id      = "cert-abc-123"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    custom_certificate = {
      enabled = true
      id      = "cert-abc-123"
    }
  }
}`,
		},
		{
			Name: "certificate",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  certificate {
    id = "cert-xyz-789"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    certificate = {
      id = "cert-xyz-789"
    }
  }
}`,
		},
		// TODO - Check with service team if this is captured in another resource / removal is intentional
		{
			Name: "ssh_session_log removed",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  ssh_session_log {
    public_key = "testvSXw8BfbrGCi0fhGiD/3yXk2SiV1Nzg2lru3oj0="
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		// TODO - Check with service team if this is captured in another resource / removal is intentional
		{
			Name: "payload_log removed",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  payload_log {
    public_key = "testPayloadKeyABC123XYZ"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}`,
		},
		{
			Name: "proxy block creates separate resource",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  proxy {
    tcp              = true
    udp              = true
    root_ca          = true
    virtual_ip       = false
    disable_for_time = 300
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
  }
}
resource "cloudflare_zero_trust_device_settings" "test_device_settings" {
  account_id                            = "f037e56e89293a057740de681ac9abbe"
  gateway_proxy_enabled                 = true
  gateway_udp_proxy_enabled             = true
  root_certificate_installation_enabled = true
  use_zt_virtual_ip                     = false
  disable_for_time                      = 300
}`,
		},
		{
			Name: "antivirus without notification_settings",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = true
    fail_closed            = false
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    browser_isolation = {
      url_browser_isolation_enabled = false
      non_identity_enabled          = false
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = true
      fail_closed            = false
    }
  }
}`,
		},
		{
			Name: "comprehensive config with all fields and multiple resources",
			Input: `
resource "cloudflare_teams_account" "test" {
  account_id                             = "f037e56e89293a057740de681ac9abbe"
  activity_log_enabled                   = true
  tls_decrypt_enabled                    = true
  protocol_detection_enabled             = false
  url_browser_isolation_enabled          = true
  non_identity_browser_isolation_enabled = true

  block_page {
    enabled          = true
    name             = "Custom Block Page"
    footer_text      = "Contact your IT department"
    header_text      = "Access Blocked"
    logo_path        = "https://example.com/logo.png"
    background_color = "#FF0000"
    mailto_address   = "it@example.com"
    mailto_subject   = "Access Request"
  }

  body_scanning {
    inspection_mode = "deep"
  }

  fips {
    tls = true
  }

  antivirus {
    enabled_download_phase = true
    enabled_upload_phase   = false
    fail_closed            = true

    notification_settings {
      enabled     = true
      message     = "File is being scanned for threats"
      support_url = "https://support.example.com/antivirus"
    }
  }

  extended_email_matching {
    enabled = true
  }

  custom_certificate {
    enabled = true
    id      = "cert-abc-123"
  }

  certificate {
    id = "cert-xyz-789"
  }

  logging {
    settings_by_rule_type {
      dns {
        log_all    = true
        log_blocks = false
      }
      http {
        log_all    = false
        log_blocks = true
      }
      l4 {
        log_all    = true
        log_blocks = true
      }
    }
    redact_pii = true
  }

  proxy {
    tcp              = true
    udp              = false
    root_ca          = true
    virtual_ip       = true
    disable_for_time = 600
  }

  ssh_session_log {
    public_key = "test-ssh-key-abc123"
  }

  payload_log {
    public_key = "test-payload-key-xyz789"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_settings" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  settings = {
    activity_log = {
      enabled = true
    }
    tls_decrypt = {
      enabled = true
    }
    protocol_detection = {
      enabled = false
    }
    browser_isolation = {
      url_browser_isolation_enabled = true
      non_identity_enabled          = true
    }
    block_page = {
      enabled          = true
      name             = "Custom Block Page"
      footer_text      = "Contact your IT department"
      header_text      = "Access Blocked"
      logo_path        = "https://example.com/logo.png"
      background_color = "#FF0000"
      mailto_address   = "it@example.com"
      mailto_subject   = "Access Request"
    }
    body_scanning = {
      inspection_mode = "deep"
    }
    fips = {
      tls = true
    }
    antivirus = {
      enabled_download_phase = true
      enabled_upload_phase   = false
      fail_closed            = true
      notification_settings = {
        enabled     = true
        support_url = "https://support.example.com/antivirus"
        msg         = "File is being scanned for threats"
      }
    }
    extended_email_matching = {
      enabled = true
    }
    custom_certificate = {
      enabled = true
      id      = "cert-abc-123"
    }
    certificate = {
      id = "cert-xyz-789"
    }
  }
}
resource "cloudflare_zero_trust_gateway_logging" "test_logging" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  settings_by_rule_type = {
    dns = {
      log_all    = true
      log_blocks = false
    }
    http = {
      log_all    = false
      log_blocks = true
    }
    l4 = {
      log_all    = true
      log_blocks = true
    }
  }
  redact_pii = true
}
resource "cloudflare_zero_trust_device_settings" "test_device_settings" {
  account_id                            = "f037e56e89293a057740de681ac9abbe"
  gateway_proxy_enabled                 = true
  gateway_udp_proxy_enabled             = false
  root_certificate_installation_enabled = true
  use_zt_virtual_ip                     = true
  disable_for_time                      = 600
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "minimal config",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "block_page",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "block_page": [
      {
        "enabled": true,
        "name": "Custom Block Page",
        "footer_text": "Contact your IT department",
        "header_text": "Access Blocked",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#FF0000",
        "mailto_address": "it@example.com",
        "mailto_subject": "Access Request"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "block_page": {
        "enabled": true,
        "name": "Custom Block Page",
        "footer_text": "Contact your IT department",
        "header_text": "Access Blocked",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#FF0000",
        "mailto_address": "it@example.com",
        "mailto_subject": "Access Request"
      }
    }
  }
}`,
		},
		{
			Name: "body_scanning",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "body_scanning": [
      {
        "inspection_mode": "deep"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "body_scanning": {
        "inspection_mode": "deep"
      }
    }
  }
}`,
		},
		{
			Name: "fips",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "fips": [
      {
        "tls": true
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "fips": {
        "tls": true
      }
    }
  }
}`,
		},
		{
			Name: "antivirus with notification_settings",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "antivirus": [
      {
        "enabled_download_phase": true,
        "enabled_upload_phase": false,
        "fail_closed": true,
        "notification_settings": [
          {
            "enabled": true,
            "message": "File is being scanned for threats",
            "support_url": "https://support.example.com/antivirus"
          }
        ]
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "antivirus": {
        "enabled_download_phase": true,
        "enabled_upload_phase": false,
        "fail_closed": true,
        "notification_settings": {
          "enabled": true,
          "support_url": "https://support.example.com/antivirus",
          "msg": "File is being scanned for threats"
        }
      }
    }
  }
}`,
		},
		{
			Name: "antivirus without notification_settings",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "antivirus": [
      {
        "enabled_download_phase": true,
        "enabled_upload_phase": true,
        "fail_closed": false
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "antivirus": {
        "enabled_download_phase": true,
        "enabled_upload_phase": true,
        "fail_closed": false
      }
    }
  }
}`,
		},
		{
			Name: "tls_decrypt_enabled",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tls_decrypt_enabled": true
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "tls_decrypt": {
        "enabled": true
      },
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "protocol_detection_enabled",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "protocol_detection_enabled": false
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "protocol_detection": {
        "enabled": false
      },
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "activity_log_enabled",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "activity_log_enabled": true
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "activity_log": {
        "enabled": true
      },
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "url_browser_isolation_enabled",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "url_browser_isolation_enabled": true
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": true,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "non_identity_browser_isolation_enabled",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "non_identity_browser_isolation_enabled": true
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": true
      }
    }
  }
}`,
		},
		{
			Name: "both url_browser_isolation_enabled and non_identity_browser_isolation_enabled",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "url_browser_isolation_enabled": true,
    "non_identity_browser_isolation_enabled": true
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": true,
        "non_identity_enabled": true
      }
    }
  }
}`,
		},
		{
			Name: "extended_email_matching",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "extended_email_matching": [
      {
        "enabled": true
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "extended_email_matching": {
        "enabled": true
      }
    }
  }
}`,
		},
		{
			Name: "custom_certificate",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "custom_certificate": [
      {
        "enabled": true,
        "id": "cert-abc-123"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "custom_certificate": {
        "enabled": true,
        "id": "cert-abc-123"
      }
    }
  }
}`,
		},
		{
			Name: "certificate",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "certificate": [
      {
        "id": "cert-xyz-789"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "certificate": {
        "id": "cert-xyz-789"
      }
    }
  }
}`,
		},
		{
			Name: "ssh_session_log removed",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "ssh_session_log": [
      {
        "public_key": "testvSXw8BfbrGCi0fhGiD/3yXk2SiV1Nzg2lru3oj0="
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "payload_log removed",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "payload_log": [
      {
        "public_key": "testPayloadKeyABC123XYZ"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "logging removed",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "logging": [
      {
        "settings_by_rule_type": [
          {
            "dns": [
              {
                "log_all": true,
                "log_blocks": false
              }
            ],
            "http": [
              {
                "log_all": true,
                "log_blocks": false
              }
            ],
            "l4": [
              {
                "log_all": true,
                "log_blocks": false
              }
            ]
          }
        ],
        "redact_pii": true
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "proxy removed",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "proxy": [
      {
        "tcp": true,
        "udp": true,
        "root_ca": true,
        "virtual_ip": false,
        "disable_for_time": 300
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      }
    }
  }
}`,
		},
		{
			Name: "comprehensive config with all fields",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "activity_log_enabled": true,
    "tls_decrypt_enabled": true,
    "protocol_detection_enabled": false,
    "url_browser_isolation_enabled": true,
    "non_identity_browser_isolation_enabled": true,
    "block_page": [
      {
        "enabled": true,
        "name": "Custom Block Page",
        "footer_text": "Contact your IT department",
        "header_text": "Access Blocked",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#FF0000",
        "mailto_address": "it@example.com",
        "mailto_subject": "Access Request"
      }
    ],
    "body_scanning": [
      {
        "inspection_mode": "deep"
      }
    ],
    "fips": [
      {
        "tls": true
      }
    ],
    "antivirus": [
      {
        "enabled_download_phase": true,
        "enabled_upload_phase": false,
        "fail_closed": true,
        "notification_settings": [
          {
            "enabled": true,
            "message": "File is being scanned for threats",
            "support_url": "https://support.example.com/antivirus"
          }
        ]
      }
    ],
    "extended_email_matching": [
      {
        "enabled": true
      }
    ],
    "custom_certificate": [
      {
        "enabled": true,
        "id": "cert-abc-123"
      }
    ],
    "certificate": [
      {
        "id": "cert-xyz-789"
      }
    ],
    "logging": [
      {
        "settings_by_rule_type": [
          {
            "dns": [
              {
                "log_all": true,
                "log_blocks": false
              }
            ],
            "http": [
              {
                "log_all": false,
                "log_blocks": true
              }
            ],
            "l4": [
              {
                "log_all": true,
                "log_blocks": true
              }
            ]
          }
        ],
        "redact_pii": true
      }
    ],
    "proxy": [
      {
        "tcp": true,
        "udp": false,
        "root_ca": true,
        "virtual_ip": true,
        "disable_for_time": 600
      }
    ],
    "ssh_session_log": [
      {
        "public_key": "test-ssh-key-abc123"
      }
    ],
    "payload_log": [
      {
        "public_key": "test-payload-key-xyz789"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "activity_log": {
        "enabled": true
      },
      "tls_decrypt": {
        "enabled": true
      },
      "protocol_detection": {
        "enabled": false
      },
      "browser_isolation": {
        "url_browser_isolation_enabled": true,
        "non_identity_enabled": true
      },
      "block_page": {
        "enabled": true,
        "name": "Custom Block Page",
        "footer_text": "Contact your IT department",
        "header_text": "Access Blocked",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#FF0000",
        "mailto_address": "it@example.com",
        "mailto_subject": "Access Request"
      },
      "body_scanning": {
        "inspection_mode": "deep"
      },
      "fips": {
        "tls": true
      },
      "antivirus": {
        "enabled_download_phase": true,
        "enabled_upload_phase": false,
        "fail_closed": true,
        "notification_settings": {
          "enabled": true,
          "support_url": "https://support.example.com/antivirus",
          "msg": "File is being scanned for threats"
        }
      },
      "extended_email_matching": {
        "enabled": true
      },
      "custom_certificate": {
        "enabled": true,
        "id": "cert-abc-123"
      },
      "certificate": {
        "id": "cert-xyz-789"
      }
    }
  }
}`,
		},
		{
			Name: "block_page with empty optional fields (not in HCL) - should be transformed to null",
			Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_teams_account",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "block_page": [
          {
            "enabled": true,
            "name": "Test Block Page",
            "footer_text": "Contact IT",
            "header_text": "Blocked",
            "logo_path": "https://example.com/logo.png",
            "background_color": "#FF0000",
            "mailto_address": "",
            "mailto_subject": "",
            "suppress_footer": false,
            "include_context": false,
            "mode": "customized_block_page",
            "target_uri": ""
          }
        ]
      }
    }]
  }]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_settings",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "settings": {
          "browser_isolation": {
            "url_browser_isolation_enabled": false,
            "non_identity_enabled": false
          },
          "block_page": {
            "enabled": true,
            "name": "Test Block Page",
            "footer_text": "Contact IT",
            "header_text": "Blocked",
            "logo_path": "https://example.com/logo.png",
            "background_color": "#FF0000",
            "mailto_address": null,
            "mailto_subject": null,
            "suppress_footer": null,
            "include_context": null,
            "target_uri": null
          }
        }
      }
    }]
  }]
}`,
			Config: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  block_page {
    enabled = true
    name = "Test Block Page"
    footer_text = "Contact IT"
    header_text = "Blocked"
    logo_path = "https://example.com/logo.png"
    background_color = "#FF0000"
  }
}`,
		},
		{
			Name: "block_page with explicitly set empty fields - transformed to null (known limitation)",
			Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_teams_account",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "block_page": [
          {
            "enabled": true,
            "name": "Test",
            "footer_text": "Footer",
            "mailto_address": "",
            "mailto_subject": ""
          }
        ]
      }
    }]
  }]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_settings",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "settings": {
          "browser_isolation": {
            "url_browser_isolation_enabled": false,
            "non_identity_enabled": false
          },
          "block_page": {
            "enabled": true,
            "name": "Test",
            "footer_text": "Footer",
            "mailto_address": null,
            "mailto_subject": null
          }
        }
      }
    }]
  }]
}`,
			Config: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  block_page {
    enabled = true
    name = "Test"
    footer_text = "Footer"
    mailto_address = ""
    mailto_subject = ""
  }
}`,
		},
		{
			Name: "body_scanning with empty optional fields - should be transformed to null",
			Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_teams_account",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "body_scanning": [
          {
            "inspection_mode": "deep",
            "some_empty_field": ""
          }
        ]
      }
    }]
  }]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_settings",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "settings": {
          "browser_isolation": {
            "url_browser_isolation_enabled": false,
            "non_identity_enabled": false
          },
          "body_scanning": {
            "inspection_mode": "deep",
            "some_empty_field": null
          }
        }
      }
    }]
  }]
}`,
			Config: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  body_scanning {
    inspection_mode = "deep"
  }
}`,
		},
		{
			Name: "fips with empty optional fields - should be transformed to null",
			Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_teams_account",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "fips": [
          {
            "tls": true,
            "empty_field": "",
            "false_field": false
          }
        ]
      }
    }]
  }]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_gateway_settings",
    "name": "test",
    "instances": [{
      "schema_version": 0,
      "attributes": {
        "account_id": "f037e56e89293a057740de681ac9abbe",
        "settings": {
          "browser_isolation": {
            "url_browser_isolation_enabled": false,
            "non_identity_enabled": false
          },
          "fips": {
            "tls": true,
            "empty_field": null,
            "false_field": null
          }
        }
      }
    }]
  }]
}`,
			Config: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  fips {
    tls = true
  }
}`,
		},
		{
			Name: "block_page with API-computed fields - should be removed (not in v4 schema)",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "block_page": [
      {
        "enabled": true,
        "name": "Test Block Page",
        "header_text": "Access Blocked",
        "footer_text": "Contact IT",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#000000",
        "mailto_address": "security@example.com",
        "mailto_subject": "Access Request",
        "mode": "customized_block_page",
        "version": 0,
        "read_only": false,
        "source_account": "abc123",
        "include_context": false,
        "suppress_footer": false,
        "target_uri": ""
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "block_page": {
        "enabled": true,
        "name": "Test Block Page",
        "header_text": "Access Blocked",
        "footer_text": "Contact IT",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#000000",
        "mailto_address": "security@example.com",
        "mailto_subject": "Access Request",
        "include_context": null,
        "suppress_footer": null,
        "target_uri": null
      }
    }
  }
}`,
		},
		{
			Name: "block_page with API-computed fields and empty values - computed fields removed, empty values to null",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "block_page": [
      {
        "enabled": true,
        "name": "Test Block Page",
        "header_text": "Blocked",
        "footer_text": "Contact IT",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#1a1a1a",
        "mailto_address": "",
        "mailto_subject": "",
        "mode": "customized_block_page",
        "version": 5,
        "read_only": false,
        "source_account": "",
        "include_context": false,
        "suppress_footer": false,
        "target_uri": ""
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "block_page": {
        "enabled": true,
        "name": "Test Block Page",
        "header_text": "Blocked",
        "footer_text": "Contact IT",
        "logo_path": "https://example.com/logo.png",
        "background_color": "#1a1a1a",
        "mailto_address": null,
        "mailto_subject": null,
        "include_context": null,
        "suppress_footer": null,
        "target_uri": null
      }
    }
  }
}`,
			Config: `
resource "cloudflare_teams_account" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  block_page {
    enabled          = true
    name             = "Test Block Page"
    header_text      = "Blocked"
    footer_text      = "Contact IT"
    logo_path        = "https://example.com/logo.png"
    background_color = "#1a1a1a"
  }
}`,
		},
		{
			Name: "extended_email_matching with API-computed fields - should be removed (not in v4 schema)",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "extended_email_matching": [
      {
        "enabled": true,
        "version": 2,
        "read_only": false,
        "source_account": "xyz789"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "extended_email_matching": {
        "enabled": true
      }
    }
  }
}`,
		},
		{
			Name: "comprehensive with block_page and extended_email_matching computed fields - all removed",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "activity_log_enabled": true,
    "tls_decrypt_enabled": true,
    "block_page": [
      {
        "enabled": true,
        "name": "Corporate Block Page",
        "header_text": "Access Denied",
        "footer_text": "Contact Security",
        "logo_path": "https://corp.example.com/logo.png",
        "background_color": "#FF0000",
        "mailto_address": "security@example.com",
        "mailto_subject": "Access Required",
        "mode": "customized_block_page",
        "version": 10,
        "read_only": true,
        "source_account": "parent-account-123",
        "include_context": false,
        "suppress_footer": false,
        "target_uri": ""
      }
    ],
    "extended_email_matching": [
      {
        "enabled": true,
        "version": 5,
        "read_only": true,
        "source_account": "parent-account-123"
      }
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "settings": {
      "activity_log": {
        "enabled": true
      },
      "tls_decrypt": {
        "enabled": true
      },
      "browser_isolation": {
        "url_browser_isolation_enabled": false,
        "non_identity_enabled": false
      },
      "block_page": {
        "enabled": true,
        "name": "Corporate Block Page",
        "header_text": "Access Denied",
        "footer_text": "Contact Security",
        "logo_path": "https://corp.example.com/logo.png",
        "background_color": "#FF0000",
        "mailto_address": "security@example.com",
        "mailto_subject": "Access Required",
        "include_context": null,
        "suppress_footer": null,
        "target_uri": null
      },
      "extended_email_matching": {
        "enabled": true
      }
    }
  }
}`,
			Config: `
resource "cloudflare_teams_account" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  activity_log_enabled = true
  tls_decrypt_enabled  = true

  block_page {
    enabled          = true
    name             = "Corporate Block Page"
    header_text      = "Access Denied"
    footer_text      = "Contact Security"
    logo_path        = "https://corp.example.com/logo.png"
    background_color = "#FF0000"
    mailto_address   = "security@example.com"
    mailto_subject   = "Access Required"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}
