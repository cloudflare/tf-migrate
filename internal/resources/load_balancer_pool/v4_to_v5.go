package load_balancer_pool

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of load balancer pool resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_load_balancer_pool", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (unchanged from v4)
	return "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_load_balancer_pool doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_load_balancer_pool", "cloudflare_load_balancer_pool"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Transform origins blocks to origins attribute array
	// v4: origins { name = "origin1" address = "1.2.3.4" }
	// v5: origins = [{ name = "origin1" address = "1.2.3.4" }]
	tfhcl.ConvertBlocksToArrayAttribute(body, "origins", false)

	// Transform load_shedding block to attribute (MaxItems:1)
	// v4: load_shedding { ... }
	// v5: load_shedding = { ... }
	tfhcl.ConvertSingleBlockToAttribute(body, "load_shedding", "load_shedding")

	// Transform origin_steering block to attribute (MaxItems:1)
	// v4: origin_steering { ... }
	// v5: origin_steering = { ... }
	tfhcl.ConvertSingleBlockToAttribute(body, "origin_steering", "origin_steering")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	// Transform load_shedding from array to object (or null if empty)
	// v4: "load_shedding": [{ ... }] or []
	// v5: "load_shedding": { ... } or null
	result = state.TransformFieldArrayToObject(result, "attributes", attrs, "load_shedding", state.ArrayToObjectOptions{
		TransformEmptyToNull: true,
	})

	// Transform origin_steering from array to object (or null if empty)
	// v4: "origin_steering": [{ ... }] or []
	// v5: "origin_steering": { ... } or null
	result = state.TransformFieldArrayToObject(result, "attributes", attrs, "origin_steering", state.ArrayToObjectOptions{
		TransformEmptyToNull: true,
	})

	// Transform header field inside each origin from array to object
	// v4: origins[*].header = [] or [{ ... }]
	// v5: origins[*].header = {} or { ... } (provider expects object, not array)
	// Re-parse to get updated state after previous transformations
	updatedState := gjson.Parse(result)
	origins := updatedState.Get("attributes.origins")
	if origins.Exists() && origins.IsArray() {
		originsArray := origins.Array()
		for i, origin := range originsArray {
			// Only transform header if it exists
			if origin.Get("header").Exists() {
				// Use EnsureObjectExists so empty arrays become {} instead of null
				result = state.TransformFieldArrayToObject(result, fmt.Sprintf("attributes.origins.%d", i), origin, "header", state.ArrayToObjectOptions{
					EnsureObjectExists: true,
				})
			}
		}
	}

	return result, nil
}
