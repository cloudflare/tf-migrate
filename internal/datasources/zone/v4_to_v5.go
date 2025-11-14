package zone

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of zone datasource from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_zone", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - schemas are identical
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	// No postprocessing needed
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Zone datasource name is unchanged between v4 and v5
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	// No rename - return empty strings
	return "", ""
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No config transformation needed - schemas are identical
	// The datasource name stays as "cloudflare_zone" in both v4 and v5
	return nil, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	// Start with the instance JSON
	stateJSON := instance.Raw

	// The only change needed is to ensure schema_version is set to 0 for v5
	// All other fields are identical between v4 and v5
	stateJSON, _ = sjson.Set(stateJSON, "schema_version", 0)

	return stateJSON, nil
}
