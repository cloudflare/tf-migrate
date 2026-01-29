package byo_ip_prefix

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of BYO IP prefix resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_byo_ip_prefix", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name stays the same in v5
	return "cloudflare_byo_ip_prefix"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_byo_ip_prefix"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for BYO IP prefix
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Remove v4-only fields that don't exist in v5
	// prefix_id - becomes 'id' in v5 (computed, not in config)
	// advertisement - replaced by 'advertised' in v5 (computed)
	tfhcl.RemoveAttributes(body, "prefix_id", "advertisement")

	// Note: We do NOT add asn and cidr here
	// These are required in v5 but don't exist in v4
	// User must manually add them after migration
	// v5 provider will fetch all computed fields on first refresh

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// This function receives a single instance and returns the transformed instance JSON
	result := instance.String()

	// Handle invalid/incomplete instances gracefully
	if !instance.Exists() || !instance.Get("attributes").Exists() {
		// Even for invalid instances, set schema_version to 0 for v5
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	attrs := instance.Get("attributes")

	// Set schema_version to 0 for v5 (MANDATORY)
	result, _ = sjson.Set(result, "schema_version", 0)

	// Rename prefix_id to id
	// In v4: prefix_id is the identifier
	// In v5: id is the identifier (computed field)
	result = state.RenameField(result, "attributes", attrs, "prefix_id", "id")

	// Remove v4-only field: advertisement
	// In v5, this is replaced by 'advertised' (computed boolean)
	result = state.RemoveFields(result, "attributes", attrs, "advertisement")

	// Note: We do NOT add asn, cidr, or other computed fields
	// The v5 provider will populate these on first refresh/plan
	// State after migration will be minimal: id, account_id, description
	// State after first refresh will be complete with all computed fields

	return result, nil
}
