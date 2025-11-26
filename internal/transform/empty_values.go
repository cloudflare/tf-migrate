// Package transform provides utilities for transforming Terraform configurations and state
package transform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// TransformEmptyValuesToNullOptions contains options for TransformEmptyValuesToNull
type TransformEmptyValuesToNullOptions struct {
	// Context for accessing HCL files and diagnostics
	Ctx *Context
	// The state JSON string being transformed
	Result string
	// JSON path to the object/fields (e.g., "attributes" for top-level, "attributes.input" for nested)
	FieldPath string
	// Parsed gjson result for the field (can be object or attributes)
	FieldResult gjson.Result
	// Resource name to match in HCL
	ResourceName string
	// HCL attribute path (empty string for top-level, "input" for nested in input block)
	HCLAttributePath string
	// Function to check if a resource type can be handled by this migrator
	CanHandle func(string) bool
}

// TransformEmptyValuesToNull transforms empty values to null in state, but only if they were not
// explicitly set in the HCL configuration. This handles the v4→v5 migration where v4 sets empty
// strings for optional fields while v5 uses null.
//
// This is a common pattern needed when:
//   - v4 provider sets fields to empty string ("") when not configured
//   - v5 provider sets fields to null when not configured
//   - We need to transform the state to match v5 behavior
//   - But preserve explicitly configured empty strings in the config
//
// Example usage in logpush_job (top-level fields):
//
//	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
//	    Ctx:              ctx,
//	    Result:           result,
//	    FieldPath:        "attributes",
//	    FieldResult:      attrs,
//	    ResourceName:     resourceName,
//	    HCLAttributePath: "",
//	    CanHandle:        m.CanHandle,
//	})
//
// Example usage in zero_trust_device_posture_rule (nested input fields):
//
//	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
//	    Ctx:              ctx,
//	    Result:           result,
//	    FieldPath:        "attributes.input",
//	    FieldResult:      inputField,
//	    ResourceName:     resourceName,
//	    HCLAttributePath: "input",
//	    CanHandle:        m.CanHandle,
//	})
func TransformEmptyValuesToNull(opts TransformEmptyValuesToNullOptions) string {
	if !opts.FieldResult.Exists() {
		return opts.Result
	}

	result := opts.Result

	// Iterate over each field in the object
	opts.FieldResult.ForEach(func(key, value gjson.Result) bool {
		fieldName := key.String()

		// Check if this is an empty value
		if !state.IsEmptyValue(value) {
			return true // continue iteration
		}

		// Check if this empty value was explicitly defined in HCL
		emptyValueDefinedInHCL := false

		if len(opts.Ctx.CFGFiles) > 0 {
		HCL_SEARCH:
			for _, file := range opts.Ctx.CFGFiles {
				resourceBlocks := tfhcl.FindBlocksByType(file.Body(), "resource")
				for _, resourceBlock := range resourceBlocks {
					resourceBlockType := tfhcl.GetResourceType(resourceBlock)
					resourceBlockName := tfhcl.GetResourceName(resourceBlock)

					// Check if this is the resource we're transforming
					if opts.CanHandle(resourceBlockType) && resourceBlockName == opts.ResourceName {
						// Check if the field is defined in HCL config
						if opts.HCLAttributePath == "" {
							// Top-level field - check directly in resource body
							if resourceBlock.Body().GetAttribute(fieldName) != nil {
								emptyValueDefinedInHCL = true
							}
						} else {
							// Nested field - check in the specified attribute
							if attr := resourceBlock.Body().GetAttribute(opts.HCLAttributePath); attr != nil {
								if tfhcl.AttributeValueContainsKey(attr, fieldName) {
									emptyValueDefinedInHCL = true
								}
							}
						}
						break HCL_SEARCH
					}
				}
			}
		}

		// If empty value was not explicitly defined in HCL, transform it to null
		if !emptyValueDefinedInHCL {
			result, _ = sjson.Set(result, opts.FieldPath+"."+fieldName, nil)
			opts.Ctx.Diagnostics = append(opts.Ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Transforming state for attribute %s from empty value to null. Will require an update in place.", fieldName),
			})
		}

		return true // continue iteration
	})

	return result
}
