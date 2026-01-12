package zero_trust_device_default_profile

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zero trust device default profile resources from v4 to v5
// This migrator ONLY handles resources where default = true
// Resources with default = false should use the zero_trust_device_custom_profile migrator
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register BOTH old resource names that had default=true
	internal.RegisterMigrator("cloudflare_zero_trust_device_profiles", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_device_settings_policy", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_device_default_profile"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both v4 resource names that might have default=true
	return resourceType == "cloudflare_zero_trust_device_profiles" ||
		resourceType == "cloudflare_device_settings_policy" ||
		resourceType == "cloudflare_zero_trust_device_default_profile"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done at HCL level
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from old names to new name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_zero_trust_device_profiles", "cloudflare_zero_trust_device_default_profile"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// 1. Rename resource type
	// Handle both possible v4 names
	tfhcl.RenameResourceType(block, "cloudflare_zero_trust_device_profiles", "cloudflare_zero_trust_device_default_profile")
	tfhcl.RenameResourceType(block, "cloudflare_device_settings_policy", "cloudflare_zero_trust_device_default_profile")

	body := block.Body()

	// 2. Remove fields that don't exist in v5 default profile
	// These fields only exist in custom profiles
	tfhcl.RemoveAttributes(body, "name", "description", "match", "precedence", "enabled", "default")

	// 3. Handle service_mode_v2 default value mismatch
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

	// 4. Merge service_mode_v2_mode + service_mode_v2_port → service_mode_v2 nested object
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

	// 5. Handle tunnel_protocol - preserve in config
	// v5 schema: Computed + Optional with default ""
	// API behavior: returns "wireguard" as the actual computed value
	// If user explicitly set tunnel_protocol in v4, keep it in v5 to avoid drift
	// The v5 schema default ("") doesn't match API behavior ("wireguard"), so we must keep explicit values

	// 6. Add new v5-only fields with default values to prevent API errors
	// These fields didn't exist in v4, so we add them with sensible defaults
	// Users can customize these after migration if needed

	// Add empty exclude list if not present (prevents API 500 error)
	// Note: exclude and include are mutually exclusive in v5, so we only add exclude
	if !tfhcl.HasAttribute(body, "exclude") && !tfhcl.HasAttribute(body, "include") {
		body.SetAttributeRaw("exclude", tfhcl.TokensForEmptyArray())
	}

	// Add register_interface_ip_with_dns with API default value (true)
	tfhcl.EnsureAttribute(body, "register_interface_ip_with_dns", true)

	// Add sccm_vpn_boundary_support with default value
	tfhcl.EnsureAttribute(body, "sccm_vpn_boundary_support", false)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()
	attrs := stateJSON.Get("attributes")

	if !attrs.Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Remove fields that don't exist in v5 default profile
	result = state.RemoveFields(result, "attributes", attrs,
		"name", "description", "match", "precedence", "enabled", "default")

	// Re-parse attrs after removal to get updated structure
	attrs = gjson.Parse(result).Get("attributes")

	// 2. Handle tunnel_protocol - preserve in state
	// v5 schema: Computed + Optional, but API returns "wireguard" as default
	// Preserve tunnel_protocol in state to match what API returns
	// This prevents drift when v5 provider refreshes from API

	// Re-parse attrs to continue processing
	attrs = gjson.Parse(result).Get("attributes")

	// 3. Convert numeric types: Int → Float64
	if autoConnect := attrs.Get("auto_connect"); autoConnect.Exists() {
		result, _ = sjson.Set(result, "attributes.auto_connect", state.ConvertToFloat64(autoConnect))
	}

	if captivePortal := attrs.Get("captive_portal"); captivePortal.Exists() {
		result, _ = sjson.Set(result, "attributes.captive_portal", state.ConvertToFloat64(captivePortal))
	}

	// 4. Create service_mode_v2 nested object from flat fields
	result = m.createServiceModeV2(result, attrs)

	// 5. Always set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// createServiceModeV2 creates the service_mode_v2 nested object from v4's flat fields
// Pattern from healthcheck migration (createTCPConfig)
//
// v4 structure:
//   service_mode_v2_mode: "warp"
//   service_mode_v2_port: 8080
//
// v5 structure:
//   service_mode_v2: {
//     mode: "warp"
//     port: 8080
//   }
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
