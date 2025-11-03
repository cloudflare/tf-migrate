package list

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "IP list state migration",
				Input: `{
  "attributes": {
    "account_id": "abc123",
    "name": "ip_list",
    "kind": "ip",
    "item": [
      {
        "comment": "First IP",
        "value": [{
          "ip": "1.1.1.1"
        }]
      },
      {
        "comment": "Second IP",
        "value": [{
          "ip": "1.1.1.2"
        }]
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "account_id": "abc123",
    "name": "ip_list",
    "kind": "ip",
    "num_items": 2,
    "items": [
      {
        "comment": "First IP",
        "ip": "1.1.1.1"
      },
      {
        "comment": "Second IP",
        "ip": "1.1.1.2"
      }
    ]
  }
}`,
			},
			{
				Name: "ASN list state migration",
				Input: `{
  "attributes": {
    "kind": "asn",
    "item": [
      {
        "comment": "Google ASN",
        "value": [{
          "asn": 15169
        }]
      },
      {
        "value": [{
          "asn": 13335
        }]
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "kind": "asn",
    "num_items": 2,
    "items": [
      {
        "comment": "Google ASN",
        "asn": 15169
      },
      {
        "asn": 13335
      }
    ]
  }
}`,
			},
			{
				Name: "Hostname list state migration",
				Input: `{
  "attributes": {
    "kind": "hostname",
    "item": [
      {
        "comment": "Example hostname",
        "value": [{
          "hostname": [{
            "url_hostname": "example.com"
          }]
        }]
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "kind": "hostname",
    "num_items": 1,
    "items": [
      {
        "comment": "Example hostname",
        "hostname": {
          "url_hostname": "example.com"
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Redirect list state migration with boolean conversions",
				Input: `{
  "attributes": {
    "kind": "redirect",
    "item": [
      {
        "comment": "Main redirect",
        "value": [{
          "redirect": [{
            "source_url": "example.com/old",
            "target_url": "example.com/new",
            "include_subdomains": "enabled",
            "subpath_matching": "disabled",
            "preserve_query_string": "enabled",
            "preserve_path_suffix": "disabled",
            "status_code": 301
          }]
        }]
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "kind": "redirect",
    "num_items": 1,
    "items": [
      {
        "comment": "Main redirect",
        "redirect": {
          "source_url": "example.com/old",
          "target_url": "example.com/new",
          "include_subdomains": true,
          "subpath_matching": false,
          "preserve_query_string": true,
          "preserve_path_suffix": false,
          "status_code": 301
        }
      }
    ]
  }
}`,
			},
			{
				Name: "Empty item array removal",
				Input: `{
  "attributes": {
    "kind": "ip",
    "item": []
  }
}`,
				Expected: `{
  "attributes": {
    "kind": "ip",
    "num_items": 0
  }
}`,
			},
			{
				Name: "IP without comment",
				Input: `{
  "attributes": {
    "kind": "ip",
    "item": [
      {
        "value": [{
          "ip": "10.0.0.1"
        }]
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "kind": "ip",
    "num_items": 1,
    "items": [
      {
        "ip": "10.0.0.1"
      }
    ]
  }
}`,
			},
			{
				Name: "Multiple hostname items",
				Input: `{
  "attributes": {
    "kind": "hostname",
    "item": [
      {
        "comment": "First host",
        "value": [{
          "hostname": [{
            "url_hostname": "example1.com"
          }]
        }]
      },
      {
        "comment": "Second host",
        "value": [{
          "hostname": [{
            "url_hostname": "example2.com"
          }]
        }]
      }
    ]
  }
}`,
				Expected: `{
  "attributes": {
    "kind": "hostname",
    "num_items": 2,
    "items": [
      {
        "comment": "First host",
        "hostname": {
          "url_hostname": "example1.com"
        }
      },
      {
        "comment": "Second host",
        "hostname": {
          "url_hostname": "example2.com"
        }
      }
    ]
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
