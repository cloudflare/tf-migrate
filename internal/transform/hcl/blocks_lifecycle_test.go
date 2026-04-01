package hcl

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseBody is a test helper that parses an HCL snippet and returns the first
// resource block's body, ready for mutation.
func parseBody(t *testing.T, src string) (*hclwrite.File, *hclwrite.Body) {
	t.Helper()
	f, diags := hclwrite.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)
	for _, block := range f.Body().Blocks() {
		if block.Type() == "resource" {
			return f, block.Body()
		}
	}
	t.Fatal("no resource block found in test input")
	return nil, nil
}

// ignoreChangesValue extracts the raw string value of ignore_changes from a
// lifecycle block inside body, for easy assertion.
func ignoreChangesValue(body *hclwrite.Body) string {
	for _, block := range body.Blocks() {
		if block.Type() == "lifecycle" {
			attr := block.Body().GetAttribute("ignore_changes")
			if attr == nil {
				return ""
			}
			return strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
		}
	}
	return ""
}

func TestAddLifecycleIgnoreChanges(t *testing.T) {
	t.Run("no existing lifecycle block — creates one with given attrs", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"
}`)
		AddLifecycleIgnoreChanges(body, "certificate", "private_key")
		val := ignoreChangesValue(body)
		assert.Equal(t, "[certificate, private_key]", val)
	})

	t.Run("existing lifecycle block without ignore_changes — adds it", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"
  lifecycle {
    create_before_destroy = true
  }
}`)
		AddLifecycleIgnoreChanges(body, "certificate")
		val := ignoreChangesValue(body)
		assert.Equal(t, "[certificate]", val)

		// Existing lifecycle attrs must be preserved
		output := string(hclwrite.Format(body.BuildTokens(nil).Bytes()))
		assert.Contains(t, output, "create_before_destroy")
	})

	t.Run("existing ignore_changes — new names are merged in", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"
  lifecycle {
    ignore_changes = [bundle_method]
  }
}`)
		AddLifecycleIgnoreChanges(body, "certificate", "private_key")
		val := ignoreChangesValue(body)
		assert.Equal(t, "[bundle_method, certificate, private_key]", val)
	})

	t.Run("duplicate names are deduplicated", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"
  lifecycle {
    ignore_changes = [certificate]
  }
}`)
		AddLifecycleIgnoreChanges(body, "certificate")
		val := ignoreChangesValue(body)
		assert.Equal(t, "[certificate]", val)
	})

	t.Run("empty attrNames is a no-op", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"
}`)
		AddLifecycleIgnoreChanges(body)
		// No lifecycle block should have been created
		for _, block := range body.Blocks() {
			assert.NotEqual(t, "lifecycle", block.Type(), "unexpected lifecycle block created")
		}
	})

	t.Run("single attr — no existing lifecycle", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_zero_trust_access_mtls_certificate" "example" {
  account_id = "abc123"
  name       = "test"
}`)
		AddLifecycleIgnoreChanges(body, "certificate")
		val := ignoreChangesValue(body)
		assert.Equal(t, "[certificate]", val)
	})

	t.Run("merges into existing ignore_changes with single entry", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_zero_trust_access_mtls_certificate" "example" {
  account_id = "abc123"
  lifecycle {
    ignore_changes = [associated_hostnames]
  }
}`)
		AddLifecycleIgnoreChanges(body, "certificate")
		val := ignoreChangesValue(body)
		assert.Equal(t, "[associated_hostnames, certificate]", val)
	})

	t.Run("existing all remains exclusive when merging", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_zero_trust_access_mtls_certificate" "example" {
  account_id = "abc123"
  lifecycle {
    ignore_changes = all
  }
}`)
		AddLifecycleIgnoreChanges(body, "certificate")
		val := ignoreChangesValue(body)
		assert.Equal(t, "all", val)
	})

	t.Run("incoming all overrides existing list", func(t *testing.T) {
		_, body := parseBody(t, `
resource "cloudflare_custom_ssl" "example" {
  zone_id = "abc123"
  lifecycle {
    ignore_changes = [certificate]
  }
}`)
		AddLifecycleIgnoreChanges(body, "all")
		val := ignoreChangesValue(body)
		assert.Equal(t, "all", val)
	})
}
