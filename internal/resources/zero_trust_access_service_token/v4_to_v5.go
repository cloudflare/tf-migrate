package zero_trust_access_service_token

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

	// Deprecated v4 Name
	internal.RegisterMigrator("cloudflare_access_service_token", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_service_token", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the current name and the deprecated v4 name
	return resourceType == "cloudflare_access_service_token" || resourceType == "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_service_token", "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	if resourceType == "cloudflare_access_service_token" {
		tfhcl.RenameResourceType(block, "cloudflare_access_service_token", "cloudflare_zero_trust_access_service_token")
	}

	body := block.Body()

	// Remove deprecated field: min_days_for_renewal
	tfhcl.RemoveAttributes(body, "min_days_for_renewal")

	// Generate moved block for resource rename
	// This triggers the provider's MoveState handler (Terraform 1.8+)
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op for this resource.
// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
// The moved block generated in TransformConfig triggers the provider's migration logic.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders
	// The provider will handle:
	// - Resource rename (cloudflare_access_service_token → cloudflare_zero_trust_access_service_token)
	// - Field removal (min_days_for_renewal)
	// - Type conversion (client_secret_version: int → float64)
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider handles all state transformations via StateUpgraders.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
