package access_policy

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "transforms email arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "email": ["user1@example.com", "user2@example.com"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"email": {"email": "user1@example.com"}},
      {"email": {"email": "user2@example.com"}}
    ]
  }
}`,
			},
			{
				Name: "transforms boolean fields to empty objects",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "everyone": true,
      "certificate": true
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"everyone": {}},
      {"certificate": {}}
    ]
  }
}`,
			},
			{
				Name: "handles exclude and require rules",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "everyone": true
    }],
    "exclude": [{
      "email": ["blocked@example.com"]
    }],
    "require": [{
      "email_domain": ["company.com"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"everyone": {}}
    ],
    "exclude": [
      {"email": {"email": "blocked@example.com"}}
    ],
    "require": [
      {"email_domain": {"domain": "company.com"}}
    ]
  }
}`,
			},
			{
				Name: "handles v5 format unchanged",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"email": {"email": "user@example.com"}}
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"email": {"email": "user@example.com"}}
    ]
  }
}`,
			},
			{
				Name: "handles empty rules",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [],
    "exclude": [],
    "require": []
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [],
    "exclude": [],
    "require": []
  }
}`,
			},
			{
				Name: "transforms geo arrays with country_code",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "geo": ["US", "CA"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"geo": {"country_code": "US"}},
      {"geo": {"country_code": "CA"}}
    ]
  }
}`,
			},
			{
				Name: "transforms ip arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "ip": ["192.0.2.1/32", "192.0.2.2/32"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"ip": {"ip": "192.0.2.1/32"}},
      {"ip": {"ip": "192.0.2.2/32"}}
    ]
  }
}`,
			},
			{
				Name: "transforms group arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "group": ["group1", "group2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"group": {"id": "group1"}},
      {"group": {"id": "group2"}}
    ]
  }
}`,
			},
			{
				Name: "handles mixed rule types",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [{
      "email": ["user1@example.com"],
      "ip": ["10.0.0.1/32"],
      "everyone": true
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-policy",
    "include": [
      {"email": {"email": "user1@example.com"}},
      {"ip": {"ip": "10.0.0.1/32"}},
      {"everyone": {}}
    ]
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
