package zone_settings

import (
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_zone_settings_override", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone_settings_override"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone_settings_override"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig splits cloudflare_zone_settings_override into multiple cloudflare_zone_setting resources
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	var newBlocks []*hclwrite.Block

	// Get the resource name from the block labels
	if len(block.Labels()) < 2 {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}
	resourceName := block.Labels()[1]

	// Get zone_id attribute
	var zoneIDAttr *hclwrite.Attribute
	if attr := block.Body().GetAttribute("zone_id"); attr != nil {
		zoneIDAttr = attr
	}

	// Find the settings block
	for _, settingsBlock := range block.Body().Blocks() {
		if settingsBlock.Type() == "settings" {
			// Process regular attributes in sorted order for consistency
			attrs := settingsBlock.Body().Attributes()
			var attrNames []string
			for name := range attrs {
				// Skip deprecated settings
				if m.isDeprecatedSetting(name) {
					continue
				}
				attrNames = append(attrNames, name)
			}
			sort.Strings(attrNames)

			for _, attrName := range attrNames {
				attr := attrs[attrName]
				// Map the v4 setting name to the correct v5 setting name
				mappedSettingName := m.mapSettingName(attrName)
				resourceFullName := resourceName + "_" + attrName
				newBlock := m.createZoneSettingResource(
					resourceFullName,
					mappedSettingName,
					zoneIDAttr,
					attr,
				)
				newBlocks = append(newBlocks, newBlock)
			}

			// Process nested blocks (security_header, nel)
			for _, nestedBlock := range settingsBlock.Body().Blocks() {
				if nestedBlock.Type() == "security_header" {
					resourceFullName := resourceName + "_security_header"
					newBlock := m.transformSecurityHeaderBlock(resourceFullName, zoneIDAttr, nestedBlock)
					newBlocks = append(newBlocks, newBlock)
				} else if nestedBlock.Type() == "nel" {
					resourceFullName := resourceName + "_nel"
					newBlock := m.transformNELBlock(resourceFullName, zoneIDAttr, nestedBlock)
					newBlocks = append(newBlocks, newBlock)
				}
			}
		}
	}

	return &transform.TransformResult{
		Blocks:         newBlocks,
		RemoveOriginal: true, // Remove the original zone_settings_override resource
	}, nil
}

func (m *V4ToV5Migrator) isDeprecatedSetting(settingName string) bool {
	deprecatedSettings := map[string]bool{
		"universal_ssl": true, // No longer exists in zone settings API
	}
	return deprecatedSettings[settingName]
}

func (m *V4ToV5Migrator) mapSettingName(v4Name string) string {
	settingNameMap := map[string]string{
		"zero_rtt": "0rtt", // v4 used "zero_rtt" but API expects "0rtt"
	}

	if v5Name, exists := settingNameMap[v4Name]; exists {
		return v5Name
	}
	return v4Name
}

func (m *V4ToV5Migrator) createZoneSettingResource(name, settingID string, zoneIDAttr, valueAttr *hclwrite.Attribute) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zone_setting", name})
	body := block.Body()

	// Set zone_id with the expression from the original attribute
	if zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("zone_id", tokens)
	}

	// Set setting_id
	body.SetAttributeValue("setting_id", cty.StringVal(settingID))

	// Set value with the expression from the original attribute
	if valueAttr != nil {
		tokens := valueAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("value", tokens)
	}

	return block
}

func (m *V4ToV5Migrator) transformSecurityHeaderBlock(name string, zoneIDAttr *hclwrite.Attribute, securityHeaderBlock *hclwrite.Block) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zone_setting", name})
	body := block.Body()

	// Set zone_id
	if zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("zone_id", tokens)
	}

	// Set setting_id
	body.SetAttributeValue("setting_id", cty.StringVal("security_header"))

	// Build the object tokens manually to preserve variable references
	// Security header needs to be wrapped in strict_transport_security for v5 API
	innerObjectTokens := m.buildObjectFromBlock(securityHeaderBlock)

	// Create the wrapper object with strict_transport_security key
	wrapperTokens := []*hclwrite.Token{
		{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte("    strict_transport_security")},
		{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
	}
	wrapperTokens = append(wrapperTokens, innerObjectTokens...)
	wrapperTokens = append(wrapperTokens,
		[]*hclwrite.Token{
			{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
			{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")},
			{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
		}...)

	body.SetAttributeRaw("value", wrapperTokens)

	return block
}

func (m *V4ToV5Migrator) transformNELBlock(name string, zoneIDAttr *hclwrite.Attribute, nelBlock *hclwrite.Block) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zone_setting", name})
	body := block.Body()

	// Set zone_id
	if zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("zone_id", tokens)
	}

	// Set setting_id
	body.SetAttributeValue("setting_id", cty.StringVal("nel"))

	// Build value object from nel block
	valueTokens := m.buildObjectFromBlock(nelBlock)
	body.SetAttributeRaw("value", valueTokens)

	return block
}

func (m *V4ToV5Migrator) buildObjectFromBlock(block *hclwrite.Block) []*hclwrite.Token {
	var tokens []*hclwrite.Token

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	// Get attributes in sorted order for consistency
	attrs := block.Body().Attributes()
	var attrNames []string
	for name := range attrs {
		attrNames = append(attrNames, name)
	}
	sort.Strings(attrNames)

	for _, attrName := range attrNames {
		attr := attrs[attrName]
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("      " + attrName),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("  "),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte("="),
		})
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("  "),
		})
		// Append the original expression tokens
		tokens = append(tokens, attr.Expr().BuildTokens(nil)...)
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
	}

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenIdent,
		Bytes: []byte("    "),
	})
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte("}"),
	})

	return tokens
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// State transformation for zone_settings is handled by the provider
	// The old cloudflare_zone_settings_override state will be removed
	// and new cloudflare_zone_setting states will be created via imports
	return stateJSON.String(), nil
}
