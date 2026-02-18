package zone_setting

import (
	"fmt"
	"sort"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"
)

// V4ToV5Migrator handles the migration of cloudflare_zone_settings_override from v4 to v5
// This is a special case: one v4 resource splits into multiple v5 cloudflare_zone_setting resources
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with OLD v4 name
	internal.RegisterMigrator("cloudflare_zone_settings_override", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone_settings_override"
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW v5 resource type
	return "cloudflare_zone_setting"
}

// Preprocess handles any string-level transformations before HCL parsing
// For zone_settings_override, no preprocessing is needed
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig performs the one-to-many transformation:
// One cloudflare_zone_settings_override → Multiple cloudflare_zone_setting resources
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	var newBlocks []*hclwrite.Block

	// Extract the original resource name (e.g., "example" from cloudflare_zone_settings_override "example")
	if len(block.Labels()) < 2 {
		return nil, fmt.Errorf("invalid resource block: expected 2 labels, got %d", len(block.Labels()))
	}
	baseName := block.Labels()[1]

	// Get zone_id attribute - we'll reuse this for all new resources
	zoneIDAttr := block.Body().GetAttribute("zone_id")
	if zoneIDAttr == nil {
		return nil, fmt.Errorf("zone_id attribute not found in resource %s", baseName)
	}

	// Extract meta-arguments (count, for_each, lifecycle, depends_on, etc.)
	// These need to be copied to all generated resources
	metaArgs := m.extractMetaArguments(block)

	// Find the settings block
	var settingsBlock *hclwrite.Block
	for _, b := range block.Body().Blocks() {
		if b.Type() == "settings" {
			settingsBlock = b
			break
		}
	}

	if settingsBlock == nil {
		// No settings block - nothing to migrate
		return &transform.TransformResult{
			Blocks:         nil,
			RemoveOriginal: true,
		}, nil
	}

	// Process each attribute in the settings block in sorted order for deterministic output
	// Each attribute becomes a separate cloudflare_zone_setting resource
	attrs := settingsBlock.Body().Attributes()
	var attrNames []string
	for attrName := range attrs {
		attrNames = append(attrNames, attrName)
	}
	sort.Strings(attrNames)

	for _, attrName := range attrNames {
		attr := attrs[attrName]
		// Skip deprecated settings
		if isDeprecatedSetting(attrName) {
			continue
		}

		// Map the setting name (e.g., "zero_rtt" → "0rtt")
		settingID := mapSettingName(attrName)

		// Generate resource name: {original_name}_{setting_name}
		resourceName := baseName + "_" + attrName

		// Create new cloudflare_zone_setting resource
		newResource := m.createZoneSettingResource(resourceName, settingID, zoneIDAttr, attr)

		// Copy meta-arguments to the new resource
		m.copyMetaArguments(newResource, metaArgs)

		newBlocks = append(newBlocks, newResource)

		// Note: Import blocks are NOT generated because:
		// 1. Import blocks can only be used in root modules, not child modules
		// 2. Import blocks don't support count/for_each meta-arguments
		// The state will be recreated when users run `terraform apply` after migration
	}

	// Process nested blocks (minify, mobile_redirect, security_header, nel, aegis)
	for _, nestedBlock := range settingsBlock.Body().Blocks() {
		settingID := nestedBlock.Type()
		resourceName := baseName + "_" + settingID

		// Skip deprecated settings
		if isDeprecatedSetting(settingID) {
			continue
		}

		var newResource *hclwrite.Block
		if settingID == "security_header" {
			// Special handling: wrap in strict_transport_security
			newResource = m.transformSecurityHeaderBlock(resourceName, settingID, zoneIDAttr, nestedBlock)
		} else {
			// Standard nested block transformation
			newResource = m.transformNestedBlock(resourceName, settingID, zoneIDAttr, nestedBlock)
		}

		// Copy meta-arguments to the new resource
		m.copyMetaArguments(newResource, metaArgs)

		newBlocks = append(newBlocks, newResource)

		// Note: Import blocks are NOT generated (same reasons as above)
	}

	return &transform.TransformResult{
		Blocks:         newBlocks,
		RemoveOriginal: true, // Remove the original zone_settings_override resource
	}, nil
}

// createZoneSettingResource creates a new cloudflare_zone_setting resource for a simple attribute
func (m *V4ToV5Migrator) createZoneSettingResource(resourceName, settingID string, zoneIDAttr, valueAttr *hclwrite.Attribute) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zone_setting", resourceName})
	body := block.Body()

	// Set zone_id with the expression from the original attribute
	if zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("zone_id", tokens)
	}

	// Set setting_id as a literal string
	body.SetAttributeValue("setting_id", cty.StringVal(settingID))

	// Set value with the expression from the original attribute
	if valueAttr != nil {
		tokens := valueAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("value", tokens)
	}

	return block
}

// transformNestedBlock converts a nested block (like minify, nel) into a cloudflare_zone_setting resource
// The block's attributes become an object value
func (m *V4ToV5Migrator) transformNestedBlock(resourceName, settingID string, zoneIDAttr *hclwrite.Attribute, nestedBlock *hclwrite.Block) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zone_setting", resourceName})
	body := block.Body()

	// Set zone_id
	if zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("zone_id", tokens)
	}

	// Set setting_id
	body.SetAttributeValue("setting_id", cty.StringVal(settingID))

	// Build object tokens from the nested block's attributes
	objectTokens := m.buildObjectFromBlock(nestedBlock)
	body.SetAttributeRaw("value", objectTokens)

	return block
}

// transformSecurityHeaderBlock handles the special case of security_header
// which requires wrapping in strict_transport_security for the v5 API
func (m *V4ToV5Migrator) transformSecurityHeaderBlock(resourceName, settingID string, zoneIDAttr *hclwrite.Attribute, securityHeaderBlock *hclwrite.Block) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"cloudflare_zone_setting", resourceName})
	body := block.Body()

	// Set zone_id
	if zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("zone_id", tokens)
	}

	// Set setting_id
	body.SetAttributeValue("setting_id", cty.StringVal(settingID))

	// Build inner object from the security_header block
	innerObjectTokens := m.buildObjectFromBlock(securityHeaderBlock)

	// Create wrapper object with strict_transport_security key
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
			{Type: hclsyntax.TokenCBrace, Bytes: []byte("  }")},
		}...)

	body.SetAttributeRaw("value", wrapperTokens)

	return block
}

// buildObjectFromBlock creates HCL object tokens from a block's attributes
// Sorts attributes alphabetically for consistent output
func (m *V4ToV5Migrator) buildObjectFromBlock(block *hclwrite.Block) hclwrite.Tokens {
	// Get all attributes from the block
	attrs := block.Body().Attributes()

	// Sort attribute names for deterministic output
	var names []string
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build a list of attribute tokens in sorted order
	var attrTokens []hclwrite.ObjectAttrTokens

	for _, name := range names {
		attr := attrs[name]
		// Create tokens for the attribute name
		nameTokens := hclwrite.TokensForIdentifier(name)

		// Get the value tokens from the attribute's expression
		valueTokens := attr.Expr().BuildTokens(nil)

		attrTokens = append(attrTokens, hclwrite.ObjectAttrTokens{
			Name:  nameTokens,
			Value: valueTokens,
		})
	}

	// Use the built-in TokensForObject function to create properly formatted object tokens
	return hclwrite.TokensForObject(attrTokens)
}

// createImportBlock generates an import block for a cloudflare_zone_setting resource
// Import ID format: ${zone_id}/{setting_id}
func (m *V4ToV5Migrator) createImportBlock(resourceName, settingID string, zoneIDAttr *hclwrite.Attribute) *hclwrite.Block {
	block := hclwrite.NewBlock("import", nil)
	body := block.Body()

	// Build the "to" value: cloudflare_zone_setting.resource_name
	toTokens := buildResourceReference("cloudflare_zone_setting", resourceName)
	body.SetAttributeRaw("to", toTokens)

	// Build the "id" value: "${zone_id}/{setting_id}"
	if zoneIDAttr != nil {
		zoneIDTokens := zoneIDAttr.Expr().BuildTokens(nil)
		idTokens := buildTemplateStringTokens(zoneIDTokens, "/"+settingID)
		body.SetAttributeRaw("id", idTokens)
	}

	return block
}

// TransformState removes cloudflare_zone_settings_override instances from state.
// This is a one-to-many migration: one v4 resource splits into N v5 cloudflare_zone_setting
// resources. The v5 provider has no schema for cloudflare_zone_settings_override, so any
// state entry with that type would cause Terraform to error with "no schema available".
//
// Returning "" signals tf-migrate to delete this state instance. After migration:
// - cloudflare_zone_settings_override entries are removed from state
// - New cloudflare_zone_setting resources are created fresh via terraform apply
// - The provider's UpgradeState handles future v5 internal schema bumps independently
//
// NOTE: This migrator intentionally does NOT implement UsesProviderStateUpgrader.
// UsesProviderStateUpgrader is designed for resources where the v4 and v5 resource
// types are the same (e.g. cloudflare_access_rule → cloudflare_access_rule) and the
// provider's UpgradeState bumps the schema version while keeping attributes intact.
// That mechanism skips --state-file entirely, so TransformState is never called.
//
// For zone_setting, skipping --state-file would leave cloudflare_zone_settings_override
// in the state file untouched. The v5 provider has no schema for that type, so Terraform
// would fail with "no schema available" when loading the state. The provider's UpgradeState
// only handles cloudflare_zone_setting resources that already exist within v5; it cannot
// bridge the type change from cloudflare_zone_settings_override → cloudflare_zone_setting.
// Therefore state cleanup must happen here in tf-migrate via TransformState returning "".
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return "", nil
}

// isDeprecatedSetting checks if a setting should be skipped during migration
func isDeprecatedSetting(settingName string) bool {
	deprecatedSettings := map[string]bool{
		"universal_ssl": true, // No longer exists in zone settings API
	}
	return deprecatedSettings[settingName]
}

// mapSettingName translates v4 setting names to v5 setting IDs
// This handles cases where the v4 provider used different names than the API
func mapSettingName(v4Name string) string {
	settingNameMap := map[string]string{
		"zero_rtt": "0rtt", // v4 used "zero_rtt" but API expects "0rtt"
	}

	if v5Name, exists := settingNameMap[v4Name]; exists {
		return v5Name
	}
	return v4Name
}

// buildResourceReference creates tokens for a resource reference
// e.g., cloudflare_zone_setting.resource_name
func buildResourceReference(resourceType, resourceName string) hclwrite.Tokens {
	return hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(resourceType)},
		{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(resourceName)},
	}
}

// buildTemplateStringTokens creates tokens for a template string
// e.g., "${zone_id}/setting_id" for variables or "literal/setting_id" for literals
func buildTemplateStringTokens(zoneIDTokens hclwrite.Tokens, suffix string) hclwrite.Tokens {
	// Check if zone_id is a literal string (OQuote, QuotedLit, CQuote pattern)
	if len(zoneIDTokens) == 3 &&
		zoneIDTokens[0].Type == hclsyntax.TokenOQuote &&
		zoneIDTokens[1].Type == hclsyntax.TokenQuotedLit &&
		zoneIDTokens[2].Type == hclsyntax.TokenCQuote {
		// It's a literal string, so concatenate directly without template interpolation
		literalValue := string(zoneIDTokens[1].Bytes)
		return hclwrite.Tokens{
			{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
			{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(literalValue + suffix)},
			{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
		}
	}

	// It's a variable or expression, use template interpolation
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
		{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")},
	}

	// Append the zone_id tokens (variable reference or complex expression)
	tokens = append(tokens, zoneIDTokens...)

	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte("}")},
		&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(suffix)},
		&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
	)

	return tokens
}

// metaArguments holds meta-arguments extracted from a resource block
type metaArguments struct {
	count     *hclwrite.Attribute
	forEach   *hclwrite.Attribute
	lifecycle *hclwrite.Block
	dependsOn *hclwrite.Attribute
	provider  *hclwrite.Attribute
	timeouts  *hclwrite.Block
}

// extractMetaArguments extracts meta-arguments from the original resource block
func (m *V4ToV5Migrator) extractMetaArguments(block *hclwrite.Block) *metaArguments {
	body := block.Body()
	meta := &metaArguments{}

	// Extract count and for_each
	if attr := body.GetAttribute("count"); attr != nil {
		meta.count = attr
	}
	if attr := body.GetAttribute("for_each"); attr != nil {
		meta.forEach = attr
	}
	if attr := body.GetAttribute("depends_on"); attr != nil {
		meta.dependsOn = attr
	}
	if attr := body.GetAttribute("provider"); attr != nil {
		meta.provider = attr
	}

	// Extract lifecycle and timeouts blocks
	for _, b := range body.Blocks() {
		switch b.Type() {
		case "lifecycle":
			meta.lifecycle = b
		case "timeouts":
			meta.timeouts = b
		}
	}

	return meta
}

// copyMetaArguments copies meta-arguments to a new resource block
// For zone_setting, we need special handling of lifecycle blocks because
// ignore_changes paths need to be transformed from v4 to v5 structure
func (m *V4ToV5Migrator) copyMetaArguments(newBlock *hclwrite.Block, meta *metaArguments) {
	if meta == nil {
		return
	}

	body := newBlock.Body()

	// Copy count
	if meta.count != nil {
		tokens := meta.count.Expr().BuildTokens(nil)
		body.SetAttributeRaw("count", tokens)
	}

	// Copy for_each
	if meta.forEach != nil {
		tokens := meta.forEach.Expr().BuildTokens(nil)
		body.SetAttributeRaw("for_each", tokens)
	}

	// Copy depends_on
	if meta.dependsOn != nil {
		tokens := meta.dependsOn.Expr().BuildTokens(nil)
		body.SetAttributeRaw("depends_on", tokens)
	}

	// Copy provider
	if meta.provider != nil {
		tokens := meta.provider.Expr().BuildTokens(nil)
		body.SetAttributeRaw("provider", tokens)
	}

	// Copy lifecycle block with special handling for zone_setting
	// Since v4 had settings[0].xxx and v5 just has 'value',
	// we need to drop the lifecycle block entirely to avoid invalid references
	// Users can re-add lifecycle blocks manually if needed after migration
	if meta.lifecycle != nil {
		// Check if lifecycle has ignore_changes with settings references
		ignoreChanges := meta.lifecycle.Body().GetAttribute("ignore_changes")
		if ignoreChanges != nil {
			// For zone_setting, ignore_changes referencing settings[0].xxx paths
			// cannot be automatically migrated because the v5 structure is completely different
			// We skip copying the lifecycle block to avoid terraform errors
			// Users will need to manually add lifecycle blocks if still needed
		} else {
			// No ignore_changes or it's safe to copy - copy other lifecycle attributes
			lifecycleBlock := body.AppendNewBlock("lifecycle", nil)
			// Copy all attributes except ignore_changes
			for name, attr := range meta.lifecycle.Body().Attributes() {
				if name != "ignore_changes" {
					tokens := attr.Expr().BuildTokens(nil)
					lifecycleBlock.Body().SetAttributeRaw(name, tokens)
				}
			}
		}
	}

	// Copy timeouts block
	if meta.timeouts != nil {
		timeoutsBlock := body.AppendNewBlock("timeouts", nil)
		m.copyBlockContents(meta.timeouts.Body(), timeoutsBlock.Body())
	}
}

// copyIterationMetaArguments copies count/for_each to import blocks
func (m *V4ToV5Migrator) copyIterationMetaArguments(importBlock *hclwrite.Block, meta *metaArguments) {
	if meta == nil {
		return
	}

	body := importBlock.Body()

	// Copy count
	if meta.count != nil {
		tokens := meta.count.Expr().BuildTokens(nil)
		body.SetAttributeRaw("count", tokens)
	}

	// Copy for_each
	if meta.forEach != nil {
		tokens := meta.forEach.Expr().BuildTokens(nil)
		body.SetAttributeRaw("for_each", tokens)
	}
}

// copyBlockContents copies all attributes and nested blocks from source to destination
func (m *V4ToV5Migrator) copyBlockContents(src, dst *hclwrite.Body) {
	// Copy attributes
	for name, attr := range src.Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		dst.SetAttributeRaw(name, tokens)
	}

	// Copy nested blocks
	for _, block := range src.Blocks() {
		newBlock := dst.AppendNewBlock(block.Type(), block.Labels())
		m.copyBlockContents(block.Body(), newBlock.Body())
	}
}
