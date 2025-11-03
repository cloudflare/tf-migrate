package managed_transforms

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of Managed Transforms resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_managed_transforms", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for Managed Transforms
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Check if managed_request_headers exists
	requestHeaders := body.GetAttribute("managed_request_headers")
	if requestHeaders == nil {
		// Add empty list for managed_request_headers
		body.SetAttributeValue("managed_request_headers", cty.ListValEmpty(cty.DynamicPseudoType))
	}

	// Check if managed_response_headers exists
	responseHeaders := body.GetAttribute("managed_response_headers")
	if responseHeaders == nil {
		// Add empty list for managed_response_headers
		body.SetAttributeValue("managed_response_headers", cty.ListValEmpty(cty.DynamicPseudoType))
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformations needed for Managed Transforms
	return stateJSON.String(), nil
}
