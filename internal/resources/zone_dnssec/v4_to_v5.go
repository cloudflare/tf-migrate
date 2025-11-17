package zone_dnssec

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_zone_dnssec from v4 to v5.
// This migration requires converting flags and key_tag from TypeInt (v4) to Float64 (v5).
type V4ToV5Migrator struct {
}

// NewV4ToV5Migrator creates a new migrator for cloudflare_zone_dnssec v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register the migrator with the internal registry
	internal.RegisterMigrator("cloudflare_zone_dnssec", "v4", "v5", migrator)

	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone_dnssec"
}

// CanHandle determines if this migrator can handle the given resource type.
// Returns true for cloudflare_zone_dnssec (resource name is the same in v4 and v5).
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone_dnssec"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing is needed for this migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_zone_dnssec", "cloudflare_zone_dnssec"
}

// TransformConfig handles configuration file transformations.
// 1. Adds status attribute from state (changed from computed-only to optional in v5)
// 2. Removes modified_on attribute if present (changed from optional+computed to computed-only in v5)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Remove modified_on attribute if present (changed from optional+computed to computed-only)
	// RemoveAttribute is safe to call even if the attribute doesn't exist
	body.RemoveAttribute("modified_on")

	// Status changed from Computed (v4) to Optional (v5), so we need to preserve the current value
	// If there is no state value there is no value to preserve
	if ctx.StateJSON == "" {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// Get the resource name from the block labels (e.g., "example" in resource "cloudflare_zone_dnssec" "example")
	labels := block.Labels()
	if len(labels) >= 2 {
		resourceName := labels[1]

		// Parse state JSON and find this specific resource
		state := gjson.Parse(ctx.StateJSON)
		state.Get("resources").ForEach(func(key, resource gjson.Result) bool {
			// Match by resource type and name
			if resource.Get("type").String() == "cloudflare_zone_dnssec" &&
				resource.Get("name").String() == resourceName {
				// Get status from the first instance
				status := resource.Get("instances.0.attributes.status")
				statusValue := status.String()
				// Only add status if it's a valid v5 value ("active" or "disabled")
				// The v5 schema only accepts these two values, not "pending" or other intermediate states
				if status.Exists() && status.Type != gjson.Null && statusValue != "" {
					if statusValue == "active" || statusValue == "pending" {
						// Add status attribute to config using the value from state
						body.SetAttributeValue("status", cty.StringVal("active"))
					} else if statusValue == "disabled" || statusValue == "pending-disabled" {
						body.SetAttributeValue("status", cty.StringVal("disabled"))
					}
				}
				return false // stop iterating
			}
			return true // continue iterating
		})
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// This function receives a single resource instance and returns the transformed instance JSON.
// Converts flags and key_tag from TypeInt (v4) to Float64 (v5).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Check if it's a valid zone_dnssec instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")
	if !attrs.Get("zone_id").Exists() {
		return result, nil
	}

	// Convert flags from int to float64 if it exists
	flags := attrs.Get("flags")
	if flags.Exists() && flags.Type == gjson.Number {
		result, _ = sjson.Set(result, "attributes.flags", flags.Float())
	}

	// Convert key_tag from int to float64 if it exists
	keyTag := attrs.Get("key_tag")
	if keyTag.Exists() && keyTag.Type == gjson.Number {
		result, _ = sjson.Set(result, "attributes.key_tag", keyTag.Float())
	}

	// Convert modified_on from v4 format to RFC3339 if it exists
	// v4 format: "Tue, 04 Nov 2025 21:52:44 +0000"
	// v5 format (RFC3339): "2025-11-04T21:52:44Z"
	modifiedOn := attrs.Get("modified_on")
	if modifiedOn.Exists() && modifiedOn.Type == gjson.String && modifiedOn.String() != "" {
		result = transform.ConvertDateToRFC3339(result, "attributes.modified_on", modifiedOn.String())
	}
	// Handle status field: v5 only accepts "active" or "disabled"
	// If status is "pending" or any other invalid value, set it to null
	status := attrs.Get("status")
	if status.Exists() && status.Type == gjson.String {
		statusValue := status.String()
		if statusValue != "" && statusValue != "active" && statusValue != "disabled" {
			if statusValue == "pending" {
				result, _ = sjson.Set(result, "attributes.status", "active")
			} else if statusValue == "pending-disabled" {
				result, _ = sjson.Set(result, "attributes.status", "disabled")
			} else {
				// Set to null for invalid values
				// Use sjson.SetRaw to explicitly set JSON null
				result, _ = sjson.SetRaw(result, "attributes.status", "null")
			}
		}
	}

	return result, nil
}
