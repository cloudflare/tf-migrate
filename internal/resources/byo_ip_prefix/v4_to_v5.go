package byo_ip_prefix

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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

	// Add warning comment for required v5 fields
	// User must manually add asn and cidr after migration
	warningMsg := "This resource requires manual intervention to add v5 required fields 'asn' and 'cidr'. Find values in Cloudflare Dashboard → Manage Account → IP Addresses → IP Prefixes. See migration documentation for details."
	tfhcl.AppendWarningComment(body, warningMsg)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(_ *transform.Context, stateJSON gjson.Result, _, _ string) (string, error) {
	// State transformation is handled by the v5 provider's UpgradeState (schema_version 0→1).
	// tf-migrate only handles HCL config transformation; pass state through unchanged.
	return stateJSON.String(), nil
}
