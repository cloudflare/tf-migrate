package queue

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
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
	result := stateJSON.String()

	// Validate instance exists and has attributes
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		// Even for invalid instances, set schema_version to 0 for v5
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// Transformation 1: rename name → queue_name
	result = state.RenameField(result, "attributes", attrs, "name", "queue_name")

	// Transformation 2: Copy id to queue_id
	// v5 requires both id and queue_id fields (they should be equal)
	// v4 only had id, so we copy it to queue_id for v5
	if idValue := attrs.Get("id"); idValue.Exists() {
		result, _ = sjson.Set(result, "attributes.queue_id", idValue.String())
	}

	// MANDATORY: Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
