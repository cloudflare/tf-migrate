package snippet

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of snippet resources from v4 to v5
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates and registers a new snippet migrator
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (both v4 and v5 use the same name)
	internal.RegisterMigrator("cloudflare_snippet", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type
func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name unchanged in v5
	return "cloudflare_snippet"
}

// CanHandle checks if this migrator can handle the given resource type
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_snippet"
}

// Preprocess performs string-level transformations before HCL parsing
// Not needed for snippet migration - ConvertBlocksToArrayAttribute handles files conversion
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// Postprocess performs string-level transformations after HCL generation
func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// GetResourceRename returns the old and new resource names if renamed
// Snippet resource is NOT renamed (same name in v4 and v5)
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_snippet", "cloudflare_snippet"
}

// TransformConfig transforms the HCL configuration from v4 to v5
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Step 1: Rename name → snippet_name
	tfhcl.RenameAttribute(body, "name", "snippet_name")

	// Step 2: Convert files blocks to array attribute
	// v4: files { name = "..." content = "..." }
	// v5: files = [{ name = "..." content = "..." }]
	tfhcl.ConvertBlocksToArrayAttribute(body, "files", false)

	// Step 3: Create metadata wrapper for main_module
	// v4: main_module = "main.js"
	// v5: metadata = { main_module = "main.js" }
	tfhcl.MoveAttributesToNestedObject(body, "metadata", []string{"main_module"})

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op for snippet migration.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string, resourceName string) (string, error) {
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
