package api_token_permission_groups

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the cloudflare_api_token_permission_groups data source.
//
// In v4, this data source returns a map of permission names to IDs (e.g.
// data.cloudflare_api_token_permission_groups.all.permissions["DNS Read"]).
//
// In v5, this data source was replaced by cloudflare_api_token_permission_groups_list,
// which returns a list of objects with id/name/scopes fields. The migrator renames
// the block so that the data source is still available for reference resolution.
//
// Any expression references in cloudflare_api_token.permission_groups that use the
// old map-style lookups are rewritten by the api_token resource migrator to use
// for-expressions against the new list-style data source.
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for the
// cloudflare_api_token_permission_groups datasource (v4 → v5).
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("data.cloudflare_api_token_permission_groups", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_api_token_permission_groups"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "data.cloudflare_api_token_permission_groups"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig renames the data source block from
// cloudflare_api_token_permission_groups to cloudflare_api_token_permission_groups_list.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	tfhcl.RenameResourceType(block, "cloudflare_api_token_permission_groups", "cloudflare_api_token_permission_groups_list")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
