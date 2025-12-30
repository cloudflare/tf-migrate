package notification_policy_webhooks

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
	internal.RegisterMigrator("cloudflare_notification_policy_webhooks", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_notification_policy_webhooks"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_notification_policy_webhooks"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_notification_policy_webhooks", "cloudflare_notification_policy_webhooks"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - all fields remain the same
	// Note: We assume all v4 configs have the url field since it was optional in v4
	// but is required in v5. If url is missing, the v5 provider will catch it during validation.
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() {
		return result, nil
	}

	// Set schema_version to 0 for v5
	// Note: We assume all v4 states have the url field. While it was optional in v4,
	// the Cloudflare API requires it, so all real resources should have it.
	// If url is missing, the v5 provider will catch it during the next apply.
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
