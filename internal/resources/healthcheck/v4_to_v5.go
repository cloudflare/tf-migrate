package healthcheck

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles the migration of cloudflare_healthcheck from v4 to v5
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator and registers it
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the migrator for cloudflare_healthcheck
	internal.Register("cloudflare_healthcheck", "v4", "v5", func() transform.ResourceTransformer {
		return migrator
	})
	return migrator
}

// CanHandle determines if this migrator can handle the given resource type
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_healthcheck"
}

// GetResourceType returns the resource type this migrator handles
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_healthcheck"
}

// Preprocess handles string-level transformations before HCL parsing
// No preprocessing needed for healthcheck migration
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles HCL configuration transformations
// Major transformation: Flat structure → Nested http_config/tcp_config based on type
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Get the type attribute to determine HTTP vs TCP
	typeAttr := body.GetAttribute("type")
	if typeAttr == nil {
		// If no type specified, v5 defaults to "HTTP", so treat as HTTP
		return m.transformToHTTPConfig(body)
	}

	// Extract the type value
	healthcheckType := hcl.ExtractStringFromAttribute(typeAttr)

	// Transform based on type
	if strings.ToUpper(healthcheckType) == "TCP" {
		return m.transformToTCPConfig(body)
	} else {
		// HTTP or HTTPS both use http_config
		return m.transformToHTTPConfig(body)
	}
}

// transformToHTTPConfig creates http_config nested attribute and moves HTTP fields into it
func (m *V4ToV5Migrator) transformToHTTPConfig(body *hclwrite.Body) (*transform.TransformResult, error) {
	// List of fields that should move into http_config
	httpFields := []string{
		"method", "port", "path", "expected_codes", "expected_body",
		"follow_redirects", "allow_insecure",
	}

	// Collect field tokens before moving
	fields := make(map[string]hclwrite.Tokens)
	for _, fieldName := range httpFields {
		if attr := body.GetAttribute(fieldName); attr != nil {
			fields[fieldName] = attr.Expr().BuildTokens(nil)
		}
	}

	// Handle header blocks specially - they need to convert from Set to Map
	// v4: header { header = "Host" values = ["example.com"] }
	// v5: http_config = { header = { "Host" = ["example.com"] } }
	headerTokens, err := m.buildHeaderMapTokens(body)
	if err != nil {
		return nil, err
	}
	if headerTokens != nil {
		fields["header"] = headerTokens
	}

	// Create the http_config nested attribute if we have any fields
	if len(fields) > 0 {
		hcl.CreateNestedAttributeFromFields(body, "http_config", fields)

		// Remove the original attributes that moved into http_config
		for _, fieldName := range httpFields {
			body.RemoveAttribute(fieldName)
		}

		// Remove header blocks
		hcl.RemoveBlocksByType(body, "header")
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{},
		RemoveOriginal: false,
	}, nil
}

// transformToTCPConfig creates tcp_config nested attribute and moves TCP fields into it
func (m *V4ToV5Migrator) transformToTCPConfig(body *hclwrite.Body) (*transform.TransformResult, error) {
	// TCP config only has method and port
	// method should be "connection_established"
	// port defaults to 80

	// List of fields that should move into tcp_config
	tcpFields := []string{"method", "port"}

	// Use the helper function to move attributes into tcp_config
	moved := hcl.MoveAttributesToNestedObject(body, "tcp_config", tcpFields)

	// Even if no fields were moved, return success
	_ = moved

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{},
		RemoveOriginal: false,
	}, nil
}

// buildHeaderMapTokens converts v4 header blocks to v5 header map tokens
// v4: header { header = "Host" values = ["example.com"] }
// v5: header = { "Host" = ["example.com"] }
func (m *V4ToV5Migrator) buildHeaderMapTokens(body *hclwrite.Body) (hclwrite.Tokens, error) {
	// Find all header blocks
	headerBlocks := hcl.FindBlocksByType(body, "header")
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

// TransformState handles JSON state transformations
// Major transformation: Restructure based on type field + type conversions
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		// Set schema_version even for invalid instances
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Convert numeric fields (Int → Float64)
	numericFields := []string{
		"consecutive_fails", "consecutive_successes", "retries",
		"timeout", "interval", "port",
	}

	for _, field := range numericFields {
		if fieldVal := attrs.Get(field); fieldVal.Exists() {
			floatVal := state.ConvertGjsonValue(fieldVal)
			result, _ = sjson.Set(result, "attributes."+field, floatVal)
		}
	}

	// Get the type to determine HTTP vs TCP
	healthcheckType := attrs.Get("type").String()

	if strings.ToUpper(healthcheckType) == "TCP" {
		// Create tcp_config object
		result = m.createTCPConfig(result, attrs)
	} else {
		// Create http_config object (for HTTP and HTTPS)
		result = m.createHTTPConfig(result, attrs)
	}

	// Remove computed fields that don't exist in v5 state or exist but shouldn't be touched
	// Note: created_on, modified_on, id, status, failure_reason are all computed
	// We should NOT remove them, just leave them as-is for provider to handle

	// Set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// createHTTPConfig creates the http_config nested object in state
func (m *V4ToV5Migrator) createHTTPConfig(stateJSON string, attrs gjson.Result) string {
	httpConfig := make(map[string]interface{})

	// Move HTTP-specific fields into http_config
	httpFields := map[string]string{
		"method":           "method",
		"port":             "port",
		"path":             "path",
		"expected_codes":   "expected_codes",
		"expected_body":    "expected_body",
		"follow_redirects": "follow_redirects",
		"allow_insecure":   "allow_insecure",
	}

	for oldField, newField := range httpFields {
		if val := attrs.Get(oldField); val.Exists() {
			httpConfig[newField] = state.ConvertGjsonValue(val)
		}
	}

	// Handle header transformation (Set → Map)
	if header := attrs.Get("header"); header.Exists() && header.IsArray() {
		headerMap := m.transformHeaderSetToMap(header)
		if len(headerMap) > 0 {
			httpConfig["header"] = headerMap
		}
	}

	// Set the http_config if we have any fields
	if len(httpConfig) > 0 {
		stateJSON, _ = sjson.Set(stateJSON, "attributes.http_config", httpConfig)

		// Remove the root-level fields that moved into http_config
		for oldField := range httpFields {
			if attrs.Get(oldField).Exists() {
				stateJSON, _ = sjson.Delete(stateJSON, "attributes."+oldField)
			}
		}
		// Remove header from root if it exists
		if attrs.Get("header").Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, "attributes.header")
		}
	}

	return stateJSON
}

// createTCPConfig creates the tcp_config nested object in state
func (m *V4ToV5Migrator) createTCPConfig(stateJSON string, attrs gjson.Result) string {
	tcpConfig := make(map[string]interface{})

	// Move TCP-specific fields into tcp_config
	if method := attrs.Get("method"); method.Exists() {
		tcpConfig["method"] = method.String()
	}
	if port := attrs.Get("port"); port.Exists() {
		tcpConfig["port"] = state.ConvertGjsonValue(port)
	}

	// Set the tcp_config if we have any fields
	if len(tcpConfig) > 0 {
		stateJSON, _ = sjson.Set(stateJSON, "attributes.tcp_config", tcpConfig)

		// Remove the root-level fields that moved into tcp_config
		if attrs.Get("method").Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, "attributes.method")
		}
		if attrs.Get("port").Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, "attributes.port")
		}
	}

	return stateJSON
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
