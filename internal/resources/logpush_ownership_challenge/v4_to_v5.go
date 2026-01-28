package logpush_ownership_challenge

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name - same as v5 in this case
	internal.RegisterMigrator("cloudflare_logpush_ownership_challenge", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_logpush_ownership_challenge"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_logpush_ownership_challenge"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - config stays identical
	return content
}

// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_logpush_ownership_challenge", "cloudflare_logpush_ownership_challenge"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// NO TRANSFORMATIONS NEEDED
	// The HCL configuration is identical in v4 and v5:
	// - Same resource name
	// - Same field names
	// - Same field types
	// - Same validation behavior

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		// Set schema_version even for invalid/incomplete instances
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// ONLY ACTION: Remove v4 computed field that doesn't exist in v5
	// ownership_challenge_filename was a computed field in v4
	// In v5, it's replaced by separate computed fields: filename, message, valid
	// We remove the old field and let v5 provider regenerate the new computed fields
	result = state.RemoveFields(result, "attributes", attrs,
		"ownership_challenge_filename",
	)

	// CRITICAL: Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
