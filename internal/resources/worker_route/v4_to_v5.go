package worker_route

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
	// Register with both v4 resource names (plural and singular forms)
	internal.RegisterMigrator("cloudflare_workers_route", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_worker_route", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (plural form)
	return "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both v4 resource names (plural and singular)
	return resourceType == "cloudflare_workers_route" || resourceType == "cloudflare_worker_route"
}

// GetResourceRename implements the ResourceRenamer interface
// Handles both cloudflare_worker_route (singular) and cloudflare_workers_route (plural)
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_worker_route", "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for this simple migration
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Handle resource rename: cloudflare_worker_route → cloudflare_workers_route
	tfhcl.RenameResourceType(block, "cloudflare_worker_route", "cloudflare_workers_route")

	// Rename field: script_name → script
	tfhcl.RenameAttribute(body, "script_name", "script")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Rename field in state: script_name → script
	result = state.RenameField(result, "attributes", attrs, "script_name", "script")

	// Always set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
