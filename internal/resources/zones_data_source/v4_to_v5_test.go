package zones_data_source

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestZonesDataSource_ConfigTransform(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "minimal - name only",
			Input: `data "cloudflare_zones" "example" {
  filter {
    name = "example.com"
  }
}
`,
			Expected: `data "cloudflare_zones" "example" {
  name = "example.com"
}
`,
		},
		{
			Name: "account_id only",
			Input: `data "cloudflare_zones" "example" {
  filter {
    account_id = "abc123"
  }
}
`,
			Expected: `data "cloudflare_zones" "example" {
  account = {
    id = "abc123"
  }
}
`,
		},
		{
			Name: "all migratable fields",
			Input: `data "cloudflare_zones" "example" {
  filter {
    name       = "example.com"
    account_id = "abc123"
    status     = "active"
  }
}
`,
			Expected: `data "cloudflare_zones" "example" {
  name   = "example.com"
  status = "active"
  account = {
    id = "abc123"
  }
}
`,
		},
		{
			Name: "with non-migratable fields",
			Input: `data "cloudflare_zones" "example" {
  filter {
    name        = "example.com"
    lookup_type = "contains"
    match       = ".*\\.com$"
    paused      = false
  }
}
`,
			Expected: `data "cloudflare_zones" "example" {
  name = "example.com"
}
`,
		},
		{
			Name: "with variables",
			Input: `data "cloudflare_zones" "example" {
  filter {
    name       = var.zone_name
    account_id = var.account_id
  }
}
`,
			Expected: `data "cloudflare_zones" "example" {
  name = var.zone_name
  account = {
    id = var.account_id
  }
}
`,
		},
		{
			Name: "multiple datasources",
			Input: `data "cloudflare_zones" "active" {
  filter {
    status = "active"
  }
}

data "cloudflare_zones" "by_account" {
  filter {
    account_id = "abc123"
  }
}
`,
			Expected: `data "cloudflare_zones" "active" {
  status = "active"
}

data "cloudflare_zones" "by_account" {
  account = {
    id = "abc123"
  }
}
`,
		},
		{
			Name: "empty filter block",
			Input: `data "cloudflare_zones" "all" {
  filter {
  }
}
`,
			Expected: `data "cloudflare_zones" "all" {
}
`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestZonesDataSource_StateTransform(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "basic zones state",
			Input: `{
  "version": 4,
  "terraform_version": "1.0.0",
  "resources": [
    {
      "type": "cloudflare_zones",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "checksum123",
            "filter": [
              {
                "account_id": "abc123",
                "name": "example.com",
                "status": "active",
                "paused": false,
                "lookup_type": "exact",
                "match": null
              }
            ],
            "zones": [
              {
                "id": "zone1",
                "name": "example1.com"
              },
              {
                "id": "zone2",
                "name": "example2.com"
              }
            ]
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.0.0",
  "resources": [
    {
      "type": "cloudflare_zones",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "checksum123",
            "filter": [
              {
                "account_id": "abc123",
                "name": "example.com",
                "status": "active",
                "paused": false,
                "lookup_type": "exact",
                "match": null
              }
            ],
            "result": [
              {
                "id": "zone1",
                "name": "example1.com"
              },
              {
                "id": "zone2",
                "name": "example2.com"
              }
            ]
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`,
		},
		{
			Name: "zones state with nulls",
			Input: `{
  "version": 4,
  "terraform_version": "1.0.0",
  "resources": [
    {
      "type": "cloudflare_zones",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "checksum456",
            "filter": [
              {
                "account_id": null,
                "name": "test.com",
                "status": null,
                "paused": false,
                "lookup_type": "exact",
                "match": null
              }
            ],
            "zones": [
              {
                "id": "zone3",
                "name": "test.com"
              }
            ]
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.0.0",
  "resources": [
    {
      "type": "cloudflare_zones",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "checksum456",
            "filter": [
              {
                "account_id": null,
                "name": "test.com",
                "status": null,
                "paused": false,
                "lookup_type": "exact",
                "match": null
              }
            ],
            "result": [
              {
                "id": "zone3",
                "name": "test.com"
              }
            ]
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`,
		},
		{
			Name: "empty zones list",
			Input: `{
  "version": 4,
  "terraform_version": "1.0.0",
  "resources": [
    {
      "type": "cloudflare_zones",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "checksum-empty",
            "filter": [
              {
                "name": "nonexistent.com"
              }
            ],
            "zones": []
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`,
			Expected: `{
  "version": 4,
  "terraform_version": "1.0.0",
  "resources": [
    {
      "type": "cloudflare_zones",
      "name": "example",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "checksum-empty",
            "filter": [
              {
                "name": "nonexistent.com"
              }
            ],
            "result": []
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
