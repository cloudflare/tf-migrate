package access_application

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both old and new names
	internal.RegisterMigrator("cloudflare_access_application", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_application", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_access_application" ||
		resourceType == "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Transform toset(...) to list [...] for allowed_idps and custom_pages
	tosetPattern := regexp.MustCompile(`toset\s*\(([^)]+)\)`)
	content = tosetPattern.ReplaceAllString(content, "$1")

	// Update old policy resource references: cloudflare_access_policy -> cloudflare_zero_trust_access_policy
	content = strings.ReplaceAll(content, "cloudflare_access_policy.", "cloudflare_zero_trust_access_policy.")

	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename cloudflare_access_application to cloudflare_zero_trust_access_application (if needed)
	if len(block.Labels()) >= 1 && block.Labels()[0] == "cloudflare_access_application" {
		tfhcl.RenameResourceType(block, "cloudflare_access_application", "cloudflare_zero_trust_access_application")
	}

	body := block.Body()

	// STEP 1: Add default type attribute if missing (v4 defaulted to "self_hosted")
	m.addDefaultTypeAttribute(body)

	// STEP 2: Remove unsupported attributes
	m.removeUnsupportedAttributes(body)

	// STEP 3: Convert destinations blocks to list attribute
	m.convertDestinationsBlocksToAttribute(body)

	// STEP 4: Transform policies attribute
	m.transformPoliciesAttribute(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) addDefaultTypeAttribute(body *hclwrite.Body) {
	// Check if type attribute exists
	typeAttr := body.GetAttribute("type")
	if typeAttr == nil {
		// Add default type = "self_hosted" as this was the v4 default
		body.SetAttributeValue("type", cty.StringVal("self_hosted"))
	}
}

func (m *V4ToV5Migrator) removeUnsupportedAttributes(body *hclwrite.Body) {
	// Remove domain_type attribute - not supported in v5
	body.RemoveAttribute("domain_type")

	// Remove skip_app_launcher_login_page if type is not "app_launcher"
	typeAttr := body.GetAttribute("type")
	if typeAttr != nil {
		// Get the type value to check if it's "app_launcher"
		tokens := typeAttr.Expr().BuildTokens(nil)
		var typeValue string
		for _, token := range tokens {
			typeValue += string(token.Bytes)
		}

		// Remove quotes if present and check the value
		typeValue = strings.Trim(typeValue, `"`)
		if typeValue != "app_launcher" {
			// Remove skip_app_launcher_login_page attribute when type is not app_launcher
			body.RemoveAttribute("skip_app_launcher_login_page")
		}
	} else {
		// If no type attribute, remove skip_app_launcher_login_page as it would be invalid
		body.RemoveAttribute("skip_app_launcher_login_page")
	}
}

func (m *V4ToV5Migrator) transformPoliciesAttribute(body *hclwrite.Body) {
	policiesAttr := body.GetAttribute("policies")
	if policiesAttr == nil {
		return
	}

	// Parse the expression to check its type
	tokens := policiesAttr.Expr().BuildTokens(nil)
	tokensStr := ""
	for _, token := range tokens {
		tokensStr += string(token.Bytes)
	}

	// If it's empty array or complex expression (concat, for, etc.), leave it as is
	tokensStr = strings.TrimSpace(tokensStr)
	if tokensStr == "[]" || strings.Contains(tokensStr, "concat(") || strings.Contains(tokensStr, "[for") {
		return
	}

	// Parse as HCL to extract list elements
	testCode := "test = " + tokensStr
	syntaxFile, syntaxDiags := hclsyntax.ParseConfig([]byte(testCode), "", hcl.InitialPos)
	if syntaxDiags.HasErrors() {
		return
	}

	// Get the tuple expression
	var tupleExpr *hclsyntax.TupleConsExpr
	for _, bodyAttr := range syntaxFile.Body.(*hclsyntax.Body).Attributes {
		if bodyAttr.Name == "test" {
			if tup, ok := bodyAttr.Expr.(*hclsyntax.TupleConsExpr); ok {
				tupleExpr = tup
				break
			}
		}
	}

	if tupleExpr == nil || len(tupleExpr.Exprs) == 0 {
		return
	}

	// Build wrapped objects by reconstructing each element
	var newObjectStrs []string
	originalContent := []byte(testCode)
	for _, elem := range tupleExpr.Exprs {
		// Get the source bytes for this element
		elemRange := elem.Range()
		elemBytes := elemRange.SliceBytes(originalContent)
		elemStr := strings.TrimSpace(string(elemBytes))

		newObjectStrs = append(newObjectStrs, fmt.Sprintf("{ id = %s }", elemStr))
	}

	// Build the new policies HCL and parse it
	newPoliciesHCL := fmt.Sprintf("policies = [%s]", strings.Join(newObjectStrs, ", "))
	newFile, newDiags := hclwrite.ParseConfig([]byte(newPoliciesHCL), "", hcl.InitialPos)
	if newDiags.HasErrors() {
		return
	}

	newAttr := newFile.Body().GetAttribute("policies")
	if newAttr != nil {
		body.SetAttributeRaw("policies", newAttr.Expr().BuildTokens(nil))
	}
}

func (m *V4ToV5Migrator) convertDestinationsBlocksToAttribute(body *hclwrite.Body) {
	// Find all destinations blocks
	var destBlocks []*hclwrite.Block
	for _, childBlock := range body.Blocks() {
		if childBlock.Type() == "destinations" {
			destBlocks = append(destBlocks, childBlock)
		}
	}

	if len(destBlocks) == 0 {
		return // No destinations blocks to convert
	}

	// Build HCL string for the destinations list attribute
	var destObjects []string
	for _, destBlock := range destBlocks {
		// Convert each block to an object representation
		objectStr := m.convertDestinationBlockToObject(destBlock)
		if objectStr != "" {
			destObjects = append(destObjects, objectStr)
		}
	}

	if len(destObjects) > 0 {
		// Create the destinations attribute HCL
		destAttrHCL := fmt.Sprintf("destinations = [\n  %s\n]", strings.Join(destObjects, ",\n  "))

		// Parse the HCL and extract the attribute
		file, parseDiags := hclwrite.ParseConfig([]byte(destAttrHCL), "", hcl.InitialPos)
		if parseDiags.HasErrors() {
			return
		}

		// Find the destinations attribute in the parsed file and copy it
		for name, attr := range file.Body().Attributes() {
			if name == "destinations" {
				body.SetAttributeRaw("destinations", attr.Expr().BuildTokens(nil))
				break
			}
		}
	}

	// Remove all destinations blocks after converting to attribute
	for _, destBlock := range destBlocks {
		body.RemoveBlock(destBlock)
	}
}

func (m *V4ToV5Migrator) convertDestinationBlockToObject(block *hclwrite.Block) string {
	if block == nil {
		return ""
	}

	// Get all attributes from the block
	attrs := block.Body().Attributes()
	if len(attrs) == 0 {
		return "{}" // Empty object
	}

	// Build attribute strings
	var attrStrings []string
	var attrNames []string
	for name := range attrs {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)

	for _, name := range attrNames {
		attr := attrs[name]
		// Get the raw tokens for the expression to preserve references and formatting
		tokens := attr.Expr().BuildTokens(nil)
		var exprStr string
		for _, token := range tokens {
			exprStr += string(token.Bytes)
		}
		attrStrings = append(attrStrings, fmt.Sprintf("    %s = %s", name, exprStr))
	}

	return fmt.Sprintf("{\n%s\n  }", strings.Join(attrStrings, "\n"))
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformation needed
	return stateJSON.String(), nil
}
