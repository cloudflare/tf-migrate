package zero_trust_list

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

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
	// Register the OLD (v4) resource name: cloudflare_teams_list
	internal.RegisterMigrator("cloudflare_teams_list", "v4", "v5", migrator)
	// Also register the NEW (v5) resource name so already-renamed resources are still processed (BUGS-2009)
	internal.RegisterMigrator("cloudflare_zero_trust_list", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Accept both the OLD (v4) name and the NEW (v5) name.
	// The v5 name is needed when a resource was already renamed (e.g. by a prior
	// partial migration) but its items_with_description blocks were not yet converted (BUGS-2009).
	return resourceType == m.oldType || resourceType == m.newType
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	// Both cloudflare_teams_list and cloudflare_zero_trust_list were valid v4 names
	return []string{"cloudflare_teams_list", "cloudflare_zero_trust_list"}, "cloudflare_zero_trust_list"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original resource type before any modifications (for moved block generation)
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Rename cloudflare_teams_list to cloudflare_zero_trust_list
	// Skip when already v5-named (resource was renamed in a prior migration run).
	if originalResourceType == "cloudflare_teams_list" {
		tfhcl.RenameResourceType(block, "cloudflare_teams_list", "cloudflare_zero_trust_list")
	}

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

	// Generate moved block only when renaming from v4 name.
	// If already v5-named: no rename, no moved block — just apply items transformations above.
	if originalResourceType == "cloudflare_teams_list" {
		from := originalResourceType + "." + resourceName
		to := m.newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
