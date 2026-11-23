package zero_trust_access_service_token

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"

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

resource "cloudflare_zero_trust_access_service_token" "second" {
  zone_id = "zone456"
  name    = "second_token"
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
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("RenewalRemovedWarns", func(t *testing.T) {
		for _, tc := range []struct {
			body  string
			warns bool
		}{
			{"  duration             = \"8760h\"\n  min_days_for_renewal = 30\n", true},
			{"  min_days_for_renewal = var.days\n", true},
			{"  duration             = \"forever\"\n  min_days_for_renewal = 30\n", false},
			{"  duration             = \"8760h\"\n  min_days_for_renewal = 0\n", false},
			{"  duration = \"8760h\"\n", false},
		} {
			input := "resource \"cloudflare_access_service_token\" \"t\" {\n  account_id = \"a\"\n  name = \"t\"\n" + tc.body + "}\n"
			file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())
			ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
			_, err := migrator.TransformConfig(ctx, file.Body().Blocks()[0])
			require.NoError(t, err)
			if !tc.warns {
				assert.Empty(t, ctx.Diagnostics, tc.body)
				continue
			}
			require.Len(t, ctx.Diagnostics, 1, tc.body)
			assert.Equal(t, hcl.DiagWarning, ctx.Diagnostics[0].Severity)
			assert.Contains(t, ctx.Diagnostics[0].Summary, "cloudflare_zero_trust_access_service_token.t")
			assert.Contains(t, ctx.Diagnostics[0].Detail, `duration = "forever"`)
		}
	})
}
