package workers_kv

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic resource",
				Input: `
resource "cloudflare_workers_kv" "example" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "my_key"
  value        = "my_value"
}`,
				Expected: `
resource "cloudflare_workers_kv" "example" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = "my_value"
  key_name     = "my_key"
}`,
			},
			{
				Name: "multiple resources",
				Input: `
resource "cloudflare_workers_kv" "first" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "key1"
  value        = "value1"
}

resource "cloudflare_workers_kv" "second" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "key2"
  value        = "value2"
}`,
				Expected: `
resource "cloudflare_workers_kv" "first" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = "value1"
  key_name     = "key1"
}

resource "cloudflare_workers_kv" "second" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = "value2"
  key_name     = "key2"
}`,
			},
			{
				Name: "with special characters in key",
				Input: `
resource "cloudflare_workers_kv" "encoded" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  key          = "my%20encoded%20key"
  value        = "{\"test\": \"json\"}"
}`,
				Expected: `
resource "cloudflare_workers_kv" "encoded" {
  account_id   = "d41d8cd98f00b204e9800998ecf8427e"
  namespace_id = "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3"
  value        = "{\"test\": \"json\"}"
  key_name     = "my%20encoded%20key"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state",
				Input: `{
					"attributes": {
						"id": "test-id",
						"account_id": "d41d8cd98f00b204e9800998ecf8427e",
						"namespace_id": "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3",
						"key": "my_key",
						"value": "my_value"
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id",
						"account_id": "d41d8cd98f00b204e9800998ecf8427e",
						"namespace_id": "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3",
						"key_name": "my_key",
						"value": "my_value"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "with special characters",
				Input: `{
					"attributes": {
						"id": "test-id",
						"account_id": "d41d8cd98f00b204e9800998ecf8427e",
						"namespace_id": "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3",
						"key": "my%20encoded%20key",
						"value": "{\"test\": \"json\"}"
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id",
						"account_id": "d41d8cd98f00b204e9800998ecf8427e",
						"namespace_id": "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3",
						"key_name": "my%20encoded%20key",
						"value": "{\"test\": \"json\"}"
					},
					"schema_version": 0
				}`,
			},
			{
				Name: "empty value",
				Input: `{
					"attributes": {
						"id": "test-id",
						"account_id": "d41d8cd98f00b204e9800998ecf8427e",
						"namespace_id": "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3",
						"key": "empty_key",
						"value": ""
					}
				}`,
				Expected: `{
					"attributes": {
						"id": "test-id",
						"account_id": "d41d8cd98f00b204e9800998ecf8427e",
						"namespace_id": "f451d2e8c4a1b3d7e6f9c8a7b6d5e4f3",
						"key_name": "empty_key",
						"value": ""
					},
					"schema_version": 0
				}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
