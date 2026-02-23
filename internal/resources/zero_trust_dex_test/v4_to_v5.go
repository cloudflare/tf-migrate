package zero_trust_dex_test

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust DEX Test resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_device_dex_test", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_dex_test", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_dex_test"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_dex_test" || resourceType == "cloudflare_zero_trust_dex_test"
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_dex_test", "cloudflare_zero_trust_dex_test"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceName := tfhcl.GetResourceName(block)

	// rename cloudflare_device_dex_test → cloudflare_zero_trust_dex_test
	tfhcl.RenameResourceType(block, "cloudflare_device_dex_test", "cloudflare_zero_trust_dex_test")

	body := block.Body()

	// Transform data field: TypeList MaxItems:1 → SingleNestedAttribute
	tfhcl.ConvertSingleBlockToAttribute(body, "data", "data")

	// Remove computed timestamp fields if they exist in config (unlikely but possible)
	tfhcl.RemoveAttributes(body, "updated", "created")

	// Generate moved block for state migration
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState)
	// The moved block generated in TransformConfig triggers the provider's migration logic
	// This function is a no-op for zero_trust_dex_test migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
