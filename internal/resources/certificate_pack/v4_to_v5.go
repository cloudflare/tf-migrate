package certificate_pack

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of certificate_pack resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_certificate_pack", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "", ""
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// wait_for_active_status: removed in v5
	tfhcl.RemoveAttributes(body, "wait_for_active_status")

	// validation_records, validation_errors: were Optional+Computed in v4, only Computed in v5
	tfhcl.RemoveAttributes(body, "validation_records", "validation_errors")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// The provider's migration logic handles all state transformations including:
	// - Removing wait_for_active_status field
	// - Converting types.Set to customfield.Set
	// - Transforming validation_records to remove cname fields
	// This function is a no-op for certificate_pack migration
	return stateJSON.String(), nil
}
