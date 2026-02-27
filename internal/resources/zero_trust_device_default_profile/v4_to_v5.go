package zero_trust_device_default_profile

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_split_tunnel"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zero trust device profile resources from v4 to v5
// This migrator routes to either default or custom profile based on match/precedence presence
type V4ToV5Migrator struct {
	oldType           string
	oldTypeDeprecated string
	newTypeDefault    string
	newTypeCustom     string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType:           "cloudflare_zero_trust_device_profiles",
		oldTypeDeprecated: "cloudflare_device_settings_policy",
		newTypeDefault:    "cloudflare_zero_trust_device_default_profile",
		newTypeCustom:     "cloudflare_zero_trust_device_custom_profile",
	}

	// Register BOTH old resource names - migrator will route based on match/precedence
	internal.RegisterMigrator("cloudflare_zero_trust_device_profiles", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_device_settings_policy", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return empty string - the actual type will be determined per-resource in TransformState
	// and set via ctx.StateTypeRenames to avoid state bleeding between resources
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both v4 resource names AND both v5 resource names
	return resourceType == m.oldType ||
		resourceType == m.oldTypeDeprecated ||
		resourceType == m.newTypeDefault ||
		resourceType == m.newTypeCustom
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Check if this is JSON state content (starts with '{')
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") {
		// Process cross-resource state migrations (merge split_tunnel into device profiles, remove split_tunnels)
		return zero_trust_split_tunnel.ProcessCrossResourceStateMigration(content)
	}
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from old names to new name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_zero_trust_device_profiles", "cloudflare_zero_trust_device_default_profile"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Process cross-resource migration - merge split_tunnel resources into device profiles
	// This is idempotent - safe to call multiple times
	if ctx.CFGFile != nil {
		zero_trust_split_tunnel.ProcessCrossResourceConfigMigration(ctx.CFGFile)
	}

	body := block.Body()

	// 1. Check if this is a custom profile (has match and precedence) or default profile
	// Priority: default field > presence of match+precedence
	defaultAttr := body.GetAttribute("default")
	matchAttr := body.GetAttribute("match")
	precedenceAttr := body.GetAttribute("precedence")

	hasMatch := matchAttr != nil
	hasPrecedence := precedenceAttr != nil

	// Check if default is explicitly set to true
	isExplicitDefault := false
	if defaultAttr != nil {
		defaultValue, _ := tfhcl.ExtractBoolFromAttribute(defaultAttr)
		isExplicitDefault = defaultValue
	}

	// If default=true explicitly, it's a default profile (even if match/precedence present - invalid config)
	// Otherwise, if it has match AND precedence, it's a custom profile
	// Otherwise, default to default profile
	isCustomProfile := !isExplicitDefault && hasMatch && hasPrecedence

	var newResourceType string
	if isCustomProfile {
		newResourceType = m.newTypeCustom
	} else {
		newResourceType = m.newTypeDefault
	}

	// 2. Rename resource type to appropriate v5 resource
	currentType := tfhcl.GetResourceType(block)
	tfhcl.RenameResourceType(block, currentType, newResourceType)

	// 3. Remove fields based on profile type
	if isCustomProfile {
		// Custom profile: update precedence to avoid conflicts, then remove 'default' and 'enabled'
		// Registry provider has precedence as Required, so we must keep it in config
		// Set to a high value (999 + original value) to avoid conflicts with existing policies
		// The API will accept this new value and update the resource
		if precedenceAttr != nil {
			// Extract original precedence value
			originalPrecedence := 100.0 // default
			if tokens := precedenceAttr.Expr().BuildTokens(nil); len(tokens) > 0 {
				precedenceStr := string(tokens[0].Bytes)
				// Try to parse as float
				if val := gjson.Parse(precedenceStr); val.Exists() {
					originalPrecedence = val.Float()
				}
			}
			// Set to a high value to avoid conflicts
			newPrecedence := 900 + originalPrecedence
			tfhcl.SetAttribute(body, "precedence", newPrecedence)
		}

		tfhcl.RemoveAttributes(body, "default", "enabled")
	} else {
		// Default profile: remove custom-only fields
		tfhcl.RemoveAttributes(body, "name", "description", "match", "precedence", "enabled", "default")
	}

	// 4. Handle service_mode_v2 default value mismatch (applies to both profile types)
	// v4 default: mode="warp" (with port unset), v5 has no default
	// Remove service_mode_v2_mode if it's "warp" (v4 default) and port is not set
	modeAttr := body.GetAttribute("service_mode_v2_mode")
	portAttr := body.GetAttribute("service_mode_v2_port")
	if modeAttr != nil && portAttr == nil {
		modeValue := tfhcl.ExtractStringFromAttribute(modeAttr)
		if modeValue == "warp" {
			// Remove the mode field - it's just the v4 default
			tfhcl.RemoveAttributes(body, "service_mode_v2_mode")
			modeAttr = nil
		}
	}

	// 5. Merge service_mode_v2_mode + service_mode_v2_port → service_mode_v2 nested object (both profile types)
	// Only do this if we still have at least one field to merge
	if modeAttr != nil || portAttr != nil {
		// Need to rename fields first to strip prefix, then move to nested object
		tfhcl.RenameAttribute(body, "service_mode_v2_mode", "mode")
		tfhcl.RenameAttribute(body, "service_mode_v2_port", "port")
		tfhcl.MoveAttributesToNestedObject(body, "service_mode_v2", []string{
			"mode",
			"port",
		})
	}

	// 6. Handle tunnel_protocol - preserve in config (both profile types)
	// v5 schema: Computed + Optional with default ""
	// API behavior: returns "wireguard" as the actual computed value
	// If user explicitly set tunnel_protocol in v4, keep it in v5 to avoid drift
	// The v5 schema default ("") doesn't match API behavior ("wireguard"), so we must keep explicit values

	// 7. Handle exclude/include fields (both profile types)
	// These fields are Optional+Computed in v5 schema, meaning:
	// - If not specified in config, API will populate with defaults (no drift)
	// - If specified in config, that value will be enforced
	// Therefore, don't add them if they weren't in v4 config - let API handle defaults

	// 8. Default profile specific: Add required fields that don't exist in custom profile
	if !isCustomProfile {
		// Add register_interface_ip_with_dns with API default value (true)
		tfhcl.EnsureAttribute(body, "register_interface_ip_with_dns", true)

		// Add sccm_vpn_boundary_support with default value
		tfhcl.EnsureAttribute(body, "sccm_vpn_boundary_support", false)
	}

	// 9. Generate moved block for Terraform 1.8+ state migration
	// The moved block triggers the provider's MoveState handler
	resourceName := tfhcl.GetResourceName(block)
	movedBlock := m.createMovedBlock(currentType, resourceName, newResourceType, resourceName)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// createMovedBlock creates a moved block for state migration tracking
// This triggers Terraform 1.8+ to call the provider's MoveState handler
func (m *V4ToV5Migrator) createMovedBlock(fromType, fromName, toType, toName string) *hclwrite.Block {
	from := fmt.Sprintf("%s.%s", fromType, fromName)
	to := fmt.Sprintf("%s.%s", toType, toName)
	return tfhcl.CreateMovedBlock(from, to)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// STATE TRANSFORMATION DISABLED - Provider handles state migration via MoveState + UpgradeState
	//
	// This migrator now uses Provider StateUpgraders (NEW pattern):
	// - tf-migrate generates moved blocks (handled in TransformConfig)
	// - Terraform 1.8+ triggers provider's MoveState when it sees moved blocks
	// - Provider's MoveState + UpgradeState handle all state transformations:
	//   * Type conversions (Int64 → Float64)
	//   * Structure changes (flatten → nested)
	//   * ID extraction (custom profile)
	//   * Field removals/additions
	//
	// State is passed through UNCHANGED - provider will migrate it when user runs terraform apply.

	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	if !attrs.Exists() {
		// No attributes - return as-is
		return result, nil
	}

	// ROUTING LOGIC - Determine target resource type for moved blocks
	// This logic is still needed to generate correct moved blocks in TransformConfig
	defaultAttr := attrs.Get("default")
	matchAttr := attrs.Get("match")
	precedenceAttr := attrs.Get("precedence")

	// Check if default is explicitly set to true
	isExplicitDefault := defaultAttr.Exists() && defaultAttr.Bool()

	// If default=true explicitly, it's a default profile (even if match/precedence present)
	// Otherwise, if it has match AND precedence, it's a custom profile
	isCustomProfile := !isExplicitDefault && matchAttr.Exists() && precedenceAttr.Exists()

	// Store the determined type in StateTypeRenames for the pipeline to apply
	// This tells TransformConfig which resource type to use in moved blocks
	var newResourceType string
	if isCustomProfile {
		newResourceType = m.newTypeCustom
	} else {
		newResourceType = m.newTypeDefault
	}

	// Initialize StateTypeRenames map if needed
	if ctx.StateTypeRenames == nil {
		ctx.StateTypeRenames = make(map[string]interface{})
	}

	// Store the type rename using the format expected by state_transform.go
	// The key format is "resourceType.resourceName"
	// We need to store for BOTH old resource type names since we don't know which one is being used
	stateTypeRenameKey1 := fmt.Sprintf("%s.%s", m.oldType, resourceName)
	stateTypeRenameKey2 := fmt.Sprintf("%s.%s", m.oldTypeDeprecated, resourceName)
	ctx.StateTypeRenames[stateTypeRenameKey1] = newResourceType
	ctx.StateTypeRenames[stateTypeRenameKey2] = newResourceType

	// Return state UNCHANGED - provider will handle all transformations
	return result, nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
// When true, tf-migrate will not perform state transformation - the provider handles it via MoveState + UpgradeState
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

