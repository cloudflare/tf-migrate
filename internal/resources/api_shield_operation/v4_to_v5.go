package api_shield_operation

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

const (
	v4ResourceType = "cloudflare_api_shield_operation"
	v5ResourceType = "cloudflare_api_shield_operation"
)

// V4ToV5Migrator handles the migration of cloudflare_api_shield_operation from v4 to v5.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_api_shield_operation", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return v5ResourceType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == v4ResourceType || resourceType == v5ResourceType
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return v4ResourceType, v5ResourceType
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformation needed - all user-provided fields are identical between v4 and v5.
	// New v5 fields (operation_id, last_updated, features) are all computed and don't appear in config.
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

	// Set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	// Copy id to operation_id for v5 provider
	// V5 requires operation_id to make API calls during plan/refresh operations
	operationID := stateJSON.Get("attributes.id").String()
	if operationID != "" {
		result, _ = sjson.Set(result, "attributes.operation_id", operationID)
	}

	return result, nil
}
