package zones

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "filter with account_id only",
			Input: `data "cloudflare_zones" "test" {
  filter {
    account_id = "f037e56e89293a057740de681ac9abbe"
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
}`,
		},
		{
			Name: "filter with name only",
			Input: `data "cloudflare_zones" "test" {
  filter {
    name = "example.com"
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  name = "example.com"
}`,
		},
		{
			Name: "filter with status only",
			Input: `data "cloudflare_zones" "test" {
  filter {
    status = "active"
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  status = "active"
}`,
		},
		{
			Name: "filter with account_id and name",
			Input: `data "cloudflare_zones" "test" {
  filter {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "example.com"
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  name = "example.com"
}`,
		},
		{
			Name: "filter with all supported fields",
			Input: `data "cloudflare_zones" "test" {
  filter {
    account_id = "f037e56e89293a057740de681ac9abbe"
    name       = "example.com"
    status     = "active"
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  name   = "example.com"
  status = "active"
}`,
		},
		{
			Name: "filter with dropped fields (lookup_type, match, paused)",
			Input: `data "cloudflare_zones" "test" {
  filter {
    account_id  = "f037e56e89293a057740de681ac9abbe"
    name        = "example"
    lookup_type = "contains"
    match       = "^not-"
    paused      = false
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  account = {
    id = "f037e56e89293a057740de681ac9abbe"
  }
  name = "example"
}`,
		},
		{
			Name: "variable references preserved",
			Input: `data "cloudflare_zones" "test" {
  filter {
    account_id = var.account_id
    name       = var.zone_name
    status     = var.status
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  account = {
    id = var.account_id
  }
  name   = var.zone_name
  status = var.status
}`,
		},
		{
			Name: "local references preserved",
			Input: `data "cloudflare_zones" "test" {
  filter {
    account_id = local.account_id
    name       = local.zone_name
  }
}`,
			Expected: `data "cloudflare_zones" "test" {
  account = {
    id = local.account_id
  }
  name = local.zone_name
}`,
		},
		{
			Name: "multiple datasources",
			Input: `data "cloudflare_zones" "by_account" {
  filter {
    account_id = "abc123"
  }
}

data "cloudflare_zones" "by_name" {
  filter {
    name = "example.com"
  }
}

data "cloudflare_zones" "by_both" {
  filter {
    account_id = "abc123"
    name       = "example.com"
    status     = "active"
  }
}`,
			Expected: `data "cloudflare_zones" "by_account" {
  account = {
    id = "abc123"
  }
}

data "cloudflare_zones" "by_name" {
  name = "example.com"
}

data "cloudflare_zones" "by_both" {
  account = {
    id = "abc123"
  }
  name   = "example.com"
  status = "active"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "zones array to result array",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "checksum123",
    "zones": [
      {"id": "zone1", "name": "example.com"},
      {"id": "zone2", "name": "test.com"}
    ]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "checksum123",
    "result": [
      {"id": "zone1", "name": "example.com"},
      {"id": "zone2", "name": "test.com"}
    ]
  }
}`,
		},
		{
			Name: "empty zones array",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "checksum123",
    "zones": []
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "checksum123",
    "result": []
  }
}`,
		},
		{
			Name: "null zones field",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "checksum123",
    "zones": null
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "checksum123"
  }
}`,
		},
		{
			Name: "minimal state with no zones",
			Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "checksum123"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "checksum123"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}

// Note: Warnings for dropped fields (lookup_type, match, paused) are added to ctx.Diagnostics
// They can be verified through integration testing or manual testing
