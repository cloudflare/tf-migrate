package r2_bucket

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_r2_bucket", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_r2_bucket"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_r2_bucket"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For r2_bucket, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For r2_bucket, the config is identical between v4 and v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - config is identical
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the JSON state from v4 to v5.
// For r2_bucket, we need to add the new v5 fields with their default values
// to prevent plan changes after migration.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	instance := stateJSON.Get("attributes")

	// Add new v5 fields with their default values to match the schema
	// jurisdiction: default is "default"
	result = state.EnsureField(result, "attributes", instance, "jurisdiction", "default")

	// storage_class: default is "Standard"
	result = state.EnsureField(result, "attributes", instance, "storage_class", "Standard")

	// Note: creation_date is not added here as it's purely computed and will be set by the API

	return result, nil
}
