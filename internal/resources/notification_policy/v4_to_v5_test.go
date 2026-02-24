package notification_policy

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "basic notification policy",
			Input: `
		resource "cloudflare_notification_policy" "example" {
		 account_id = "f037e56e89293a057740de681ac9abbe"
		 alert_type = "universal_ssl_event_type"
         enabled = true
         name = "name"
		 description = "description"
		}`,
			Expected: `
		resource "cloudflare_notification_policy" "example" {
		 account_id = "f037e56e89293a057740de681ac9abbe"
		 alert_type = "universal_ssl_event_type"
         enabled = true
         name = "name"
		 description = "description"
		}`,
		},
		{
			Name: "all v4 filter fields",
			Input: `
resource "cloudflare_notification_policy" "all_filters" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "All Filters Test"
  alert_type = "advanced_ddos_attack_l7_alert"
  enabled    = true

  filters {
    actions                    = ["block", "challenge"]
    airport_code               = ["SJC", "LAX"]
    affected_components        = ["API", "Dashboard"]
    status                     = ["enabled", "disabled"]
    health_check_id            = ["healthcheck-1", "healthcheck-2"]
    zones                      = ["zone-1", "zone-2"]
    services                   = ["service-1", "service-2"]
    product                    = ["worker_requests", "worker_durable_objects_requests"]
    limit                      = ["100", "200"]
    enabled                    = ["true", "false"]
    pool_id                    = ["pool-1", "pool-2"]
    slo                        = ["99.9", "99.99"]
    where                      = ["filter1", "filter2"]
    group_by                   = ["zone", "host"]
    alert_trigger_preferences  = ["slo"]
    requests_per_second        = ["1000", "2000"]
    target_zone_name           = ["example.com", "test.com"]
    target_hostname            = ["www.example.com", "api.example.com"]
    target_ip                  = ["192.0.2.0/24", "198.51.100.0/24"]
    packets_per_second         = ["50000", "100000"]
    protocol                   = ["tcp", "udp"]
    project_id                 = ["project-1", "project-2"]
    environment                = ["ENVIRONMENT_PREVIEW", "ENVIRONMENT_PRODUCTION"]
    event                      = ["EVENT_DEPLOYMENT_STARTED", "EVENT_DEPLOYMENT_FAILED"]
    event_source               = ["pool", "origin"]
    new_health                 = ["healthy", "unhealthy"]
    input_id                   = ["input-1", "input-2"]
    event_type                 = ["type-1", "type-2"]
    megabits_per_second        = ["500", "1000"]
    incident_impact            = ["INCIDENT_IMPACT_MINOR", "INCIDENT_IMPACT_MAJOR"]
    new_status                 = ["healthy", "down"]
    selectors                  = ["selector-1", "selector-2"]
    tunnel_id                  = ["tunnel-1", "tunnel-2"]
    tunnel_name                = ["tunnel-name-1", "tunnel-name-2"]
  }
}`,
			Expected: `
resource "cloudflare_notification_policy" "all_filters" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "All Filters Test"
  alert_type = "advanced_ddos_attack_l7_alert"
  enabled    = true

  filters = {
    actions                   = ["block", "challenge"]
    airport_code              = ["SJC", "LAX"]
    affected_components       = ["API", "Dashboard"]
    status                    = ["enabled", "disabled"]
    health_check_id           = ["healthcheck-1", "healthcheck-2"]
    zones                     = ["zone-1", "zone-2"]
    services                  = ["service-1", "service-2"]
    product                   = ["worker_requests", "worker_durable_objects_requests"]
    limit                     = ["100", "200"]
    enabled                   = ["true", "false"]
    pool_id                   = ["pool-1", "pool-2"]
    slo                       = ["99.9", "99.99"]
    where                     = ["filter1", "filter2"]
    group_by                  = ["zone", "host"]
    alert_trigger_preferences = ["slo"]
    requests_per_second       = ["1000", "2000"]
    target_zone_name          = ["example.com", "test.com"]
    target_hostname           = ["www.example.com", "api.example.com"]
    target_ip                 = ["192.0.2.0/24", "198.51.100.0/24"]
    packets_per_second        = ["50000", "100000"]
    protocol                  = ["tcp", "udp"]
    project_id                = ["project-1", "project-2"]
    environment               = ["ENVIRONMENT_PREVIEW", "ENVIRONMENT_PRODUCTION"]
    event                     = ["EVENT_DEPLOYMENT_STARTED", "EVENT_DEPLOYMENT_FAILED"]
    event_source              = ["pool", "origin"]
    new_health                = ["healthy", "unhealthy"]
    input_id                  = ["input-1", "input-2"]
    event_type                = ["type-1", "type-2"]
    megabits_per_second       = ["500", "1000"]
    incident_impact           = ["INCIDENT_IMPACT_MINOR", "INCIDENT_IMPACT_MAJOR"]
    new_status                = ["healthy", "down"]
    selectors                 = ["selector-1", "selector-2"]
    tunnel_id                 = ["tunnel-1", "tunnel-2"]
    tunnel_name               = ["tunnel-name-1", "tunnel-name-2"]
  }
}`,
		},
		{
			Name: "all three integration types with multiple integrations",
			Input: `
resource "cloudflare_notification_policy" "multi_integration" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Multi Integration Test"
  alert_type = "universal_ssl_event_type"
  enabled    = true

  email_integration {
    id   = "email-123"
    name = "primary@example.com"
  }

  email_integration {
    id   = "email-456"
    name = "secondary@example.com"
  }

  email_integration {
    id   = "email-789"
    name = "tertiary@example.com"
  }

  webhooks_integration {
    id   = "webhook-111"
    name = "Primary Webhook"
  }

  webhooks_integration {
    id   = "webhook-222"
    name = "Secondary Webhook"
  }

  pagerduty_integration {
    id   = "pagerduty-333"
    name = "Production PagerDuty"
  }

  pagerduty_integration {
    id   = "pagerduty-444"
    name = "Staging PagerDuty"
  }
}`,
			Expected: `
resource "cloudflare_notification_policy" "multi_integration" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Multi Integration Test"
  alert_type = "universal_ssl_event_type"
  enabled    = true

  mechanisms = {
    email = [{
      id = "email-123"
      }, {
      id = "email-456"
      }, {
      id = "email-789"
    }]
    webhooks = [{
      id = "webhook-111"
      }, {
      id = "webhook-222"
    }]
    pagerduty = [{
      id = "pagerduty-333"
      }, {
      id = "pagerduty-444"
    }]
  }
}`,
		},
		{
			Name: "enabled=false preservation (CRITICAL)",
			Input: `
resource "cloudflare_notification_policy" "disabled" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Disabled Policy"
  alert_type = "universal_ssl_event_type"
  enabled    = false

  email_integration {
    id = "email-999"
  }
}`,
			Expected: `
resource "cloudflare_notification_policy" "disabled" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Disabled Policy"
  alert_type = "universal_ssl_event_type"
  enabled    = false

  mechanisms = {
    email = [{
      id = "email-999"
    }]
  }
}`,
		},
		{
			Name: "combined filters and all three integration types",
			Input: `
resource "cloudflare_notification_policy" "combined" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Combined Test"
  alert_type = "load_balancing_health_alert"
  enabled    = true

  filters {
    zones           = ["zone-abc", "zone-def"]
    pool_id         = ["pool-123"]
    event_source    = ["pool", "origin"]
    new_health      = ["unhealthy"]
  }

  email_integration {
    id   = "email-111"
    name = "ops@example.com"
  }

  webhooks_integration {
    id   = "webhook-222"
    name = "Ops Webhook"
  }

  pagerduty_integration {
    id   = "pagerduty-333"
    name = "Ops PagerDuty"
  }
}`,
			Expected: `
resource "cloudflare_notification_policy" "combined" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Combined Test"
  alert_type = "load_balancing_health_alert"
  enabled    = true

  filters = {
    zones        = ["zone-abc", "zone-def"]
    pool_id      = ["pool-123"]
    event_source = ["pool", "origin"]
    new_health   = ["unhealthy"]
  }
  mechanisms = {
    email = [{
      id = "email-111"
    }]
    webhooks = [{
      id = "webhook-222"
    }]
    pagerduty = [{
      id = "pagerduty-333"
    }]
  }
}`,
		},
		{
			Name: "empty/missing integration IDs are skipped",
			Input: `
resource "cloudflare_notification_policy" "missing_ids" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Missing IDs Test"
  alert_type = "universal_ssl_event_type"
  enabled    = true

  email_integration {
    name = "no-id@example.com"
  }

  email_integration {
    id   = "email-valid"
    name = "valid@example.com"
  }

  webhooks_integration {
    name = "No ID Webhook"
  }
}`,
			Expected: `
resource "cloudflare_notification_policy" "missing_ids" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Missing IDs Test"
  alert_type = "universal_ssl_event_type"
  enabled    = true

  mechanisms = {
    email = [{
      id = "email-valid"
    }]
    webhooks = []
  }
}`,
		},
		{
			Name: "empty filters block",
			Input: `
resource "cloudflare_notification_policy" "empty_filters" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Empty Filters"
  alert_type = "universal_ssl_event_type"
  enabled    = true

  filters {
  }

  email_integration {
    id = "email-empty"
  }
}`,
			Expected: `
resource "cloudflare_notification_policy" "empty_filters" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  name       = "Empty Filters"
  alert_type = "universal_ssl_event_type"
  enabled    = true

  filters = {}
  mechanisms = {
    email = [{
      id = "email-empty"
    }]
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

// TestV4ToV5TransformationState_Removed is a skip marker test.
// State transformation tests have been removed because state migration
// is now handled by the provider's StateUpgraders (UpgradeFromV4).
func TestV4ToV5TransformationState_Removed(t *testing.T) {
	t.Skip("State transformation tests removed - state migration is now handled by provider's StateUpgraders")
}
