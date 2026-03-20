package account_member

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_account_member", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_account_member"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_account_member"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_account_member"}, "cloudflare_account_member"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	tfhcl.RenameAttribute(body, "email_address", "email")
	tfhcl.RenameAttribute(body, "role_ids", "roles")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState returns the state unchanged (no-op).
//
// State migration is now handled by the provider's StateUpgrader (v5.19+).
// The provider implements UpgradeState with slot 0 handling v4 SDKv2 state
// (schema_version=0) and transforming email_address→email, role_ids→roles.
//
// tf-migrate only needs to transform the HCL configuration; Terraform will
// invoke the provider's state upgrader when it detects schema_version mismatch.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// No-op: Provider's StateUpgrader handles v4→v5 state transformation
	return stateJSON.String(), nil
}
