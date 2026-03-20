package zero_trust_access_service_token

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/hashicorp/hcl/v2/hclwrite"

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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_access_service_token", "cloudflare_zero_trust_access_service_token"}, "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original type at the START
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	if originalResourceType == "cloudflare_access_service_token" {
		tfhcl.RenameResourceType(block, "cloudflare_access_service_token", "cloudflare_zero_trust_access_service_token")
	}

	body := block.Body()

	// Remove deprecated field: min_days_for_renewal
	tfhcl.RemoveAttributes(body, "min_days_for_renewal")

	// Generate moved block only if resource type was renamed
	_, newType := m.GetResourceRename()
	if originalResourceType != newType {
		from := originalResourceType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

