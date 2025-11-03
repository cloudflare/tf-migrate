package list

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_list", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_list"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_list"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Config transformation is extremely complex (involves list_item_merge cross-resource logic)
	// Users should use the provider's migrate tool for config transformation
	// This migrator focuses on state transformation only
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	// Get the kind to know how to process items
	kind := stateJSON.Get("attributes.kind").String()

	// Get the item array
	itemPath := "attributes.item"
	items := stateJSON.Get(itemPath)

	if !items.Exists() {
		return result, nil
	}

	if items.IsArray() {
		itemArray := items.Array()
		if len(itemArray) == 0 {
			// Empty item array - remove it and set num_items: 0
			result, _ = sjson.Delete(result, itemPath)
			result, _ = sjson.Set(result, "attributes.num_items", 0)
		} else {
			// Transform items
			transformedItems := m.transformItems(itemArray, kind)
			result, _ = sjson.Set(result, "attributes.items", transformedItems)
			result, _ = sjson.Delete(result, itemPath)
			result, _ = sjson.Set(result, "attributes.num_items", len(transformedItems))
		}
	}

	return result, nil
}

// transformItems transforms the v4 item array to v5 items format
func (m *V4ToV5Migrator) transformItems(items []gjson.Result, kind string) []interface{} {
	var transformed []interface{}

	for _, item := range items {
		transformedItem := m.transformItem(item, kind)
		if transformedItem != nil {
			transformed = append(transformed, transformedItem)
		}
	}

	return transformed
}

// transformItem transforms a single v4 item to v5 format
func (m *V4ToV5Migrator) transformItem(item gjson.Result, kind string) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy comment if present
	comment := item.Get("comment")
	if comment.Exists() {
		result["comment"] = comment.Value()
	}

	// Get the value array (should have 1 element)
	valueArray := item.Get("value")
	if !valueArray.Exists() || !valueArray.IsArray() {
		return result
	}

	values := valueArray.Array()
	if len(values) == 0 {
		return result
	}

	value := values[0] // Take first element

	// Extract the appropriate field based on kind
	switch kind {
	case "ip":
		// value.ip → flatten to top level
		ip := value.Get("ip")
		if ip.Exists() {
			result["ip"] = ip.Value()
		}

	case "asn":
		// value.asn → flatten to top level
		asn := value.Get("asn")
		if asn.Exists() {
			result["asn"] = asn.Value()
		}

	case "hostname":
		// value.hostname[0] → flatten to top level as object
		hostname := value.Get("hostname")
		if hostname.Exists() && hostname.IsArray() {
			hostnameArray := hostname.Array()
			if len(hostnameArray) > 0 {
				result["hostname"] = hostnameArray[0].Value()
			}
		}

	case "redirect":
		// value.redirect[0] → flatten to top level as object
		// Also convert boolean string fields ("enabled"/"disabled") to actual booleans
		redirect := value.Get("redirect")
		if redirect.Exists() && redirect.IsArray() {
			redirectArray := redirect.Array()
			if len(redirectArray) > 0 {
				redirectObj := redirectArray[0]
				redirectResult := make(map[string]interface{})

				// Copy all fields, converting boolean strings
				redirectMap := redirectObj.Map()
				for key, val := range redirectMap {
					switch key {
					case "include_subdomains", "subpath_matching", "preserve_query_string", "preserve_path_suffix":
						// Convert "enabled"/"disabled" to boolean
						if val.String() == "enabled" {
							redirectResult[key] = true
						} else if val.String() == "disabled" {
							redirectResult[key] = false
						} else {
							redirectResult[key] = val.Value()
						}
					default:
						redirectResult[key] = val.Value()
					}
				}

				result["redirect"] = redirectResult
			}
		}
	}

	return result
}
