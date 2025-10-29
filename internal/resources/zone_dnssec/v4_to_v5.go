package zone_dnssec

import (
	"time"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of cloudflare_zone_dnssec from v4 to v5.
// This migration requires converting flags and key_tag from TypeInt (v4) to Float64 (v5).
type V4ToV5Migrator struct {
}

// init registers the migrator on package initialization
func init() {
	NewV4ToV5Migrator()
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
					if statusValue == "active" || statusValue == "disabled" {
						// Add status attribute to config using the value from state
						tokens := hcl.TokensForSimpleValue(statusValue)
						if tokens != nil {
							body.SetAttributeRaw("status", tokens)
						}
					} else if statusValue == "pending" {
						body.SetAttributeValue("status", cty.StringVal("active"))
					} else if statusValue == "pending-disabled" {
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
// Converts flags and key_tag from TypeInt (v4) to Float64 (v5).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// This function can receive either:
	// 1. A full state document (in unit tests)
	// 2. A single resource instance (in actual migration framework)
	// We need to handle both cases

	result := stateJSON.String()

	// Check if this is a full state document (has "resources" key) or a single instance
	if stateJSON.Get("resources").Exists() {
		// Full state document - transform all cloudflare_zone_dnssec resources
		return m.transformFullState(result, stateJSON)
	}

	// Single instance - check if it's a valid zone_dnssec instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")
	if !attrs.Get("zone_id").Exists() {
		return result, nil
	}

	// Transform the single instance
	result = m.transformSingleInstance(result, stateJSON)

	return result, nil
}

// transformFullState handles transformation of a full state document
func (m *V4ToV5Migrator) transformFullState(result string, stateJSON gjson.Result) (string, error) {
	// Process all resources in the state
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		// Check if this is a zone_dnssec resource we need to migrate
		if !m.CanHandle(resourceType) {
			return true // continue
		}

		// Process each instance
		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			// Transform the instance attributes in place
			attrs := instance.Get("attributes")
			if attrs.Exists() && attrs.Get("zone_id").Exists() {
				// Get the instance JSON string
				instJSON := instance.String()
				// Transform it
				transformedInst := m.transformSingleInstance(instJSON, instance)
				// Parse the transformed instance
				transformedInstParsed := gjson.Parse(transformedInst)
				// Update the result with the transformed instance
				result, _ = sjson.SetRaw(result, instPath, transformedInstParsed.Raw)
			}
			return true
		})

		return true
	})

	return result, nil
}

// transformSingleInstance transforms a single zone_dnssec instance
func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	attrs := instance.Get("attributes")

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
		result = transformModifiedOnFormat(result, modifiedOn.String())
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

	return result
}

// transformModifiedOnFormat converts modified_on from v4 format to RFC3339
func transformModifiedOnFormat(result, modifiedOn string) string {
	// Try to parse the v4 format (RFC1123Z)
	t, err := time.Parse(time.RFC1123Z, modifiedOn)
	if err != nil {
		// If parsing fails, try RFC1123 without timezone
		t, err = time.Parse(time.RFC1123, modifiedOn)
		if err != nil {
			// If still fails, keep original value
			return result
		}
	}

	// Convert to RFC3339 format
	rfc3339 := t.Format(time.RFC3339)
	result, _ = sjson.Set(result, "attributes.modified_on", rfc3339)

	return result
}
