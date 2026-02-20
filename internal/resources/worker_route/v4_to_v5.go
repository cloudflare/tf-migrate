package worker_route

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both v4 resource names (plural and singular forms)
	internal.RegisterMigrator("cloudflare_workers_route", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_worker_route", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (plural form)
	return "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both v4 resource names (plural and singular)
	return resourceType == "cloudflare_workers_route" || resourceType == "cloudflare_worker_route"
}

// GetResourceRename implements the ResourceRenamer interface
// Handles both cloudflare_worker_route (singular) and cloudflare_workers_route (plural)
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_worker_route", "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for this simple migration
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Check if this is the singular form (needs moved block)
	wasSingular := block.Type() == "resource" && len(block.Labels()) > 0 && block.Labels()[0] == "cloudflare_worker_route"

	// Handle resource rename: cloudflare_worker_route → cloudflare_workers_route
	tfhcl.RenameResourceType(block, "cloudflare_worker_route", "cloudflare_workers_route")

	// Rename field: script_name → script
	tfhcl.RenameAttribute(body, "script_name", "script")

	// Generate moved block if the resource was renamed (singular → plural)
	if wasSingular {
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
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

// TransformState is a no-op for workers_route migration.
// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
