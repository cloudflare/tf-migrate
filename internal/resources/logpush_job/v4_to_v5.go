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
	return "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_logpush_job", "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Convert output_options block to attribute (block → attribute syntax)
	// This handles: output_options { ... } → output_options = { ... }
	if outputBlock := tfhcl.FindBlockByType(body, "output_options"); outputBlock != nil {
		outputBody := outputBlock.Body()

		// Rename cve20214428 → cve_2021_44228 BEFORE conversion
		tfhcl.RenameAttribute(outputBody, "cve20214428", "cve_2021_44228")

		// Add v4 schema defaults if not already present (to preserve v4 behavior in v5)
		// v5 does not have defaults for these fields, so we must make them explicit
		m.ensureV4SchemaDefaults(outputBody)

		tfhcl.ConvertSingleBlockToAttribute(body, "output_options", "output_options")
	}

	// 2. Handle kind = "instant-logs" → remove attribute
	// "instant-logs" is no longer valid in v5, remove the attribute entirely
	if kindAttr := body.GetAttribute("kind"); kindAttr != nil {
		kindValue := tfhcl.ExtractStringFromAttribute(kindAttr)
		if kindValue == "instant-logs" {
			// Remove the attribute entirely since instant-logs is not valid in v5
			body.RemoveAttribute("kind")
		}
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// ensureV4SchemaDefaults adds v4 schema defaults to output_options if not present
// This preserves v4 behavior in v5, which has no defaults for these fields
func (m *V4ToV5Migrator) ensureV4SchemaDefaults(body *hclwrite.Body) {
	// Use a slice to ensure deterministic ordering of defaults
	type defaultPair struct {
		field string
		value interface{}
	}

	v4Defaults := []defaultPair{
		{"field_delimiter", ","},
		{"record_prefix", "{"},
		{"record_suffix", "}\n"},
		{"timestamp_format", "unixnano"},
		{"sample_rate", 1.0},
	}

	for _, pair := range v4Defaults {
		if body.GetAttribute(pair.field) == nil {
			// Field not present, add the v4 default
			tokens := hcl.TokensForSimpleValue(pair.value)
			if tokens != nil {
				body.SetAttributeRaw(pair.field, tokens)
			}
		}
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// 1. Convert integer fields to float64 for int64 compatibility, remove 0 defaults
	result = m.convertNumericFields(result, attrs)

	// 2. Transform empty string defaults to null for v4 fields (filter, logpull_options, name)
	// v4 sets these to "" when not configured, v5 uses null
	// Only transform if not explicitly set to "" in config
	result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
		Ctx:              ctx,
		Result:           result,
		FieldPath:        "attributes",
		FieldResult:      attrs,
		ResourceName:     resourceName,
		HCLAttributePath: "",
		CanHandle:        m.CanHandle,
	})

	// 3. Transform output_options array to object
	result = m.transformOutputOptions(result, attrs)

	// 4. Remove computed-only fields (these should not be in state)
	result = state.RemoveFields(result, "attributes", attrs,
		"error_message", "last_complete", "last_error")

	// 5. Handle kind value change: "instant-logs" → remove attribute
	// "instant-logs" is no longer valid in v5, remove it entirely
	if kind := attrs.Get("kind"); kind.Exists() && kind.String() == "instant-logs" {
		result, _ = sjson.Delete(result, "attributes.kind")
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
		if val := attrs.Get(field); val.Exists() {
			// if val == 0, make it null in v5 to avoid a diff
			if val.Type == gjson.Number && val.Float() == 0 {
				result, _ = sjson.Delete(result, "attributes."+field)
			} else {
				// Convert to float64 for int64 compatibility
				result, _ = sjson.Set(result, "attributes."+field, state.ConvertToFloat64(val))
			}
		}
	}

	return result
}

// transformOutputOptions transforms output_options from array to object and renames fields
// Preserves v4 schema defaults in state to match config (v5 has no defaults for these fields)
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

				// Keep all values including v4 schema defaults (they're now in migrated config)
				obj[k] = state.ConvertGjsonValue(value)
				return true
			})

			// Add v4 schema defaults if not present (to match migrated config)
			m.addV4SchemaDefaultsToState(obj)

			result, _ = sjson.Set(result, "attributes.output_options", obj)
		}
	} else if outputOpts.IsObject() {
		// Already an object, just rename field if needed
		result = state.RenameField(result, "attributes.output_options", outputOpts,
			"cve20214428", "cve_2021_44228")

		// Add v4 schema defaults if not present (to match migrated config)
		// First, get the current object from result
		updatedOpts := gjson.Get(result, "attributes.output_options")
		if updatedOpts.Exists() {
			obj := make(map[string]interface{})
			updatedOpts.ForEach(func(key, value gjson.Result) bool {
				obj[key.String()] = state.ConvertGjsonValue(value)
				return true
			})
			m.addV4SchemaDefaultsToState(obj)
			result, _ = sjson.Set(result, "attributes.output_options", obj)
		}
	}

	return result
}

// addV4SchemaDefaultsToState adds v4 schema defaults to state object if not present
func (m *V4ToV5Migrator) addV4SchemaDefaultsToState(obj map[string]interface{}) {
	v4Defaults := map[string]interface{}{
		"field_delimiter":  ",",
		"record_prefix":    "{",
		"record_suffix":    "}\n",
		"timestamp_format": "unixnano",
		"sample_rate":      1.0,
	}

	for field, defaultValue := range v4Defaults {
		if _, exists := obj[field]; !exists {
			obj[field] = defaultValue
		}
	}
}
