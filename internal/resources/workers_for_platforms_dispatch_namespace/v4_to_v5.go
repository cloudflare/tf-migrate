package workers_for_platforms_dispatch_namespace

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// v4 had both cloudflare_workers_for_platforms_namespace (deprecated)
	// and cloudflare_workers_for_platforms_dispatch_namespace (current)
	// Both map to cloudflare_workers_for_platforms_dispatch_namespace in v5
	internal.RegisterMigrator("cloudflare_workers_for_platforms_namespace", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_workers_for_platforms_dispatch_namespace", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_for_platforms_dispatch_namespace"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_for_platforms_namespace" ||
		resourceType == "cloudflare_workers_for_platforms_dispatch_namespace"
}

// GetResourceRename implements the ResourceRenamer interface
// Maps the deprecated name to the current name
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_workers_for_platforms_namespace", "cloudflare_workers_for_platforms_dispatch_namespace"}, "cloudflare_workers_for_platforms_dispatch_namespace"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For workers_for_platforms_dispatch_namespace, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For cloudflare_workers_for_platforms_namespace (deprecated): renames the resource type
// and generates a `moved` block to trigger the provider's MoveState handler.
// For cloudflare_workers_for_platforms_dispatch_namespace: config is identical, no changes needed.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original name/type for moved block generation
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	if originalResourceType == "cloudflare_workers_for_platforms_namespace" {
		// Rename deprecated resource type to current name
		tfhcl.RenameResourceType(block, "cloudflare_workers_for_platforms_namespace", "cloudflare_workers_for_platforms_dispatch_namespace")

		// Generate moved block to trigger provider's MoveState handler (Terraform 1.8+)
		// This allows the provider's StateUpgraders to handle state transformation automatically
		_, newType := m.GetResourceRename()
		from := originalResourceType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	// cloudflare_workers_for_platforms_dispatch_namespace: config is identical between v4 and v5
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

