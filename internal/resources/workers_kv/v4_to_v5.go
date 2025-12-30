package workers_kv

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_workers_kv", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_kv"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_kv"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for this simple migration
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_workers_kv", "cloudflare_workers_kv"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Rename field: key → key_name
	tfhcl.RenameAttribute(body, "key", "key_name")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// Rename field: key → key_name
	result = state.RenameField(result, "attributes", attrs, "key", "key_name")

	// Set schema_version to 0
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
