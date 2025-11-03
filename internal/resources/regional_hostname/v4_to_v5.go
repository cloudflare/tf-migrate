package regional_hostname

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of Regional Hostname resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_regional_hostname", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_regional_hostname"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_regional_hostname"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for Regional Hostname
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Find and remove timeouts blocks
	var blocksToRemove []*hclwrite.Block
	for _, nestedBlock := range body.Blocks() {
		if nestedBlock.Type() == "timeouts" {
			blocksToRemove = append(blocksToRemove, nestedBlock)
		}
	}

	// Remove the timeouts blocks
	for _, blockToRemove := range blocksToRemove {
		body.RemoveBlock(blockToRemove)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformations needed for Regional Hostname
	return stateJSON.String(), nil
}
