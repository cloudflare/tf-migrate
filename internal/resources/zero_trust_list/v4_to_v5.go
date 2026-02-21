package zero_trust_list

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust List resources from v4 to v5
type V4ToV5Migrator struct {
	oldType string
	newType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType: "cloudflare_teams_list",
		newType: "cloudflare_zero_trust_list",
	}
	internal.RegisterMigrator("cloudflare_teams_list", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_teams_list", "cloudflare_zero_trust_list"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceName := tfhcl.GetResourceName(block)

	// Rename cloudflare_teams_list to cloudflare_zero_trust_list
	tfhcl.RenameResourceType(block, "cloudflare_teams_list", "cloudflare_zero_trust_list")

	body := block.Body()

	// Transform items (string array) and items_with_description (blocks)
	// into a single items attribute with object array
	tfhcl.MergeAttributeAndBlocksToObjectArray(
		body,
		"items",                  // arrayAttrName
		"items_with_description", // blockType
		"items",                  // outputAttrName
		"value",                  // primaryField
		[]string{"description"},  // optionalFields
		true,                     // blocksFirst (to match API order)
	)

	// Generate moved block for resource rename
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op for zero_trust_list migration.
// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
