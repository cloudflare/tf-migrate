package zero_trust_list

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
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

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// This function receives a single instance and needs to return the transformed instance JSON
	result := stateJSON.String()

	// Transform the instance attributes
	attrs := stateJSON.Get("attributes")
	if attrs.Exists() {
		var transformedItems []map[string]interface{}

		// IMPORTANT: Process items_with_description FIRST to match API order
		// The Cloudflare API returns items with descriptions first, then items without
		itemsWithDesc := attrs.Get("items_with_description")
		if itemsWithDesc.Exists() && itemsWithDesc.IsArray() {
			// Add items with description first
			itemsWithDesc.ForEach(func(k, v gjson.Result) bool {
				item := map[string]interface{}{
					"value": v.Get("value").String(),
				}
				if desc := v.Get("description").String(); desc != "" {
					item["description"] = desc
				}
				transformedItems = append(transformedItems, item)
				return true
			})
		}

		// Then process regular items (without descriptions)
		items := attrs.Get("items")
		if items.Exists() && items.IsArray() && len(items.Array()) > 0 {
			items.ForEach(func(k, v gjson.Result) bool {
				if v.Type == gjson.String {
					transformedItems = append(transformedItems, map[string]interface{}{
						"value": v.String(),
					})
				}
				return true
			})
		}

		// Set the combined items array (or delete if empty)
		if len(transformedItems) > 0 {
			result, _ = sjson.Set(result, "attributes.items", transformedItems)
		} else {
			result, _ = sjson.Delete(result, "attributes.items")
		}

		// Remove items_with_description from state
		result, _ = sjson.Delete(result, "attributes.items_with_description")
	}

	// Ensure schema_version is 0
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
