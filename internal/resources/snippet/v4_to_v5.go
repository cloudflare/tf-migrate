package snippet

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
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

// TransformState transforms the Terraform state from v4 to v5
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Step 1: Rename name → snippet_name
	result = state.RenameField(result, "attributes", attrs, "name", "snippet_name")

	// Step 2: Move main_module → metadata.main_module
	if mainModule := attrs.Get("main_module"); mainModule.Exists() {
		mainModuleVal := mainModule.Value()

		// Create metadata object with main_module inside
		metadata := map[string]interface{}{
			"main_module": mainModuleVal,
		}

		// Set metadata in state
		result, _ = sjson.Set(result, "attributes.metadata", metadata)

		// Remove main_module from root
		result, _ = sjson.Delete(result, "attributes.main_module")
	}

	// Step 3: Ensure files array is preserved
	// In v4, files is stored as an array in state and remains as an array in v5
	// We need to explicitly ensure it exists to prevent it from being nil
	attrs = gjson.Get(result, "attributes")
	result = state.EnsureField(result, "attributes", attrs, "files", []interface{}{})

	// Step 4: Set schema_version = 0
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
