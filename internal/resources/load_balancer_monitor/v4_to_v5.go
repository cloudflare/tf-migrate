package load_balancer_monitor

import (
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
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_load_balancer_monitor", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (unchanged from v4)
	return "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done with HCL helpers
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Transform header blocks to map attribute
	// v4: header { header = "Host" values = ["example.com"] }
	// v5: header = { "Host" = ["example.com"] }
	headerTokens, err := m.buildHeaderMapTokens(body)
	if err != nil {
		return nil, err
	}
	if headerTokens != nil {
		body.SetAttributeRaw("header", headerTokens)
		// Remove the old header blocks
		tfhcl.RemoveBlocksByType(body, "header")
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// buildHeaderMapTokens converts v4 header blocks to v5 header map tokens
// v4: header { header = "Host" values = ["example.com"] }
// v5: header = { "Host" = ["example.com"] }
func (m *V4ToV5Migrator) buildHeaderMapTokens(body *hclwrite.Body) (hclwrite.Tokens, error) {
	// Find all header blocks
	headerBlocks := tfhcl.FindBlocksByType(body, "header")
	if len(headerBlocks) == 0 {
		return nil, nil
	}

	// Build a list of object attributes for the header map
	var headerAttrs []hclwrite.ObjectAttrTokens

	for _, block := range headerBlocks {
		blockBody := block.Body()

		// Get the header name
		headerAttr := blockBody.GetAttribute("header")
		if headerAttr == nil {
			continue
		}

		// Get the values
		valuesAttr := blockBody.GetAttribute("values")
		if valuesAttr == nil {
			continue
		}

		// Use the header value as the map key and values as the map value
		nameTokens := headerAttr.Expr().BuildTokens(nil)
		valueTokens := valuesAttr.Expr().BuildTokens(nil)

		headerAttrs = append(headerAttrs, hclwrite.ObjectAttrTokens{
			Name:  nameTokens,
			Value: valueTokens,
		})
	}

	if len(headerAttrs) == 0 {
		return nil, nil
	}

	// Create the object tokens for the header map
	return hclwrite.TokensForObject(headerAttrs), nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		// Set schema_version even for invalid instances
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Convert numeric fields (TypeInt → Int64Attribute)
	numericFields := []string{
		"interval", "port", "retries", "timeout",
		"consecutive_down", "consecutive_up",
	}

	for _, field := range numericFields {
		if fieldVal := attrs.Get(field); fieldVal.Exists() {
			floatVal := state.ConvertToFloat64(fieldVal)
			result, _ = sjson.Set(result, "attributes."+field, floatVal)
		}
	}

	// Add v5 default values for fields that were optional without defaults in v4
	// These defaults prevent PATCH operations when migrating
	result = state.EnsureField(result, "attributes", attrs, "allow_insecure", false)
	result = state.EnsureField(result, "attributes", attrs, "description", "")
	result = state.EnsureField(result, "attributes", attrs, "expected_body", "")
	result = state.EnsureField(result, "attributes", attrs, "expected_codes", "")
	result = state.EnsureField(result, "attributes", attrs, "follow_redirects", false)
	result = state.EnsureField(result, "attributes", attrs, "probe_zone", "")

	// Transform header Set to Map
	// v4: [{"header": "Host", "values": ["example.com"]}, ...]
	// v5: {"Host": ["example.com"], ...}
	if header := attrs.Get("header"); header.Exists() && header.IsArray() {
		headerMap := m.transformHeaderSetToMap(header)
		if len(headerMap) > 0 {
			result, _ = sjson.Set(result, "attributes.header", headerMap)
		} else {
			// Remove empty header
			result, _ = sjson.Delete(result, "attributes.header")
		}
	}

	// Re-parse attrs after transformations to get updated structure
	updatedAttrs := gjson.Parse(result).Get("attributes")

	// Transform empty/zero values to null for fields not explicitly set in config
	// This handles consecutive_down and consecutive_up which have Default: 0 in v4
	// If they're 0 and not explicitly set in config, they should be removed from state
	// because v5 treats absence as "use API default" (same as v4's Default: 0)
	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
		Ctx:              ctx,
		Result:           result,
		FieldPath:        "attributes",
		FieldResult:      updatedAttrs,
		ResourceName:     resourceName,
		HCLAttributePath: "",
		CanHandle:        m.CanHandle,
	})

	// Set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformHeaderSetToMap converts v4 header Set structure to v5 Map structure
// v4: [{"header": "Host", "values": ["example.com"]}, {"header": "User-Agent", "values": ["Bot"]}]
// v5: {"Host": ["example.com"], "User-Agent": ["Bot"]}
func (m *V4ToV5Migrator) transformHeaderSetToMap(headerSet gjson.Result) map[string][]string {
	headerMap := make(map[string][]string)

	if !headerSet.IsArray() {
		return headerMap
	}

	for _, item := range headerSet.Array() {
		headerName := item.Get("header").String()
		values := item.Get("values")

		if headerName == "" || !values.Exists() {
			continue
		}

		// Extract values array
		var valuesList []string
		if values.IsArray() {
			for _, val := range values.Array() {
				valuesList = append(valuesList, val.String())
			}
		}

		if len(valuesList) > 0 {
			headerMap[headerName] = valuesList
		}
	}

	return headerMap
}

func init() {
	// Register the migrator when the package is imported
	NewV4ToV5Migrator()
}
