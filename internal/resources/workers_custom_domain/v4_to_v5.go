package workers_custom_domain

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_worker_domain", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_custom_domain"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_worker_domain"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_worker_domain", "cloudflare_workers_custom_domain"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceName := tfhcl.GetResourceName(block)
	tfhcl.RenameResourceType(block, "cloudflare_worker_domain", "cloudflare_workers_custom_domain")

	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op because provider StateUpgrader/MoveState handles migration.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates this resource relies on provider state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
