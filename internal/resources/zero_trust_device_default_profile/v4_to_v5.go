package zero_trust_device_default_profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_split_tunnel"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of zero trust device profile resources from v4 to v5
// This migrator routes to either default or custom profile based on match/precedence presence
type V4ToV5Migrator struct {
	oldType             string
	oldTypeDeprecated   string
	newTypeDefault      string
	newTypeCustom       string
	lastTransformedType string
}

// findStateFile searches upward from the given directory to find terraform.tfstate
// Returns empty string if not found
func findStateFile(startDir string) string {
	dir := startDir
	for {
		stateFile := filepath.Join(dir, "terraform.tfstate")
		if _, err := os.Stat(stateFile); err == nil {
			return stateFile
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}
	return ""
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType:             "cloudflare_zero_trust_device_profiles",
		oldTypeDeprecated:   "cloudflare_device_settings_policy",
		newTypeDefault:      "cloudflare_zero_trust_device_default_profile",
		newTypeCustom:       "cloudflare_zero_trust_device_custom_profile",
		lastTransformedType: "",
	}

	// Register BOTH old resource names - migrator will route based on match/precedence
	internal.RegisterMigrator("cloudflare_zero_trust_device_profiles", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_device_settings_policy", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the type used in the last transformation
	// If not set, default to default profile
	if m.lastTransformedType != "" {
		return m.lastTransformedType
	}
	return m.newTypeDefault
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
	// Reset lastTransformedType at start of each transformation to avoid test interference
	m.lastTransformedType = ""

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
	m.lastTransformedType = newResourceType

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

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// Reset lastTransformedType at start of each transformation to avoid interference between resources
	m.lastTransformedType = ""

	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	if !attrs.Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Check if this is a custom profile (has match and precedence) or default profile
	// Priority: default field > presence of match+precedence
	defaultAttr := attrs.Get("default")
	matchAttr := attrs.Get("match")
	precedenceAttr := attrs.Get("precedence")

	// Check if default is explicitly set to true
	isExplicitDefault := defaultAttr.Exists() && defaultAttr.Bool()

	// If default=true explicitly, it's a default profile (even if match/precedence present)
	// Otherwise, if it has match AND precedence, it's a custom profile
	isCustomProfile := !isExplicitDefault && matchAttr.Exists() && precedenceAttr.Exists()

	// Set lastTransformedType for GetResourceType to use
	if isCustomProfile {
		m.lastTransformedType = m.newTypeCustom
	} else {
		m.lastTransformedType = m.newTypeDefault
	}

	// 2. Remove fields based on profile type
	if isCustomProfile {
		// Custom profile: remove only 'default' and 'enabled' fields
		result = state.RemoveFields(result, "attributes", attrs, "default", "enabled")
	} else {
		// Default profile: remove custom-only fields
		result = state.RemoveFields(result, "attributes", attrs,
			"name", "description", "match", "precedence", "enabled", "default")
	}

	// Re-parse attrs after removing fields to get updated structure
	attrs = gjson.Parse(result).Get("attributes")

	// 3. Remove fallback_domains if present (both profile types)
	// v4 allowed fallback_domains as nested attribute, v5 requires separate resource
	// Remove from state to prevent drift and API errors when trying to update
	// Users should migrate to cloudflare_zero_trust_device_default_profile_local_domain_fallback
	// or cloudflare_zero_trust_device_custom_profile_local_domain_fallback resources
	if attrs.Get("fallback_domains").Exists() {
		result, _ = sjson.Delete(result, "attributes.fallback_domains")
	}

	// 4. Remove empty exclude arrays (Optional+Computed field)
	// If exclude is an empty array in v4 state, remove it to let v5 API provide defaults
	if exclude := attrs.Get("exclude"); exclude.Exists() && exclude.IsArray() && len(exclude.Array()) == 0 {
		result, _ = sjson.Delete(result, "attributes.exclude")
	}

	// Re-parse attrs after removals to continue processing
	attrs = gjson.Parse(result).Get("attributes")

	// 5. Handle tunnel_protocol - preserve in state
	// v5 schema: Computed + Optional, but API returns "wireguard" as default
	// Preserve tunnel_protocol in state to match what API returns
	// This prevents drift when v5 provider refreshes from API

	// 6. Convert numeric types: Int → Float64
	if autoConnect := attrs.Get("auto_connect"); autoConnect.Exists() {
		result, _ = sjson.Set(result, "attributes.auto_connect", state.ConvertToFloat64(autoConnect))
	}

	if captivePortal := attrs.Get("captive_portal"); captivePortal.Exists() {
		result, _ = sjson.Set(result, "attributes.captive_portal", state.ConvertToFloat64(captivePortal))
	}

	// Custom profile: convert precedence Int → Float64
	if isCustomProfile {
		if precedence := attrs.Get("precedence"); precedence.Exists() {
			result, _ = sjson.Set(result, "attributes.precedence", state.ConvertToFloat64(precedence))
		}

		// Custom profile: set policy_id from the profile ID portion of the composite ID
		// v4 ID format: account_id/profile_id
		// v5 custom profile needs policy_id set to just the profile_id
		if id := attrs.Get("id"); id.Exists() {
			idStr := id.String()
			// Split on "/" to get the profile_id
			if slashIdx := strings.Index(idStr, "/"); slashIdx != -1 && slashIdx < len(idStr)-1 {
				policyID := idStr[slashIdx+1:]
				result, _ = sjson.Set(result, "attributes.policy_id", policyID)
			}
		}
	}

	// 7. Create service_mode_v2 nested object from flat fields
	result = m.createServiceModeV2(result, attrs)

	// 8. Always set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// updateDeviceProfileTypesInState updates all cloudflare_zero_trust_device_profiles resource types
// to their correct v5 types (default or custom) based on their attributes.
// This must be called before individual resource processing so GetResourceType() returns the correct type.
func (m *V4ToV5Migrator) updateDeviceProfileTypesInState(stateJSON string) string {
	result := stateJSON

	resources := gjson.Get(stateJSON, "resources")
	if !resources.Exists() || !resources.IsArray() {
		return result
	}

	for i, resource := range resources.Array() {
		resourceType := resource.Get("type").String()
		if resourceType != "cloudflare_zero_trust_device_profiles" {
			continue
		}

		// Check attributes to determine if this is a custom or default profile
		attrs := resource.Get("instances.0.attributes")
		if !attrs.Exists() {
			continue
		}

		defaultAttr := attrs.Get("default")
		matchAttr := attrs.Get("match")
		precedenceAttr := attrs.Get("precedence")

		isExplicitDefault := defaultAttr.Exists() && defaultAttr.Bool()
		isCustomProfile := !isExplicitDefault && matchAttr.Exists() && precedenceAttr.Exists()

		var newType string
		if isCustomProfile {
			newType = m.newTypeCustom
		} else {
			newType = m.newTypeDefault
		}

		// Update the resource type
		typePath := fmt.Sprintf("resources.%d.type", i)
		result, _ = sjson.Set(result, typePath, newType)
	}

	return result
}

// createServiceModeV2 creates the service_mode_v2 nested object from v4's flat fields
// Pattern from healthcheck migration (createTCPConfig)
//
// v4 structure:
//
//	service_mode_v2_mode: "warp"
//	service_mode_v2_port: 8080
//
// v5 structure:
//
//	service_mode_v2: {
//	  mode: "warp"
//	  port: 8080
//	}
func (m *V4ToV5Migrator) createServiceModeV2(stateJSON string, attrs gjson.Result) string {
	mode := attrs.Get("service_mode_v2_mode")
	port := attrs.Get("service_mode_v2_port")

	// Check if port has a meaningful value (not null/zero/empty)
	hasPort := port.Exists() && port.Value() != nil && port.Int() != 0

	// Handle v4 default: mode="warp" with no meaningful port value
	// Don't create service_mode_v2 if it's just the v4 default
	if mode.Exists() && mode.String() == "warp" && !hasPort {
		// Remove the mode field - it's just the v4 default
		stateJSON, _ = sjson.Delete(stateJSON, "attributes.service_mode_v2_mode")
		// Also remove port if it exists but is null/zero
		if port.Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, "attributes.service_mode_v2_port")
		}
		return stateJSON
	}

	// Only create nested object if we have at least one non-default field
	serviceMode := make(map[string]interface{})

	// Collect fields for nested object
	if mode.Exists() && mode.String() != "" {
		serviceMode["mode"] = mode.String()
	}
	if hasPort {
		// Convert Int → Float64
		serviceMode["port"] = state.ConvertToFloat64(port)
	}

	// Only create nested object if we have at least one field
	if len(serviceMode) > 0 {
		stateJSON, _ = sjson.Set(stateJSON, "attributes.service_mode_v2", serviceMode)

		// Remove the old flat fields
		if mode.Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, "attributes.service_mode_v2_mode")
		}
		if port.Exists() {
			stateJSON, _ = sjson.Delete(stateJSON, "attributes.service_mode_v2_port")
		}
	}

	return stateJSON
}
