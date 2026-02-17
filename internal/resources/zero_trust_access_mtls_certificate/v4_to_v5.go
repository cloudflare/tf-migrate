package zero_trust_access_mtls_certificate

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register both the deprecated v4 name and the intermediate v5 name
	internal.RegisterMigrator("cloudflare_access_mutual_tls_certificate", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_mtls_certificate", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the current name and the deprecated v4 name
	return resourceType == "cloudflare_access_mutual_tls_certificate" ||
		resourceType == "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations can be done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_mutual_tls_certificate", "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture resource name and original type BEFORE any modifications
	resourceName := tfhcl.GetResourceName(block)
	originalType := block.Labels()[0]
	needsMovedBlock := originalType == "cloudflare_access_mutual_tls_certificate"

	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_access_mutual_tls_certificate", "cloudflare_zero_trust_access_mtls_certificate")

	// No other config transformations needed - all fields remain the same

	// Build result blocks
	resultBlocks := []*hclwrite.Block{block}

	// Generate moved block for state migration (only when renaming from old type)
	if needsMovedBlock {
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)
		resultBlocks = append(resultBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         resultBlocks,
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op - state transformation is handled by the provider's StateUpgraders.
// The moved block generated in TransformConfig triggers the provider's migration logic.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
