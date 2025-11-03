package load_balancer

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "renames fallback_pool_id to fallback_pool",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "fallback_pool_id": "pool-123"
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "fallback_pool": "pool-123"
  }
}`,
			},
			{
				Name: "renames default_pool_ids to default_pools",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "default_pool_ids": ["pool-1", "pool-2"]
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "default_pools": ["pool-1", "pool-2"]
  }
}`,
			},
			{
				Name: "removes empty single object attribute arrays",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "adaptive_routing": [],
    "location_strategy": [],
    "random_steering": [],
    "session_affinity_attributes": []
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb"
  }
}`,
			},
			{
				Name: "converts empty map attribute arrays to empty maps",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "country_pools": [],
    "pop_pools": [],
    "region_pools": []
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "country_pools": {},
    "pop_pools": {},
    "region_pools": {}
  }
}`,
			},
			{
				Name: "keeps rules as array",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "rules": []
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "rules": []
  }
}`,
			},
			{
				Name: "handles all transformations together",
				Input: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "fallback_pool_id": "pool-123",
    "default_pool_ids": ["pool-1", "pool-2"],
    "adaptive_routing": [],
    "location_strategy": [],
    "random_steering": [],
    "session_affinity_attributes": [],
    "country_pools": [],
    "pop_pools": [],
    "region_pools": [],
    "rules": []
  }
}`,
				Expected: `{
  "attributes": {
    "id": "test-id",
    "name": "test-lb",
    "fallback_pool": "pool-123",
    "default_pools": ["pool-1", "pool-2"],
    "country_pools": {},
    "pop_pools": {},
    "region_pools": {},
    "rules": []
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
