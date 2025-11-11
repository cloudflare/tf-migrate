package load_balancer_monitor

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of load balancer monitor resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_load_balancer_monitor", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_monitor"
}

// Preprocess is not needed - we do everything in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert header blocks to map attribute
	m.transformHeaderBlocks(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformHeaderBlocks converts multiple v4 header blocks to a v5 map attribute
func (m *V4ToV5Migrator) transformHeaderBlocks(body *hclwrite.Body) {
	headerBlocks := tfhcl.FindBlocksByType(body, "header")
	if len(headerBlocks) == 0 {
		return
	}

	// Collect header entries as ObjectAttrTokens
	var headerAttrs []hclwrite.ObjectAttrTokens

	for _, headerBlock := range headerBlocks {
		headerBody := headerBlock.Body()

		// Get the "header" attribute (the header name)
		headerAttr := headerBody.GetAttribute("header")
		if headerAttr == nil {
			continue
		}

		// Get the "values" attribute (the header values)
		valuesAttr := headerBody.GetAttribute("values")
		if valuesAttr == nil {
			continue
		}

		headerAttrs = append(headerAttrs, hclwrite.ObjectAttrTokens{
			Name:  headerAttr.Expr().BuildTokens(nil),
			Value: valuesAttr.Expr().BuildTokens(nil),
		})
	}

	if len(headerAttrs) == 0 {
		return
	}

	// Build the header map using TokensForObject
	headerMapTokens := hclwrite.TokensForObject(headerAttrs)

	// Set the header attribute with the map
	body.SetAttributeRaw("header", headerMapTokens)

	// Remove the header blocks
	tfhcl.RemoveBlocksByType(body, "header")
}

// TransformState transforms the state JSON from v4 to v5
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Convert numeric fields from int to float64
	// Fields that are computed_optional should keep zero values
	// Fields that are optional-only should remove zero values
	computedOptionalFields := []string{
		"interval",
		"retries",
		"timeout",
	}

	optionalOnlyFields := []string{
		"port",
		"consecutive_down",
		"consecutive_up",
	}

	// Convert computed_optional fields (keep zero values)
	for _, field := range computedOptionalFields {
		if value := attrs.Get(field); value.Exists() {
			floatVal := state.ConvertToFloat64(value)
			result, _ = sjson.Set(result, "attributes."+field, floatVal)
		}
	}

	// Convert optional-only fields (remove zero values)
	for _, field := range optionalOnlyFields {
		if value := attrs.Get(field); value.Exists() {
			floatVal := state.ConvertToFloat64(value)
			// Only set if non-zero
			if floatVal != 0.0 && floatVal != 0 {
				result, _ = sjson.Set(result, "attributes."+field, floatVal)
			} else {
				// Remove zero values for optional-only fields
				result, _ = sjson.Delete(result, "attributes."+field)
			}
		}
	}

	// Transform header field from array-of-objects to map-of-arrays
	if header := attrs.Get("header"); header.Exists() && header.IsArray() {
		headerMap := make(map[string][]string)

		header.ForEach(func(_, value gjson.Result) bool {
			headerName := value.Get("header").String()
			values := value.Get("values")

			if headerName != "" && values.Exists() {
				var valuesList []string
				if values.IsArray() {
					values.ForEach(func(_, v gjson.Result) bool {
						valuesList = append(valuesList, v.String())
						return true
					})
				}
				headerMap[headerName] = valuesList
			}
			return true
		})

		if len(headerMap) > 0 {
			headerJSON, _ := json.Marshal(headerMap)
			result, _ = sjson.SetRaw(result, "attributes.header", string(headerJSON))
		} else {
			// Remove empty header field
			result, _ = sjson.Delete(result, "attributes.header")
		}
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
