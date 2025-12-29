package load_balancer_pool

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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

	// Check if there are any dynamic blocks for origins
	// In v5, origins must be an attribute, not a block
	// dynamic "origins" blocks need to be converted to for expressions
	dynamicOrigins := tfhcl.FindDynamicBlock(body, "origins")
	if dynamicOrigins != nil {
		// Convert dynamic "origins" block to for expression
		// v4: dynamic "origins" { for_each = X content { ... } }
		// v5: origins = [for item in X : { ... }]
		if err := tfhcl.ConvertDynamicBlockToForExpression(body, dynamicOrigins, "origins"); err != nil {
			return nil, fmt.Errorf("converting dynamic origins block: %w", err)
		}
	} else {
		// Convert static origins blocks to array attribute
		// v4: origins { name = "origin1" address = "1.2.3.4" }
		// v5: origins = [{ name = "origin1" address = "1.2.3.4" }]
		tfhcl.ConvertBlocksToArrayAttribute(body, "origins", false)
	}

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

	// Transform load_shedding from array to object (or null if empty)
	// v4: "load_shedding": [{ ... }] or []
	// v5: "load_shedding": { ... } or null
	loadShedding := stateJSON.Get("attributes.load_shedding")
	if loadShedding.Exists() && loadShedding.IsArray() {
		if len(loadShedding.Array()) > 0 {
			firstElement := loadShedding.Array()[0]
			result, _ = sjson.Set(result, "attributes.load_shedding", firstElement.Value())
		} else {
			// Empty array -> null
			result, _ = sjson.Set(result, "attributes.load_shedding", nil)
		}
	}

	// Transform origin_steering from array to object (or null if empty)
	// v4: "origin_steering": [{ ... }] or []
	// v5: "origin_steering": { ... } or null
	originSteering := stateJSON.Get("attributes.origin_steering")
	if originSteering.Exists() && originSteering.IsArray() {
		if len(originSteering.Array()) > 0 {
			firstElement := originSteering.Array()[0]
			result, _ = sjson.Set(result, "attributes.origin_steering", firstElement.Value())
		} else {
			// Empty array -> null
			result, _ = sjson.Set(result, "attributes.origin_steering", nil)
		}
	}

	// Transform header field inside each origin from array to object/null
	// v4: origins[*].header = [] or [{ ... }]
	// v5: origins[*].header = {} or null (provider expects object, not array)
	// Re-parse to get updated state after previous transformations
	updatedState := gjson.Parse(result)
	origins := updatedState.Get("attributes.origins")
	if origins.Exists() && origins.IsArray() {
		originsArray := origins.Array()
		for i, origin := range originsArray {
			header := origin.Get("header")
			if header.Exists() && header.IsArray() {
				if len(header.Array()) == 0 {
					// Empty array -> empty object (v5 provider expects object type)
					result, _ = sjson.Set(result, fmt.Sprintf("attributes.origins.%d.header", i), map[string]interface{}{})
				} else {
					// Non-empty array -> convert first element to object
					// This handles the case where v4 had header as array of objects
					firstElement := header.Array()[0]
					result, _ = sjson.Set(result, fmt.Sprintf("attributes.origins.%d.header", i), firstElement.Value())
				}
			}
		}
	}

	return result, nil
}
