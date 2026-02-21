package queue

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Queue resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the v4 resource name (same as v5 - no rename)
	internal.RegisterMigrator("cloudflare_queue", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the v5 resource name (same as v4)
	return "cloudflare_queue"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the v4 resource name
	return resourceType == "cloudflare_queue"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for queue migration
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	// No postprocessing needed
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No resource type rename needed (cloudflare_queue stays the same)
	body := block.Body()

	// Only transformation: rename name → queue_name
	tfhcl.RenameAttribute(body, "name", "queue_name")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState).
	// This function is a no-op for queue migration.
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
