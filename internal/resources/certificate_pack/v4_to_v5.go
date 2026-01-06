package certificate_pack

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of certificate_pack resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_certificate_pack", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "", ""
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// wait_for_active_status: removed in v5
	tfhcl.RemoveAttributes(body, "wait_for_active_status")

	// validation_records, validation_errors: were Optional+Computed in v4, only Computed in v5
	tfhcl.RemoveAttributes(body, "validation_records", "validation_errors")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	if !attrs.Get("zone_id").Exists() || !attrs.Get("type").Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	result = m.transformSingleInstance(result, stateJSON)

	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformSingleInstance transforms a single certificate pack instance
func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	attrs := instance.Get("attributes")

	// Convert validity_days from Int to Int64 (keep as number in JSON)
	if validityDays := attrs.Get("validity_days"); validityDays.Exists() {
		result, _ = sjson.Set(result, "attributes.validity_days", state.ConvertToFloat64(validityDays))
	}

	// Remove wait_for_active_status field (v4 only, not in v5)
	result = state.RemoveFields(result, "attributes", attrs, "wait_for_active_status")

	// Transform validation_records array - remove cname_target and cname_name from each item
	result = m.transformValidationRecords(result, attrs)

	// Note: validation_errors structure is compatible, no changes needed
	if !attrs.Get("validation_errors").Exists() || attrs.Get("validation_errors").Type == gjson.Null {
		result, _ = sjson.Set(result, "attributes.validation_errors", []interface{}{})
	}

	return result
}

// transformValidationRecords removes obsolete fields from validation_records
func (m *V4ToV5Migrator) transformValidationRecords(result string, attrs gjson.Result) string {
	validationRecords := attrs.Get("validation_records")

	if !attrs.Get("validation_records").Exists() || attrs.Get("validation_records").Type == gjson.Null {
		result, _ = sjson.Set(result, "attributes.validation_records", []interface{}{})
		return result
	}

	// Process each validation record to remove cname_target and cname_name
	records := validationRecords.Array()
	for i, record := range records {
		// Remove cname_target if it exists
		if record.Get("cname_target").Exists() {
			result, _ = sjson.Delete(result, fmt.Sprintf("attributes.validation_records.%d.cname_target", i))
		}
		// Remove cname_name if it exists
		if record.Get("cname_name").Exists() {
			result, _ = sjson.Delete(result, fmt.Sprintf("attributes.validation_records.%d.cname_name", i))
		}
	}

	return result
}
