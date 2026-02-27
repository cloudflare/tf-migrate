package pages_domain

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_pages_domain", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the v5 resource name (same as v4 - no rename)
	return "cloudflare_pages_domain"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the v4 resource name
	return resourceType == "cloudflare_pages_domain"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for this simple migration
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_pages_domain", "cloudflare_pages_domain"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Single transformation: rename domain → name
	tfhcl.RenameAttribute(body, "domain", "name")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// The provider's migration logic (in internal/services/pages_domain/migration/v500/) handles:
	//   - Field rename: domain → name
	//   - Schema version: 0 → 500
	// This function is a no-op for pages_domain migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
// via StateUpgraders rather than tf-migrate's TransformState
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
