package access_rule

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_access_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_access_rule"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_access_rule"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_rule", "cloudflare_access_rule"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For access_rule, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert configuration block to attribute (MaxItems:1 → SingleNestedAttribute)
	// v4: configuration { target = "ip" value = "1.2.3.4" }
	// v5: configuration = { target = "ip" value = "1.2.3.4" }
	tfhcl.ConvertBlocksToAttribute(body, "configuration", "configuration", func(block *hclwrite.Block) {})

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the JSON state from v4 to v5.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Get attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Even for invalid instances, set schema_version
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Ensure mutual exclusivity of account_id and zone_id
	// v5 requires that only one is set, the other must be null
	accountID := attrs.Get("account_id")
	zoneID := attrs.Get("zone_id")

	if accountID.Exists() && accountID.String() != "" {
		// This is an account-level rule, ensure zone_id is null
		result, _ = sjson.Delete(result, "attributes.zone_id")
	} else if zoneID.Exists() && zoneID.String() != "" {
		// This is a zone-level rule, ensure account_id is null
		result, _ = sjson.Delete(result, "attributes.account_id")
	}

	// Convert configuration array to object (MaxItems:1 → SingleNestedAttribute)
	// v4 state: "configuration": [{"target": "ip", "value": "1.2.3.4"}]
	// v5 state: "configuration": {"target": "ip", "value": "1.2.3.4"}
	result = state.ConvertMaxItemsOneArrayToObject(result, "attributes", attrs, "configuration")

	// Set schema_version to 0 for v5 (ALWAYS required!)
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
