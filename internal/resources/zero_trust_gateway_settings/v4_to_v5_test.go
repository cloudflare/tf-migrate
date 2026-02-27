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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
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
}
moved {
  from = cloudflare_teams_account.test
  to   = cloudflare_zero_trust_gateway_settings.test
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}

