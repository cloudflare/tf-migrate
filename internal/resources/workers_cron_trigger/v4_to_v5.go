package workers_cron_trigger

import (
	"regexp"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of Workers Cron Trigger resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the old (singular) name
	internal.RegisterMigrator("cloudflare_worker_cron_trigger", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the new (plural) resource type name
	return "cloudflare_workers_cron_trigger"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both old (singular) and new (plural) names
	return resourceType == "cloudflare_worker_cron_trigger" || resourceType == "cloudflare_workers_cron_trigger"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Rename cloudflare_worker_cron_trigger to cloudflare_workers_cron_trigger
	// Match the resource declaration and rename it
	pattern := regexp.MustCompile(`(resource\s+)"cloudflare_worker_cron_trigger"`)
	content = pattern.ReplaceAllString(content, `${1}"cloudflare_workers_cron_trigger"`)
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No config transformation needed - preprocessing handled the rename
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformation needed - the framework automatically updates
	// the resource type based on GetResourceType()
	return stateJSON.String(), nil
}
