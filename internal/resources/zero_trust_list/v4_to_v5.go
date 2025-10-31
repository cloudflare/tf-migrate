package zero_trust_list

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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

// Preprocess handles the complex transformation of items and items_with_description
func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Process each resource block separately to avoid mixing items
	// Match resource blocks properly including nested blocks
	resourcePattern := regexp.MustCompile(`(?ms)(resource\s+"cloudflare_(?:teams|zero_trust)_list"\s+"[^"]+"\s*\{(?:[^{}]|\{[^}]*\})*\})`)
	
	content = resourcePattern.ReplaceAllStringFunc(content, func(resourceBlock string) string {
		var itemsWithDescTransformed []string
		
		// First, extract items_with_description blocks for this resource
		itemsWithDescPattern := regexp.MustCompile(`(?ms)\s*items_with_description\s*\{([^}]*)\}\s*\n?`)
		matches := itemsWithDescPattern.FindAllStringSubmatch(resourceBlock, -1)
		
		for _, match := range matches {
			blockContent := match[1]
			
			// Extract value and description
			valuePattern := regexp.MustCompile(`value\s*=\s*"([^"]*)"`)
			descPattern := regexp.MustCompile(`description\s*=\s*"([^"]*)"`)
			
			valueMatch := valuePattern.FindStringSubmatch(blockContent)
			if valueMatch != nil {
				item := fmt.Sprintf(`    { value = "%s"`, valueMatch[1])
				descMatch := descPattern.FindStringSubmatch(blockContent)
				if descMatch != nil {
					item += fmt.Sprintf(`, description = "%s"`, descMatch[1])
				}
				item += " }"
				itemsWithDescTransformed = append(itemsWithDescTransformed, item)
			}
		}
		
		// Remove items_with_description blocks
		resourceBlock = itemsWithDescPattern.ReplaceAllString(resourceBlock, "")
		
		// Now handle simple items array - match the entire line including newline
		itemsPattern := regexp.MustCompile(`(?m)^(\s*)items\s*=\s*\[([^\]]*)\]\s*\n?`)
		
		hasExistingItems := itemsPattern.MatchString(resourceBlock)
		
		if hasExistingItems {
			resourceBlock = itemsPattern.ReplaceAllStringFunc(resourceBlock, func(match string) string {
				submatches := itemsPattern.FindStringSubmatch(match)
				if len(submatches) < 3 {
					return match
				}
				
				indent := submatches[1]
				itemsContent := strings.TrimSpace(submatches[2])
				
				// Extract string items for this specific items array
				var localRegularItems []string
				if itemsContent != "" {
					stringPattern := regexp.MustCompile(`"([^"]*)"`)
					stringMatches := stringPattern.FindAllStringSubmatch(itemsContent, -1)
					
					for _, sm := range stringMatches {
						localRegularItems = append(localRegularItems, 
							fmt.Sprintf(`    { value = "%s" }`, sm[1]))
					}
				}
				
				// Build items blocks
				var itemsBlocks []string
				
				// Add regular items as blocks
				for _, item := range localRegularItems {
					// Extract just the value part from the object literal
					valueMatch := regexp.MustCompile(`{\s*value\s*=\s*"([^"]+)"\s*}`).FindStringSubmatch(item)
					if len(valueMatch) > 1 {
						itemsBlocks = append(itemsBlocks, fmt.Sprintf("%sitems {\n%s  value = \"%s\"\n%s}", 
							indent, indent, valueMatch[1], indent))
					}
				}
				
				// Add items with descriptions as blocks
				for _, item := range itemsWithDescTransformed {
					// Extract value and description from the object literal
					valueMatch := regexp.MustCompile(`{\s*value\s*=\s*"([^"]+)"(?:,\s*description\s*=\s*"([^"]+)")?\s*}`).FindStringSubmatch(item)
					if len(valueMatch) > 1 {
						if len(valueMatch) > 2 && valueMatch[2] != "" {
							itemsBlocks = append(itemsBlocks, fmt.Sprintf("%sitems {\n%s  value       = \"%s\"\n%s  description = \"%s\"\n%s}", 
								indent, indent, valueMatch[1], indent, valueMatch[2], indent))
						} else {
							itemsBlocks = append(itemsBlocks, fmt.Sprintf("%sitems {\n%s  value = \"%s\"\n%s}", 
								indent, indent, valueMatch[1], indent))
						}
					}
				}
				
				// Build the new items blocks
				if len(itemsBlocks) > 0 {
					return strings.Join(itemsBlocks, "\n") + "\n"
				}
				// Remove empty items array entirely including the line - v4 stores as nil, v5 should too
				return ""
			})
		} else if len(itemsWithDescTransformed) > 0 {
			// No existing items array but we have items_with_description
			// We need to add the items blocks before the closing brace
			closePattern := regexp.MustCompile(`(?ms)(.*?)(\})`)
			resourceBlock = closePattern.ReplaceAllStringFunc(resourceBlock, func(match string) string {
				submatches := closePattern.FindStringSubmatch(match)
				if len(submatches) < 3 {
					return match
				}
				
				// Trim any trailing whitespace from the resource body
				body := strings.TrimRight(submatches[1], " \t\n")
				
				// Build items blocks from the transformed items
				var itemsBlocks []string
				for _, item := range itemsWithDescTransformed {
					// Extract value and description from the object literal
					valueMatch := regexp.MustCompile(`{\s*value\s*=\s*"([^"]+)"(?:,\s*description\s*=\s*"([^"]+)")?\s*}`).FindStringSubmatch(item)
					if len(valueMatch) > 1 {
						if len(valueMatch) > 2 && valueMatch[2] != "" {
							itemsBlocks = append(itemsBlocks, fmt.Sprintf("  items {\n    value       = \"%s\"\n    description = \"%s\"\n  }", 
								valueMatch[1], valueMatch[2]))
						} else {
							itemsBlocks = append(itemsBlocks, fmt.Sprintf("  items {\n    value = \"%s\"\n  }", 
								valueMatch[1]))
						}
					}
				}
				
				if len(itemsBlocks) > 0 {
					return fmt.Sprintf("%s\n%s\n%s",
						body,
						strings.Join(itemsBlocks, "\n"),
						submatches[2])
				}
				return match
			})
		}
		
		return resourceBlock
	})
	
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename cloudflare_teams_list to cloudflare_zero_trust_list
	tfhcl.RenameResourceType(block, "cloudflare_teams_list", "cloudflare_zero_trust_list")
	
	// The complex items transformation is handled in preprocessing
	
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	// This function receives a single instance and needs to return the transformed instance JSON
	result := instance.String()
	
	// Transform the instance attributes
	attrs := instance.Get("attributes")
	if attrs.Exists() {
		// Transform items field from string array to object array
		items := attrs.Get("items")
		if items.Exists() && items.IsArray() {
			// Only process and set items if the array is not empty
			if len(items.Array()) > 0 {
				var transformedItems []map[string]interface{}
				
				items.ForEach(func(k, v gjson.Result) bool {
					if v.Type == gjson.String {
						transformedItems = append(transformedItems, map[string]interface{}{
							"value": v.String(),
						})
					}
					return true
				})
				
				// Only set items if we have transformed items
				if len(transformedItems) > 0 {
					result, _ = sjson.Set(result, "attributes.items", transformedItems)
				}
			} else {
				// If items exists but is empty, explicitly delete it
				result, _ = sjson.Delete(result, "attributes.items")
			}
		}
		// Don't set items if it doesn't exist - matches v4/v5 provider behavior
		
		// Handle items_with_description
		itemsWithDesc := attrs.Get("items_with_description")
		if itemsWithDesc.Exists() && itemsWithDesc.IsArray() {
			var currentItems []map[string]interface{}
			
			// Get existing transformed items
			existing := gjson.Get(result, "attributes.items")
			if existing.Exists() {
				json.Unmarshal([]byte(existing.Raw), &currentItems)
			}
			
			// Add items with description
			itemsWithDesc.ForEach(func(k, v gjson.Result) bool {
				item := map[string]interface{}{
					"value": v.Get("value").String(),
				}
				if desc := v.Get("description").String(); desc != "" {
					item["description"] = desc
				}
				currentItems = append(currentItems, item)
				return true
			})
			
			result, _ = sjson.Set(result, "attributes.items", currentItems)
			result, _ = sjson.Delete(result, "attributes.items_with_description")
		}
		
		// Don't add computed fields during migration - they should be preserved as-is or removed
	}
	
	// Ensure schema_version is 0
	result, _ = sjson.Set(result, "schema_version", 0)
	
	return result, nil
}
