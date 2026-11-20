package zone_dnssec

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Basic zone_dnssec with minimal fields",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
  status  = "active"
}`,
			},
			{
				Name: "Multiple zone_dnssec resources in one file",
				Input: `
resource "cloudflare_zone_dnssec" "example1" {
  zone_id = "abc123"
}

resource "cloudflare_zone_dnssec" "example2" {
  zone_id = "def456"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example1" {
  zone_id = "abc123"
  status  = "active"
}

resource "cloudflare_zone_dnssec" "example2" {
  zone_id = "def456"
  status  = "active"
}`,
			},
			{
				Name: "Zone DNSSEC with modified_on field (should be removed)",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id     = "abc123"
  modified_on = "2024-01-15T10:30:00Z"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
  status  = "active"
}`,
			},
			{
				Name: "Existing status is left unchanged",
				Input: `
resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
  status  = "disabled"
}`,
				Expected: `resource "cloudflare_zone_dnssec" "example" {
  zone_id = "abc123"
  status  = "disabled"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StatusAddedWarns", func(t *testing.T) {
		for _, tc := range []struct {
			input string
			warns bool
		}{
			{"resource \"cloudflare_zone_dnssec\" \"example\" {\n  zone_id = \"abc123\"\n}\n", true},
			{"resource \"cloudflare_zone_dnssec\" \"example\" {\n  zone_id = \"abc123\"\n  status  = \"disabled\"\n}\n", false},
		} {
			file, diags := hclwrite.ParseConfig([]byte(tc.input), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors())
			ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
			_, err := migrator.TransformConfig(ctx, file.Body().Blocks()[0])
			require.NoError(t, err)
			if !tc.warns {
				assert.Empty(t, ctx.Diagnostics)
				continue
			}
			require.Len(t, ctx.Diagnostics, 1)
			assert.Equal(t, hcl.DiagWarning, ctx.Diagnostics[0].Severity)
			assert.Contains(t, ctx.Diagnostics[0].Summary, "cloudflare_zone_dnssec.example")
			assert.Contains(t, ctx.Diagnostics[0].Detail, `status = "disabled"`)
		}
	})
}
