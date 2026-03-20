package custom_hostname_fallback_origin

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_custom_hostname_fallback_origin", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the v5 resource name (same as v4 in this case)
	return "cloudflare_custom_hostname_fallback_origin"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the v4 resource name
	return resourceType == "cloudflare_custom_hostname_fallback_origin"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_custom_hostname_fallback_origin"}, "cloudflare_custom_hostname_fallback_origin"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - fields are identical between v4 and v5
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - v4 and v5 configs are identical
	// Fields: zone_id, origin (both unchanged)
	// Computed fields (status, id, created_at, updated_at, errors) are not in config

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

