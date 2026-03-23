package managed_transforms

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the OLD (v4) resource name
	internal.RegisterMigrator("cloudflare_managed_headers", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_managed_headers"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_managed_headers"}, "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original resource type before any modifications (for moved block generation)
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_managed_headers", "cloudflare_managed_transforms")

	body := block.Body()

	// Convert managed_request_headers blocks to array attribute
	// Set empty array if no blocks found (since v5 requires this field)
	tfhcl.ConvertBlocksToArrayAttribute(body, "managed_request_headers", true)

	// Convert managed_response_headers blocks to array attribute
	// Set empty array if no blocks found (since v5 requires this field)
	tfhcl.ConvertBlocksToArrayAttribute(body, "managed_response_headers", true)

	// Generate moved block for state migration if resource type changed
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

