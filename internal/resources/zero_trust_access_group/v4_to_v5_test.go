package zero_trust_access_group

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
    "name": "test-group",
    "include": [{
      "email": ["user1@example.com", "user2@example.com"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"email": {"email": "user1@example.com"}},
      {"email": {"email": "user2@example.com"}}
    ]
  }
}`,
			},
			{
				Name: "transforms email_domain arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "email_domain": ["example.com", "test.com"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"email_domain": {"domain": "example.com"}},
      {"email_domain": {"domain": "test.com"}}
    ]
  }
}`,
			},
			{
				Name: "transforms ip arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "ip": ["192.0.2.1/32", "192.0.2.2/32"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"ip": {"ip": "192.0.2.1/32"}},
      {"ip": {"ip": "192.0.2.2/32"}}
    ]
  }
}`,
			},
			{
				Name: "transforms geo arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "geo": ["US", "CA"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"geo": {"country_code": "US"}},
      {"geo": {"country_code": "CA"}}
    ]
  }
}`,
			},
			{
				Name: "transforms boolean fields to empty objects",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "everyone": true,
      "certificate": true,
      "any_valid_service_token": true
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"everyone": {}},
      {"certificate": {}},
      {"any_valid_service_token": {}}
    ]
  }
}`,
			},
			{
				Name: "handles exclude rules",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "exclude": [{
      "email": ["blocked@example.com"],
      "ip": ["192.168.1.1/32"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "exclude": [
      {"email": {"email": "blocked@example.com"}},
      {"ip": {"ip": "192.168.1.1/32"}}
    ]
  }
}`,
			},
			{
				Name: "handles require rules",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "require": [{
      "email_domain": ["company.com"],
      "certificate": true
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "require": [
      {"email_domain": {"domain": "company.com"}},
      {"certificate": {}}
    ]
  }
}`,
			},
			{
				Name: "handles v5 format unchanged",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"email": {"email": "user@example.com"}},
      {"everyone": {}}
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"email": {"email": "user@example.com"}},
      {"everyone": {}}
    ]
  }
}`,
			},
			{
				Name: "handles mixed rule types",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "email": ["user1@example.com", "user2@example.com"],
      "ip": ["10.0.0.1/32"],
      "everyone": true
    }],
    "exclude": [{
      "geo": ["CN", "RU"]
    }],
    "require": [{
      "email_domain": ["company.com"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"email": {"email": "user1@example.com"}},
      {"email": {"email": "user2@example.com"}},
      {"ip": {"ip": "10.0.0.1/32"}},
      {"everyone": {}}
    ],
    "exclude": [
      {"geo": {"country_code": "CN"}},
      {"geo": {"country_code": "RU"}}
    ],
    "require": [
      {"email_domain": {"domain": "company.com"}}
    ]
  }
}`,
			},
			{
				Name: "handles empty rules",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [],
    "exclude": [],
    "require": []
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [],
    "exclude": [],
    "require": []
  }
}`,
			},
			{
				Name: "transforms group arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "group": ["group1", "group2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"group": {"id": "group1"}},
      {"group": {"id": "group2"}}
    ]
  }
}`,
			},
			{
				Name: "transforms service_token arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "service_token": ["token1", "token2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"service_token": {"token_id": "token1"}},
      {"service_token": {"token_id": "token2"}}
    ]
  }
}`,
			},
			{
				Name: "transforms email_list arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "email_list": ["list1", "list2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"email_list": {"id": "list1"}},
      {"email_list": {"id": "list2"}}
    ]
  }
}`,
			},
			{
				Name: "transforms ip_list arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "ip_list": ["iplist1", "iplist2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"ip_list": {"id": "iplist1"}},
      {"ip_list": {"id": "iplist2"}}
    ]
  }
}`,
			},
			{
				Name: "transforms login_method arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "login_method": ["method1", "method2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"login_method": {"id": "method1"}},
      {"login_method": {"id": "method2"}}
    ]
  }
}`,
			},
			{
				Name: "transforms device_posture arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "device_posture": ["posture1", "posture2"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"device_posture": {"integration_uid": "posture1"}},
      {"device_posture": {"integration_uid": "posture2"}}
    ]
  }
}`,
			},
			{
				Name: "transforms common_names to common_name",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [{
      "common_names": ["cert1.example.com", "cert2.example.com"]
    }]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-group",
    "include": [
      {"common_name": {"common_name": "cert1.example.com"}},
      {"common_name": {"common_name": "cert2.example.com"}}
    ]
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
