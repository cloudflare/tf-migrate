package zero_trust_organization

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

// NewV4ToV5Migrator creates a new migrator instance and registers BOTH v4 resource names.
// v4 has two aliases: cloudflare_access_organization (deprecated) and
// cloudflare_zero_trust_access_organization (current). Both use the same schema
// and migrate to cloudflare_zero_trust_organization in v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (they're aliases with identical schemas)
	internal.RegisterMigrator("cloudflare_access_organization", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_organization", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the v5 resource name
	return "cloudflare_zero_trust_organization"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle BOTH v4 resource names
	return resourceType == "cloudflare_access_organization" ||
		resourceType == "cloudflare_zero_trust_access_organization"
}

// GetResourceRename implements the ResourceRenamer interface.
// Both v4 names rename to the same v5 name.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	// This is called for the registered name, but both v4 names go to the same v5 name
	return "cloudflare_access_organization", "cloudflare_zero_trust_organization"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For zero_trust_organization, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename resource type from EITHER v4 name to v5 name
	currentType := tfhcl.GetResourceType(block)
	if currentType == "cloudflare_access_organization" || currentType == "cloudflare_zero_trust_access_organization" {
		tfhcl.RenameResourceType(block, currentType, "cloudflare_zero_trust_organization")
	}

	body := block.Body()

	// Convert login_design block to attribute (MaxItems:1 → SingleNestedAttribute)
	// v4: login_design { background_color = "#000" ... }
	// v5: login_design = { background_color = "#000" ... }
	tfhcl.ConvertBlocksToAttribute(body, "login_design", "login_design", func(block *hclwrite.Block) {})

	// Convert custom_pages block to attribute (MaxItems:1 → SingleNestedAttribute)
	// v4: custom_pages { forbidden = "id" ... }
	// v5: custom_pages = { forbidden = "id" ... }
	tfhcl.ConvertBlocksToAttribute(body, "custom_pages", "custom_pages", func(block *hclwrite.Block) {})

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

	// Handle account_id / zone_id mutual exclusivity
	// v5 requires that only one is set, the other must be null
	accountID := attrs.Get("account_id")
	zoneID := attrs.Get("zone_id")

	if accountID.Exists() && accountID.String() != "" {
		// This is an account-level organization, ensure zone_id is null
		result, _ = sjson.Delete(result, "attributes.zone_id")
	} else if zoneID.Exists() && zoneID.String() != "" {
		// This is a zone-level organization, ensure account_id is null
		result, _ = sjson.Delete(result, "attributes.account_id")
	}

	// Convert MaxItems:1 arrays to objects
	// login_design: [{"background_color": "#000", ...}] → {"background_color": "#000", ...}
	// Handles empty arrays by deleting them
	result = state.ConvertMaxItemsOneArrayToObject(result, "attributes", attrs, "login_design")
	result = state.ConvertMaxItemsOneArrayToObject(result, "attributes", attrs, "custom_pages")

	// Add default boolean values if missing (v5 has defaults, v4 didn't)
	// This prevents PATCH operations when migrating resources that didn't set these
	result = state.EnsureField(result, "attributes", attrs, "allow_authenticate_via_warp", false)
	result = state.EnsureField(result, "attributes", attrs, "auto_redirect_to_identity", false)
	result = state.EnsureField(result, "attributes", attrs, "is_ui_read_only", false)

	// Remove the 'id' attribute - Framework manages ID separately from attributes
	// In SDKv2 (v4), ID was stored as an attribute. In Framework (v5), it's managed separately.
	result, _ = sjson.Delete(result, "attributes.id")

	// Set schema_version to 0 for v5 (ALWAYS required!)
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
