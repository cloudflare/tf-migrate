package origin_ca_certificate

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_origin_ca_certificate", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the v5 resource name (same as v4 - not renamed)
	return "cloudflare_origin_ca_certificate"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the v4 resource name
	return resourceType == "cloudflare_origin_ca_certificate"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_origin_ca_certificate", "cloudflare_origin_ca_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Config transformations:
	// - min_days_for_renewal: REMOVED in v5 (not supported)
	//
	// State transformations (handled by provider's StateUpgraders):
	// - hostnames: Set in v4 → List in v5
	// - requested_validity: Int64 in v4 → Float64 in v5 (default 5475)
	//
	// Unchanged fields:
	// - csr: String
	// - request_type: String
	// - Computed fields (id, certificate, expires_on): Not in config

	// Remove min_days_for_renewal attribute if present
	body := block.Body()
	body.RemoveAttribute("min_days_for_renewal")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// NO-OP: State transformation is handled by the provider's StateUpgraders.
	//
	// The provider implements UpgradeState which automatically transforms v4 state to v5
	// when Terraform detects a schema version mismatch (v4 schema_version=0 → v5 version=500).
	//
	// See: cloudflare-terraform-next/internal/services/origin_ca_certificate/migration/v500/
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
