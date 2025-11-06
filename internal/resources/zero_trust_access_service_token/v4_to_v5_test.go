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

	// Test state transformations
	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Type conversion - client_secret_version from int to float64",
				Input: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "my_token",
        "client_id": "client789",
        "client_secret": "secret_abc",
        "expires_at": "2025-12-31T23:59:59Z",
        "duration": "8760h",
        "client_secret_version": 1,
        "min_days_for_renewal": 30
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "terraform_version": "1.5.0",
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "my_token",
        "client_id": "client789",
        "client_secret": "secret_abc",
        "expires_at": "2025-12-31T23:59:59Z",
        "duration": "8760h",
        "client_secret_version": 1.0
      }
    }]
  }]
}`,
			},
			{
				Name: "Type conversion - larger client_secret_version values",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "name": "my_token",
        "account_id": "account456",
        "client_secret_version": 5
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "name": "my_token",
        "account_id": "account456",
        "client_secret_version": 5.0
      }
    }]
  }]
}`,
			},
			{
				Name: "Missing client_secret_version - should add default",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "name": "test_token",
        "account_id": "account456"
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "name": "test_token",
        "account_id": "account456",
		"client_secret_version": 1.0
      }
    }]
  }]
}`,
			},
			{
				Name: "Field removal - min_days_for_renewal removed from state",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "name": "test_token",
        "account_id": "account456",
        "min_days_for_renewal": 30,
        "client_secret_version": 1.0
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "test",
    "instances": [{
      "attributes": {
        "id": "token123",
        "name": "test_token",
        "account_id": "account456",
        "client_secret_version": 1.0
      }
    }]
  }]
}`,
			},
			{
				Name: "Complete state with all fields",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "complete",
    "instances": [{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "complete_token",
        "client_id": "client789",
        "client_secret": "secret_abc",
        "expires_at": "2025-12-31T23:59:59Z",
        "duration": "8760h",
        "client_secret_version": 2,
        "previous_client_secret_expires_at": "2024-12-31T23:59:59Z",
        "min_days_for_renewal": 30
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "complete",
    "instances": [{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "complete_token",
        "client_id": "client789",
        "client_secret": "secret_abc",
        "expires_at": "2025-12-31T23:59:59Z",
        "duration": "8760h",
        "client_secret_version": 2.0,
        "previous_client_secret_expires_at": "2024-12-31T23:59:59Z"
      }
    }]
  }]
}`,
			},
			{
				Name: "Zone-scoped resource (zone_id instead of account_id)",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "zone_token",
    "instances": [{
      "attributes": {
        "id": "token123",
        "zone_id": "zone456",
        "name": "zone_token",
        "client_secret_version": 1,
        "min_days_for_renewal": 15
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "zone_token",
    "instances": [{
      "attributes": {
        "id": "token123",
        "zone_id": "zone456",
        "name": "zone_token",
        "client_secret_version": 1.0
      }
    }]
  }]
}`,
			},
			{
				Name: "Legacy resource name - cloudflare_access_service_token in state",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_access_service_token",
    "name": "legacy",
    "instances": [{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "legacy_token",
        "client_secret_version": 1,
        "min_days_for_renewal": 30
      }
    }]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "legacy",
    "instances": [{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "legacy_token",
        "client_secret_version": 1.0
      }
    }]
  }]
}`,
			},
			{
				Name: "Multiple instances of the same resource",
				Input: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "multi",
    "instances": [
      {
        "attributes": {
          "id": "token1",
          "account_id": "account456",
          "name": "first_token",
          "client_secret_version": 1,
          "min_days_for_renewal": 30
        }
      },
      {
        "attributes": {
          "id": "token2",
          "account_id": "account456",
          "name": "second_token",
          "client_secret_version": 2,
          "min_days_for_renewal": 60
        }
      }
    ]
  }]
}`,
				Expected: `{
  "version": 4,
  "resources": [{
    "type": "cloudflare_zero_trust_access_service_token",
    "name": "multi",
    "instances": [
      {
        "attributes": {
          "id": "token1",
          "account_id": "account456",
          "name": "first_token",
          "client_secret_version": 1.0
        }
      },
      {
        "attributes": {
          "id": "token2",
          "account_id": "account456",
          "name": "second_token",
          "client_secret_version": 2.0
        }
      }
    ]
  }]
}`,
			},
			{
				Name: "State with empty resources",
				Input: `{
					"resources": []
				}`,
				Expected: `{
					"resources": []
				}`,
			},
			{
				Name: "State without instances",
				Input: `{
					"resources": [{
						"type": "cloudflare_zero_trust_access_service_token",
						"name": "empty",
						"instances": []
					}]
				}`,
				Expected: `{
					"resources": [{
						"type": "cloudflare_zero_trust_access_service_token",
						"name": "empty",
						"instances": []
					}]
				}`,
			},
			{
				Name: "Type conversion - single instance",
				Input: `{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "my_token",
        "client_id": "client789",
        "client_secret": "secret_abc",
        "expires_at": "2025-12-31T23:59:59Z",
        "duration": "8760h",
        "client_secret_version": 1,
        "min_days_for_renewal": 30
      }
}`,
				Expected: `{
      "attributes": {
        "id": "token123",
        "account_id": "account456",
        "name": "my_token",
        "client_id": "client789",
        "client_secret": "secret_abc",
        "expires_at": "2025-12-31T23:59:59Z",
        "duration": "8760h",
        "client_secret_version": 1.0
      }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
