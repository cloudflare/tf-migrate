package zero_trust_organization

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	// This is called for the registered name, but both v4 names go to the same v5 name
	return []string{"cloudflare_access_organization", "cloudflare_zero_trust_access_organization"}, "cloudflare_zero_trust_organization"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For zero_trust_organization, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Get the resource name and type before renaming (for moved block generation)
	resourceName := tfhcl.GetResourceName(block)
	currentType := tfhcl.GetResourceType(block)

	// Rename resource type from EITHER v4 name to v5 name
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

	// Generate moved block for state migration
	// Both v4 resource names get moved to the same v5 name
	from := currentType + "." + resourceName
	to := "cloudflare_zero_trust_organization." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

