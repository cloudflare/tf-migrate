package healthcheck

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_healthcheck from v4 to v5
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator and registers it
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the migrator for cloudflare_healthcheck
	internal.RegisterMigrator("cloudflare_healthcheck", "v4", "v5", migrator)
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

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_healthcheck doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_healthcheck", "cloudflare_healthcheck"
}

// TransformConfig handles HCL configuration transformations
// Major transformation: Flat structure → Nested http_config/tcp_config based on type
//
// Since cloudflare_healthcheck is NOT renamed, this follows Path B:
// - Transform config in-place (flat → nested structure)
// - Return the modified block (no moved block needed)
// - RemoveOriginal: false (keep the transformed resource)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Get the type attribute to determine HTTP vs TCP
	typeAttr := body.GetAttribute("type")
	if typeAttr == nil {
		// If no type specified, v5 defaults to "HTTP", so treat as HTTP
		return m.transformToHTTPConfig(block)
	}

	// Extract the type value
	healthcheckType := hcl.ExtractStringFromAttribute(typeAttr)

	// Transform based on type
	if strings.ToUpper(healthcheckType) == "TCP" {
		return m.transformToTCPConfig(block)
	} else {
		// HTTP or HTTPS both use http_config
		return m.transformToHTTPConfig(block)
	}
}

// transformToHTTPConfig creates http_config nested attribute and moves HTTP fields into it
func (m *V4ToV5Migrator) transformToHTTPConfig(block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

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

	// Path B: Resource NOT renamed
	// Return the modified block, no moved block needed
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// transformToTCPConfig creates tcp_config nested attribute and moves TCP fields into it
func (m *V4ToV5Migrator) transformToTCPConfig(block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// TCP config only has method and port
	// method should be "connection_established"
	// port defaults to 80

	// List of fields that should move into tcp_config
	tcpFields := []string{"method", "port"}

	// Use the helper function to move attributes into tcp_config
	moved := hcl.MoveAttributesToNestedObject(body, "tcp_config", tcpFields)

	// Even if no fields were moved, return success
	_ = moved

	// Path B: Resource NOT renamed
	// Return the modified block, no moved block needed
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

// TransformState is a no-op for healthcheck migration.
// State transformation is now handled by the provider's StateUpgraders (UpgradeState).
// The moved block generated in TransformConfig triggers the provider's migration logic.
//
// Provider StateUpgraders handle:
// - Flat structure → Nested http_config/tcp_config based on type
// - Header Set → Map transformation
// - CheckRegions List conversion
// - All field transformations
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// The moved block generated in TransformConfig triggers the provider's migration logic
	// This function is a no-op for healthcheck migration
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider handles state transformation, not tf-migrate.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}


func init() {
	// Register the migrator when the package is imported
	NewV4ToV5Migrator()
}
