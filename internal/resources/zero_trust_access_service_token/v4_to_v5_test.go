package zero_trust_access_service_token

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic resource with all fields - min_days_for_renewal removed",
				Input: `
resource "cloudflare_zero_trust_access_service_token" "test" {
  account_id                         = "account123"
  name                               = "my_token"
  duration                           = "8760h"
  min_days_for_renewal               = 30
  client_secret_version              = 2
  previous_client_secret_expires_at  = "2024-12-31T23:59:59Z"
}`,
				Expected: `resource "cloudflare_zero_trust_access_service_token" "test" {
  account_id                        = "account123"
  name                              = "my_token"
  duration                          = "8760h"
  client_secret_version             = 2
  previous_client_secret_expires_at = "2024-12-31T23:59:59Z"
}

moved {
  from = cloudflare_access_service_token.test
  to   = cloudflare_zero_trust_access_service_token.test
}`,
			},
			{
				Name: "Minimal resource with only required fields",
				Input: `
resource "cloudflare_zero_trust_access_service_token" "minimal" {
  zone_id = "zone456"
  name    = "minimal_token"
}`,
				Expected: `resource "cloudflare_zero_trust_access_service_token" "minimal" {
  zone_id = "zone456"
  name    = "minimal_token"
}

moved {
  from = cloudflare_access_service_token.minimal
  to   = cloudflare_zero_trust_access_service_token.minimal
}`,
			},
			{
				Name: "Legacy resource name - cloudflare_access_service_token",
				Input: `
resource "cloudflare_access_service_token" "legacy" {
  account_id = "account123"
  name       = "legacy_token"
}`,
				Expected: `resource "cloudflare_zero_trust_access_service_token" "legacy" {
  account_id = "account123"
  name       = "legacy_token"
}

moved {
  from = cloudflare_access_service_token.legacy
  to   = cloudflare_zero_trust_access_service_token.legacy
}`,
			},
			{
				Name: "Resource with min_days_for_renewal = 0 (should still be removed)",
				Input: `
resource "cloudflare_zero_trust_access_service_token" "test" {
  account_id           = "account123"
  name                 = "test_token"
  min_days_for_renewal = 0
}`,
				Expected: `resource "cloudflare_zero_trust_access_service_token" "test" {
  account_id = "account123"
  name       = "test_token"
}

moved {
  from = cloudflare_access_service_token.test
  to   = cloudflare_zero_trust_access_service_token.test
}`,
			},
			{
				Name: "Multiple resources in one file",
				Input: `
resource "cloudflare_zero_trust_access_service_token" "first" {
  account_id           = "account123"
  name                 = "first_token"
  min_days_for_renewal = 30
}

resource "cloudflare_zero_trust_access_service_token" "second" {
  zone_id              = "zone456"
  name                 = "second_token"
  min_days_for_renewal = 60
}`,
				Expected: `resource "cloudflare_zero_trust_access_service_token" "first" {
  account_id = "account123"
  name       = "first_token"
}

moved {
  from = cloudflare_access_service_token.first
  to   = cloudflare_zero_trust_access_service_token.first
}

resource "cloudflare_zero_trust_access_service_token" "second" {
  zone_id = "zone456"
  name    = "second_token"
}

moved {
  from = cloudflare_access_service_token.second
  to   = cloudflare_zero_trust_access_service_token.second
}`,
			},
			{
				Name: "Resource without min_days_for_renewal (should be unchanged)",
				Input: `
resource "cloudflare_zero_trust_access_service_token" "no_renewal" {
  account_id = "account123"
  name       = "no_renewal_token"
  duration   = "17520h"
}`,
				Expected: `resource "cloudflare_zero_trust_access_service_token" "no_renewal" {
  account_id = "account123"
  name       = "no_renewal_token"
  duration   = "17520h"
}

moved {
  from = cloudflare_access_service_token.no_renewal
  to   = cloudflare_zero_trust_access_service_token.no_renewal
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
