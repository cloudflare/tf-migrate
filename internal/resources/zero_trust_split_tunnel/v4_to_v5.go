package zero_trust_split_tunnel

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	tfstate "github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_split_tunnel", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_split_tunnel", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the default v5 resource type (for default policy case)
// Note: Custom policy case uses cloudflare_zero_trust_device_custom_profile instead
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_default_profile"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_split_tunnel" || resourceType == "cloudflare_zero_trust_split_tunnel"
}

// TODO
// GetResourceRename returns empty since this is not a simple rename
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "", ""
}

// Preprocess - no preprocessing needed
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	policyIDAttr := body.GetAttribute("policy_id")
	policyID := tfhcl.ExtractStringFromAttribute(policyIDAttr)

	if policyID == "" || policyID == "default" || policyID == "null" {
		return m.TransformConfigToDefaultPolicy(ctx, block)
	} else {
		return m.TransformConfigToCustomPolicy(ctx, block)
	}
}

func (m *V4ToV5Migrator) TransformConfigToDefaultPolicy(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	tfhcl.RenameResourceType(block, "cloudflare_split_tunnel", "cloudflare_zero_trust_device_default_profile")
	tfhcl.RenameResourceType(block, "cloudflare_zero_trust_split_tunnel", "cloudflare_zero_trust_device_default_profile")

	body := block.Body()

	// Extract mode to determine target attribute name
	mode := tfhcl.ExtractStringFromAttribute(body.GetAttribute("mode"))

	// Default to "exclude" if mode is not set (shouldn't happen but be defensive)
	if mode == "" {
		mode = "exclude"
	}

	tfhcl.RemoveAttributes(body, "mode", "policy_id")

	// Convert tunnels blocks to include/exclude array attribute
	convertBlocksToArrayWithFormatting(body, "tunnels", mode, true)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// convertBlocksToArrayWithFormatting converts blocks to an array attribute with specific formatting:
// - Opening bracket on same line
// - Each object on new line
// - Closing bracket on new line
// This matches the expected v5 terraform formatting for list attributes.
func convertBlocksToArrayWithFormatting(body *hclwrite.Body, blockType, targetAttrName string, emptyIfNone bool) {
	blocks := tfhcl.FindBlocksByType(body, blockType)

	if len(blocks) == 0 {
		if emptyIfNone {
			body.SetAttributeRaw(targetAttrName, tfhcl.TokensForEmptyArray())
		}
		return
	}

	var tokens hclwrite.Tokens

	// Opening bracket
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte("["),
	})

	// Add each object with proper formatting
	for i, block := range blocks {
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})

		objTokens := tfhcl.BuildObjectFromBlock(block)
		tokens = append(tokens, objTokens...)

		// Comma separator (except after last element)
		if i < len(blocks)-1 {
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenComma,
				Bytes: []byte(","),
			})
		}
	}

	// Closing bracket
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrack,
		Bytes: []byte("]"),
	})

	body.SetAttributeRaw(targetAttrName, tokens)

	// Remove original blocks
	tfhcl.RemoveBlocksByType(body, blockType)
}

func (m *V4ToV5Migrator) TransformConfigToCustomPolicy(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// TODO: Implement custom policy transformation
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	// Extract the original resource type from the state using the resourcePath (if available)
	// resourcePath format is "resources.X.instances.Y.attributes"
	// We need to get the type at "resources.X.type"
	var originalType string
	if resourcePath != "" {
		parts := strings.Split(resourcePath, ".")
		if len(parts) >= 2 {
			resourceIndex := parts[1]
			resourceTypePath := fmt.Sprintf("resources.%s.type", resourceIndex)
			originalType = gjson.Get(ctx.StateJSON, resourceTypePath).String()
		}
	}

	// If we couldn't extract the original type, try both known types
	// This handles test scenarios where resourcePath is empty
	if originalType == "" {
		// Default to the deprecated name for SetStateTypeRename
		// In unit tests this won't matter since the test framework doesn't use it
		originalType = "cloudflare_split_tunnel"
	}

	// Get policy_id to determine which transformation to use
	attrs := instance.Get("attributes")
	policyID := attrs.Get("policy_id").String()

	if policyID == "" || policyID == "default" {
		return m.transformStateToDefaultPolicy(ctx, instance, originalType, resourceName)
	} else {
		return m.transformStateToCustomPolicy(ctx, instance, originalType, resourceName)
	}
}

func (m *V4ToV5Migrator) transformStateToDefaultPolicy(ctx *transform.Context, instance gjson.Result, originalType, resourceName string) (string, error) {
	result := instance.String()

	if !instance.Exists() || !instance.Get("attributes").Exists() {
		return result, nil
	}

	attrs := instance.Get("attributes")

	// Extract mode to determine target attribute name
	mode := attrs.Get("mode").String()
	if mode == "" {
		mode = "exclude"
	}

	// Rename tunnels to include or exclude based on mode
	result = tfstate.RenameField(result, "attributes", attrs, "tunnels", mode)

	// Remove mode and policy_id fields
	result = tfstate.RemoveFields(result, "attributes", attrs, "mode", "policy_id")

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Register the resource type rename
	targetType := "cloudflare_zero_trust_device_default_profile"
	transform.SetStateTypeRename(ctx, resourceName, originalType, targetType)

	return result, nil
}

func (m *V4ToV5Migrator) transformStateToCustomPolicy(ctx *transform.Context, instance gjson.Result, originalType, resourceName string) (string, error) {
	// TODO: Implement custom policy state transformation
	result := instance.String()
	result, _ = sjson.Set(result, "schema_version", 0)
	return result, nil
}
