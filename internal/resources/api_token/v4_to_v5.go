package api_token

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_api_token", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_api_token"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_api_token"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_api_token", "cloudflare_api_token"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	m.transformPolicyBlocks(body)
	m.transformConditionBlock(body)
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) transformPolicyBlocks(body *hclwrite.Body) {
	policyBlocks := tfhcl.FindBlocksByType(body, "policy")
	if len(policyBlocks) == 0 {
		return
	}

	var policyObjects []hclwrite.Tokens
	for _, policyBlock := range policyBlocks {
		objTokens := hcl.BuildObjectFromBlock(policyBlock)
		policyObjects = append(policyObjects, objTokens)
	}

	listTokens := hclwrite.TokensForTuple(policyObjects)
	body.SetAttributeRaw("policies", listTokens)

	tfhcl.RemoveBlocksByType(body, "policy")
}

func (m *V4ToV5Migrator) transformConditionBlock(body *hclwrite.Body) {
	conditionBlock := tfhcl.FindBlockByType(body, "condition")
	if conditionBlock == nil {
		return
	}

	var conditionAttrs []hclwrite.ObjectAttrTokens
	for _, nestedBlock := range conditionBlock.Body().Blocks() {
		if nestedBlock.Type() == "request_ip" {
			requestIPTokens := hcl.BuildObjectFromBlock(nestedBlock)
			conditionAttrs = append(conditionAttrs, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier("request_ip"),
				Value: requestIPTokens,
			})
		}
	}

	conditionTokens := hclwrite.TokensForObject(conditionAttrs)
	body.SetAttributeRaw("condition", conditionTokens)
	body.RemoveBlock(conditionBlock)
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	attributesPath := "attributes"
	attributes := stateJSON.Get(attributesPath)
	result = state.RenameField(result, attributesPath, attributes, "policy", "policies")

	// Transform condition from array to object (v4 has array with one element, v5 has object)
	conditionPath := "attributes.condition"
	conditionData := gjson.Get(result, conditionPath)

	if conditionData.Exists() && conditionData.IsArray() {
		conditionArray := conditionData.Array()
		if len(conditionArray) > 0 {
			result, _ = sjson.SetRaw(result, conditionPath, conditionArray[0].Raw)
		}
	}

	attributes = gjson.Get(result, attributesPath)
	result = state.EnsureField(result, attributesPath, attributes, "last_used_on", nil)

	return result, nil
}
