package api_shield

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

const (
	v4ResourceType = "cloudflare_api_shield"
	v5ResourceType = "cloudflare_api_shield"
)

// V4ToV5Migrator handles the migration of cloudflare_api_shield from v4 to v5.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_api_shield", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return v5ResourceType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == v4ResourceType || resourceType == v5ResourceType
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return v4ResourceType, v5ResourceType
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert auth_id_characteristics blocks to array attribute
	tfhcl.ConvertBlocksToAttributeList(body, "auth_id_characteristics", nil)
	if body.GetAttribute("auth_id_characteristics") == nil {
		body.SetAttributeRaw("auth_id_characteristics", tfhcl.TokensForEmptyArray())
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op for api_shield because state migration is handled by the provider.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}
