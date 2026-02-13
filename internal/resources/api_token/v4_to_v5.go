package api_token

import (
	"regexp"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"
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
	// Remove deprecated data source cloudflare_api_token_permission_groups
	// It was replaced by cloudflare_api_token_permission_groups_list in v5
	// Remove just the data source line, preserving any comments
	re := regexp.MustCompile(`(?m)^data\s+"cloudflare_api_token_permission_groups"\s+"[^"]+"\s*\{\s*\}\s*\n`)
	content = re.ReplaceAllString(content, "")
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
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
		policyBody := policyBlock.Body()

		// Ensure effect attribute exists (required in v5, optional in v4)
		if policyBody.GetAttribute("effect") == nil {
			// Default to "allow" if not specified
			policyBody.SetAttributeValue("effect", cty.StringVal("allow"))
		}

		// Transform permission_groups from list of strings to list of objects with id field
		m.transformPermissionGroups(policyBody)

		// Transform resources from map to jsonencode() wrapped map
		// v4: resources = { "com.cloudflare.api.account.*" = "*" }
		// v5: resources = jsonencode({ "com.cloudflare.api.account.*" = "*" })
		m.transformResources(policyBody)

		objTokens := tfhcl.BuildObjectFromBlock(policyBlock)
		policyObjects = append(policyObjects, objTokens)
	}

	listTokens := hclwrite.TokensForTuple(policyObjects)
	body.SetAttributeRaw("policies", listTokens)

	tfhcl.RemoveBlocksByType(body, "policy")
}

// transformPermissionGroups converts permission_groups from list of strings to list of objects
// v4: permission_groups = ["id1", "id2"]
// v5: permission_groups = [{ id = "id1" }, { id = "id2" }]
func (m *V4ToV5Migrator) transformPermissionGroups(body *hclwrite.Body) {
	permGroupsAttr := body.GetAttribute("permission_groups")
	if permGroupsAttr == nil {
		return
	}

	// Parse the existing list expression to extract the permission IDs
	exprTokens := permGroupsAttr.Expr().BuildTokens(nil)

	// Build a list of objects where each string ID becomes { id = "..." }
	var permObjects []hclwrite.Tokens

	// We need to manually parse the tokens to extract string values
	// For simplicity, we'll reconstruct the structure
	inList := false
	var currentID string

	for _, token := range exprTokens {
		switch token.Type {
		case hclsyntax.TokenOBrack:
			inList = true
		case hclsyntax.TokenCBrack:
			inList = false
		case hclsyntax.TokenQuotedLit:
			if inList {
				// Extract the ID from the quoted literal (remove quotes)
				currentID = string(token.Bytes)
				// Create an object: { id = "currentID" }
				objAttrs := []hclwrite.ObjectAttrTokens{
					{
						Name:  hclwrite.TokensForIdentifier("id"),
						Value: hclwrite.TokensForValue(cty.StringVal(currentID)),
					},
				}
				permObjects = append(permObjects, hclwrite.TokensForObject(objAttrs))
			}
		}
	}

	// If we found any permission IDs, replace the attribute with the new format
	if len(permObjects) > 0 {
		body.RemoveAttribute("permission_groups")
		listTokens := hclwrite.TokensForTuple(permObjects)
		body.SetAttributeRaw("permission_groups", listTokens)
	}
}

// transformResources wraps the resources map with jsonencode()
// v4: resources = { "com.cloudflare.api.account.*" = "*" }
// v5: resources = jsonencode({ "com.cloudflare.api.account.*" = "*" })
func (m *V4ToV5Migrator) transformResources(body *hclwrite.Body) {
	resourcesAttr := body.GetAttribute("resources")
	if resourcesAttr == nil {
		return
	}

	// Get the existing expression tokens (the map value)
	exprTokens := resourcesAttr.Expr().BuildTokens(nil)

	// Build jsonencode( ... ) wrapper
	// Start with "jsonencode("
	var newTokens hclwrite.Tokens
	newTokens = append(newTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("jsonencode"),
	})
	newTokens = append(newTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOParen,
		Bytes: []byte("("),
	})

	// Add the original map expression
	newTokens = append(newTokens, exprTokens...)

	// Close with ")"
	newTokens = append(newTokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCParen,
		Bytes: []byte(")"),
	})

	// Replace the attribute with the wrapped version
	body.SetAttributeRaw("resources", newTokens)
}

func (m *V4ToV5Migrator) transformConditionBlock(body *hclwrite.Body) {
	conditionBlock := tfhcl.FindBlockByType(body, "condition")
	if conditionBlock == nil {
		return
	}

	var conditionAttrs []hclwrite.ObjectAttrTokens
	for _, nestedBlock := range conditionBlock.Body().Blocks() {
		if nestedBlock.Type() == "request_ip" {
			requestIPTokens := tfhcl.BuildObjectFromBlock(nestedBlock)
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

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (UpgradeState)
	// The provider automatically migrates state when users run terraform apply
	// This function is a no-op for api_token migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
