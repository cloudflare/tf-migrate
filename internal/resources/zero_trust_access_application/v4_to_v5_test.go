package zero_trust_access_application

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation_Basic(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic saas application with saas_app block",
			Input: `
resource "cloudflare_access_application" "test" {
  account_id = "account-123"
  name       = "Test App"
  type       = "saas"

  saas_app {
    consumer_service_url = "https://example.com/saml/consume"
    name_id_format       = "email"
    sp_entity_id         = "example-sp-entity"
  }
}`,
			Expected: `
resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "account-123"
  name       = "Test App"
  type       = "saas"

  saas_app = {
    consumer_service_url = "https://example.com/saml/consume"
    name_id_format       = "email"
    sp_entity_id         = "example-sp-entity"
  }
}`,
		},
		{
			Name: "application with domain_type removed",
			Input: `
resource "cloudflare_access_application" "test" {
  account_id  = "account-123"
  name        = "Test App"
  type        = "self_hosted"
  domain      = "test.example.com"
  domain_type = "full"
}`,
			Expected: `
resource "cloudflare_zero_trust_access_application" "test" {
  account_id = "account-123"
  name       = "Test App"
  type       = "self_hosted"
  domain     = "test.example.com"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestStateTransformation_Basic(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.StateTestCase{
		{
			Name: "saas_app array converted to single object",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "app-123",
    "account_id": "account-123",
    "name": "Test App",
    "type": "saas",
    "saas_app": [{
      "consumer_service_url": "https://example.com/saml/consume",
      "name_id_format": "email",
      "sp_entity_id": "example-sp-entity"
    }]
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "app-123",
    "account_id": "account-123",
    "name": "Test App",
    "type": "saas",
    "saas_app": {
      "consumer_service_url": "https://example.com/saml/consume",
      "name_id_format": "email",
      "sp_entity_id": "example-sp-entity"
    }
  }
}`,
		},
		{
			Name: "domain_type removed from state",
			Input: `{
  "schema_version": 0,
  "attributes": {
    "id": "app-123",
    "account_id": "account-123",
    "name": "Test App",
    "type": "self_hosted",
    "domain": "test.example.com",
    "domain_type": "full"
  }
}`,
			Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "app-123",
    "account_id": "account-123",
    "name": "Test App",
    "type": "self_hosted",
    "domain": "test.example.com"
  }
}`,
		},
	}

	testhelpers.RunStateTransformTests(t, tests, migrator)
}
