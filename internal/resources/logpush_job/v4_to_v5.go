package logpush_job

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name - same as v5 in this case
	internal.RegisterMigrator("cloudflare_logpush_job", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name - same as v4
	return "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Convert output_options block to attribute (block → attribute syntax)
	// This handles: output_options { ... } → output_options = { ... }
	if outputBlock := tfhcl.FindBlockByType(body, "output_options"); outputBlock != nil {
		// Rename cve20214428 → cve_2021_44228 BEFORE conversion
		tfhcl.RenameAttribute(outputBlock.Body(), "cve20214428", "cve_2021_44228")
		tfhcl.ConvertSingleBlockToAttribute(body, "output_options", "output_options")
	}

	// 2. Handle kind = "instant-logs" → kind = ""
	// "instant-logs" is no longer valid in v5, convert to empty string
	if kindAttr := body.GetAttribute("kind"); kindAttr != nil {
		kindValue := tfhcl.ExtractStringFromAttribute(kindAttr)
		if kindValue == "instant-logs" {
			// Set to empty string (default in v5)
			tokens := hcl.TokensForSimpleValue("")
			if tokens != nil {
				body.SetAttributeRaw("kind", tokens)
			}
		}
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// 1. Convert integer fields to float64 for int64 compatibility
	result = m.convertNumericFields(result, attrs)

	// 2. Transform output_options array to object
	result = m.transformOutputOptions(result, attrs)

	// 3. Remove computed-only fields (these should not be in state)
	result = state.RemoveFields(result, "attributes", attrs,
		"error_message", "last_complete", "last_error")

	// 4. Handle kind value change: "instant-logs" → ""
	if kind := attrs.Get("kind"); kind.Exists() && kind.String() == "instant-logs" {
		result, _ = sjson.Set(result, "attributes.kind", "")
	}

	return result, nil
}

// convertNumericFields converts integer fields to float64 for int64 compatibility
func (m *V4ToV5Migrator) convertNumericFields(result string, attrs gjson.Result) string {
	numericFields := []string{
		"max_upload_bytes",
		"max_upload_records",
		"max_upload_interval_seconds",
	}

	for _, field := range numericFields {
		if val := attrs.Get(field); val.Exists() && val.Type == gjson.Number {
			// Convert to float64 for int64 compatibility
			result, _ = sjson.Set(result, "attributes."+field, val.Float())
		}
	}

	return result
}

// transformOutputOptions transforms output_options from array to object and renames fields
func (m *V4ToV5Migrator) transformOutputOptions(result string, attrs gjson.Result) string {
	outputOpts := attrs.Get("output_options")

	if !outputOpts.Exists() {
		return result
	}

	// Handle array → object transformation
	if outputOpts.IsArray() {
		array := outputOpts.Array()
		if len(array) == 0 {
			// Empty array → remove field
			result, _ = sjson.Delete(result, "attributes.output_options")
		} else {
			// Take first element and convert to object
			firstElem := array[0]
			obj := make(map[string]interface{})

			firstElem.ForEach(func(key, value gjson.Result) bool {
				k := key.String()

				// Rename cve20214428 → cve_2021_44228
				if k == "cve20214428" {
					k = "cve_2021_44228"
				}

				obj[k] = state.ConvertGjsonValue(value)
				return true
			})

			result, _ = sjson.Set(result, "attributes.output_options", obj)
		}
	} else if outputOpts.IsObject() {
		// Already an object, just rename field if needed
		result = state.RenameField(result, "attributes.output_options", outputOpts,
			"cve20214428", "cve_2021_44228")
	}

	return result
}
