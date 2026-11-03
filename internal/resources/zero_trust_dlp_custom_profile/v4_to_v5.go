package zero_trust_dlp_custom_profile

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of DLP profiles from v4 to v5.
// Transforms cloudflare_dlp_profile or cloudflare_zero_trust_dlp_profile to the appropriate v5 resource type.
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_dlp_profile", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_dlp_profile", "v4", "v5", migrator)
	// Register under the predefined profile name so the e2e runner can find this migrator
	// when looking up by testdata resource name (zero_trust_dlp_predefined_profile).
	// This is needed because GetResourceRename only returns the custom profile mapping.
	internal.RegisterMigrator("cloudflare_zero_trust_dlp_predefined_profile", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_dlp_custom_profile"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_dlp_profile" || resourceType == "cloudflare_zero_trust_dlp_profile"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Note: This migrator handles TWO different target types (custom and predefined)
// We return the primary rename here (to custom_profile), but the actual rename
// is determined in TransformConfig based on the profile type
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	// Return both old types that map to this migrator
	// The global postprocessing will handle both cloudflare_dlp_profile and cloudflare_zero_trust_dlp_profile
	return []string{"cloudflare_dlp_profile", "cloudflare_zero_trust_dlp_profile"}, "cloudflare_zero_trust_dlp_custom_profile"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	typeAttr := body.GetAttribute("type")
	profileType := "custom"
	if typeAttr != nil {
		profileType = tfhcl.ExtractStringFromAttribute(typeAttr)
	}

	currentType := block.Labels()[0]
	needsMovedBlock := currentType != "cloudflare_zero_trust_dlp_custom_profile" &&
		currentType != "cloudflare_zero_trust_dlp_predefined_profile"

	var newType string

	switch profileType {
	case "custom":
		newType = "cloudflare_zero_trust_dlp_custom_profile"
		if currentType != newType {
			tfhcl.RenameResourceType(block, currentType, newType)
		}
		tfhcl.RemoveAttributes(body, "type")
		//m.ensureContextAwareness(body)
		m.transformCustomEntryBlocks(ctx, body, resourceName)

	case "predefined":
		newType = "cloudflare_zero_trust_dlp_predefined_profile"
		tfhcl.RenameResourceType(block, currentType, newType)
		tfhcl.RemoveAttributes(body, "type")
		tfhcl.RemoveAttributes(body, "name") // name is read-only (Computed) in v5 predefined profiles
		m.transformPredefinedEntryBlocks(ctx, body, ctx.Content, resourceName)

	default:
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, fmt.Errorf("unknown DLP profile type: %s", profileType)
	}

	resultBlocks := []*hclwrite.Block{block}

	// Generate moved block for state migration (only when renaming from old type)
	if needsMovedBlock && resourceName != "" {
		from := currentType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)
		resultBlocks = append(resultBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         resultBlocks,
		RemoveOriginal: true,
	}, nil
}

func (m *V4ToV5Migrator) transformCustomEntryBlocks(ctx *transform.Context, body *hclwrite.Body, resourceName string) {
	var entryBlocks []*hclwrite.Block
	var hasDynamicEntry bool

	for _, block := range body.Blocks() {
		if block.Type() == "entry" {
			entryBlocks = append(entryBlocks, block)
		} else if block.Type() == "dynamic" && len(block.Labels()) > 0 && block.Labels()[0] == "entry" {
			hasDynamicEntry = true
			// Since entries in v5 is a list attribute (not blocks), and dynamic blocks can't be used with attributes,
			// we need to leave dynamic blocks as-is and let the user manually convert them
			// This is a limitation of the v5 schema where entries is a list of objects
		}
	}

	// Only transform static entry blocks if there are no dynamic blocks
	// If there are dynamic blocks, we can't auto-migrate properly
	if len(entryBlocks) > 0 && !hasDynamicEntry {
		var entryObjects []hclwrite.Tokens
		for _, entryBlock := range entryBlocks {
			entryBody := entryBlock.Body()
			tfhcl.RemoveAttributes(entryBody, "id")
			m.transformPatternBlock(entryBody)
			objTokens := tfhcl.BuildObjectFromBlock(entryBlock)
			entryObjects = append(entryObjects, objTokens)
		}

		arrayTokens := hclwrite.TokensForTuple(entryObjects)
		body.SetAttributeRaw("entries", arrayTokens)
		tfhcl.RemoveBlocksByType(body, "entry")
	} else if hasDynamicEntry {
		// For resources with dynamic entry blocks, we need a different approach
		// Since v5 uses a list attribute instead of blocks, dynamic blocks won't work
		// We'll add a comment to notify the user
		body.AppendNewline()
		commentTokens := hclwrite.Tokens{
			{Type: hclsyntax.TokenComment, Bytes: []byte("# WARNING: Dynamic entry blocks cannot be automatically migrated to v5.\n")},
			{Type: hclsyntax.TokenComment, Bytes: []byte("# The v5 provider uses 'entries' as a list attribute, which doesn't support dynamic blocks.\n")},
			{Type: hclsyntax.TokenComment, Bytes: []byte("# Please manually convert dynamic entries to a static list or use for_each at the resource level.\n")},
		}
		for _, token := range commentTokens {
			body.AppendUnstructuredTokens(hclwrite.Tokens{token})
		}

		// Add diagnostic warning
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Dynamic block requires manual migration: cloudflare_zero_trust_dlp_custom_profile.%s", resourceName),
			Detail: `Dynamic entry blocks cannot be automatically migrated to v5.

The v5 provider uses 'entries' as a list attribute instead of blocks.
Dynamic blocks are not supported with list attributes.

To migrate manually:
  1. Convert dynamic entries to a static list, OR
  2. Use for_each at the resource level instead of dynamic blocks`,
		})
	}
}

// ensureContextAwareness ensures the v5 config has context_awareness.
// In v4, context_awareness was Computed — the API always returns a default even if
// the user didn't set it. The state migrator preserves this value. To keep state
// and config aligned (avoiding a plan diff that sends null to the API), we add the
// API default to the config when it's not already present.
func (m *V4ToV5Migrator) ensureContextAwareness(body *hclwrite.Body) {
	// Already present as a block
	if blocks := tfhcl.FindBlocksByType(body, "context_awareness"); len(blocks) > 0 {
		return
	}
	// Already present as an attribute
	if body.GetAttribute("context_awareness") != nil {
		return
	}

	// Build context_awareness as an object attribute using a temporary block,
	// then convert to attribute syntax via BuildObjectFromBlock.
	tmpBlock := hclwrite.NewBlock("context_awareness", nil)
	tmpBody := tmpBlock.Body()
	tmpBody.SetAttributeValue("enabled", cty.False)
	skipTmpBlock := tmpBody.AppendNewBlock("skip", nil)
	skipTmpBlock.Body().SetAttributeValue("files", cty.False)
	objTokens := tfhcl.BuildObjectFromBlock(tmpBlock)
	body.SetAttributeRaw("context_awareness", objTokens)
}

func (m *V4ToV5Migrator) transformPatternBlock(entryBody *hclwrite.Body) {
	var patternBlock *hclwrite.Block
	for _, block := range entryBody.Blocks() {
		if block.Type() == "pattern" {
			patternBlock = block
			break
		}
	}

	if patternBlock == nil {
		return
	}

	objTokens := tfhcl.BuildObjectFromBlock(patternBlock)
	entryBody.SetAttributeRaw("pattern", objTokens)
	entryBody.RemoveBlock(patternBlock)
}

// importAddressProfileIDRe matches a tf-migrate:import-address annotation of the form
// # tf-migrate:import-address=<account_id>/<profile_id>
// and captures the profile UUID (the second path segment).
var importAddressProfileIDRe = regexp.MustCompile(`#\s*tf-migrate:import-address=[^/\s]+/([0-9a-f-]{36})`)

// extractProfileIDFromImportAddress scans the raw file content for a
// tf-migrate:import-address annotation that appears immediately before the given
// resource declaration and returns the profile UUID (second path segment), or "".
func extractProfileIDFromImportAddress(content []byte, resourceName string) string {
	resourceDecl := regexp.MustCompile(
		`resource\s+"cloudflare_(?:dlp_profile|zero_trust_dlp_profile|zero_trust_dlp_predefined_profile)"\s+"` +
			regexp.QuoteMeta(resourceName) + `"`)

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if resourceDecl.MatchString(line) {
			// Search backwards through the preceding comment block for the annotation.
			for j := i - 1; j >= 0; j-- {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "" {
					continue
				}
				if !strings.HasPrefix(trimmed, "#") {
					break
				}
				if m := importAddressProfileIDRe.FindStringSubmatch(trimmed); m != nil {
					return m[1]
				}
			}
			break
		}
	}
	return ""
}

func (m *V4ToV5Migrator) transformPredefinedEntryBlocks(ctx *transform.Context, body *hclwrite.Body, fileContent []byte, resourceName string) {
	// Each element is either a literal ID (enabled = true), or an expression that
	// yields the ID when enabled and null otherwise. Expressions are kept verbatim:
	// reading them as literals would drop entries enabled through a variable or
	// write a reference such as local.ids[0] as the ID "local" (BUGS-2156).
	var elements []string
	hasEntryBlocks, conditional := false, false
	var dynamicSources []string

	for _, block := range body.Blocks() {
		switch {
		case block.Type() == "entry":
			hasEntryBlocks = true
			entryBody := block.Body()
			idAttr, enabledAttr := entryBody.GetAttribute("id"), entryBody.GetAttribute("enabled")
			if idAttr == nil || enabledAttr == nil {
				continue
			}
			id := exprSource(idAttr)
			if enabled, literal := tfhcl.ExtractBoolFromAttribute(enabledAttr); literal && isBoolLiteral(enabledAttr) {
				if enabled {
					elements = append(elements, id)
				}
				continue
			}
			elements = append(elements, fmt.Sprintf("(%s) ? %s : null", exprSource(enabledAttr), id))
			conditional = true

		case block.Type() == "dynamic" && len(block.Labels()) == 1 && block.Labels()[0] == "entry":
			hasEntryBlocks = true
			if source, ok := dynamicEntrySource(block); ok {
				dynamicSources = append(dynamicSources, source)
			} else {
				ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  fmt.Sprintf("Dynamic entry block requires manual migration: cloudflare_zero_trust_dlp_predefined_profile.%s", resourceName),
					Detail:   "The dynamic entry block has no for_each, content.id or content.enabled, so its enabled entries cannot be converted to enabled_entries. Set enabled_entries explicitly before applying v5, or restore it from the live profile with preserve-zero-trust-settings.",
				})
			}
		}
	}

	tfhcl.RemoveBlocksByType(body, "entry")
	for _, block := range body.Blocks() {
		if block.Type() == "dynamic" && len(block.Labels()) == 1 && block.Labels()[0] == "entry" {
			body.RemoveBlock(block)
		}
	}

	// With ignore_changes on entry, v4 never reconciled the live entries to the
	// configuration, so the configured entries may not be the enabled ones. Keep
	// ignoring drift under the v5 name and leave enabled_entries for
	// preserve-zero-trust-settings to restore from the live profile.
	if renameIgnoredEntry(body) {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("enabled_entries not migrated: cloudflare_zero_trust_dlp_predefined_profile.%s ignores changes to entry", resourceName),
			Detail:   "The v4 configuration ignored changes to entry, so it may not match the entries enabled live. Restore enabled_entries from the live profile with preserve-zero-trust-settings before applying v5; omitting it disables detection for every entry.",
		})
		hasEntryBlocks = false
	}

	// If there were entry blocks in v4 config, always add enabled_entries
	// (even if empty) to prevent drift between state and config
	if hasEntryBlocks {
		list := "[" + strings.Join(elements, ", ") + "]"
		if conditional {
			list = "compact(" + list + ")"
		}
		parts := []string{}
		if len(elements) > 0 || len(dynamicSources) == 0 {
			parts = append(parts, list)
		}
		parts = append(parts, dynamicSources...)
		expr := parts[0]
		if len(parts) > 1 {
			expr = "concat(" + strings.Join(parts, ", ") + ")"
		}
		body.SetAttributeRaw("enabled_entries", exprTokens(expr))
	}

	// Resolve profile_id: prefer the resource-level id attribute (v4 state import
	// pattern), then fall back to the tf-migrate:import-address annotation comment.
	// Skip entirely if profile_id is already present (e.g. already-migrated v5 file).
	if body.GetAttribute("profile_id") != nil {
		// Already set — nothing to do.
	} else if idAttr := body.GetAttribute("id"); idAttr != nil {
		tfhcl.RenameAttribute(body, "id", "profile_id")
	} else if profileID := extractProfileIDFromImportAddress(fileContent, resourceName); profileID != "" {
		body.SetAttributeValue("profile_id", cty.StringVal(profileID))
	}
}

func exprSource(attr *hclwrite.Attribute) string {
	return strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
}

func isBoolLiteral(attr *hclwrite.Attribute) bool {
	source := exprSource(attr)
	return source == "true" || source == "false"
}

// exprTokens parses generated expression source into tokens so the formatter
// lays it out like hand-written configuration.
func exprTokens(source string) hclwrite.Tokens {
	file, diags := hclwrite.ParseConfig([]byte("x = "+source+"\n"), "", hcl.InitialPos)
	if diags.HasErrors() {
		return hclwrite.TokensForIdentifier(source)
	}
	return file.Body().GetAttribute("x").Expr().BuildTokens(nil)
}

// dynamicEntrySource converts a dynamic entry block into a for expression that
// yields the IDs of its enabled entries. The iterator object is rebuilt with key
// and value, so content expressions such as entry.value.id stay valid unchanged.
func dynamicEntrySource(block *hclwrite.Block) (string, bool) {
	forEach := block.Body().GetAttribute("for_each")
	iterator := "entry"
	if attr := block.Body().GetAttribute("iterator"); attr != nil {
		iterator = exprSource(attr)
	}
	var content *hclwrite.Block
	for _, child := range block.Body().Blocks() {
		if child.Type() == "content" {
			content = child
		}
	}
	if forEach == nil || content == nil {
		return "", false
	}
	id, enabled := content.Body().GetAttribute("id"), content.Body().GetAttribute("enabled")
	if id == nil || enabled == nil {
		return "", false
	}
	return fmt.Sprintf("[for %s in [for key, value in %s : { key = key, value = value }] : %s if %s]",
		iterator, exprSource(forEach), exprSource(id), exprSource(enabled)), true
}

// renameIgnoredEntry renames entry to enabled_entries in lifecycle.ignore_changes,
// since v5 has no entry attribute, and reports whether entry changes were ignored.
func renameIgnoredEntry(body *hclwrite.Body) bool {
	for _, lifecycle := range body.Blocks() {
		if lifecycle.Type() != "lifecycle" {
			continue
		}
		attr := lifecycle.Body().GetAttribute("ignore_changes")
		if attr == nil {
			continue
		}
		tokens := attr.Expr().BuildTokens(nil)
		ignored := false
		for _, token := range tokens {
			if token.Type != hclsyntax.TokenIdent {
				continue
			}
			switch string(token.Bytes) {
			case "entry":
				token.Bytes = []byte("enabled_entries")
				ignored = true
			case "all":
				ignored = true
			}
		}
		if ignored {
			lifecycle.Body().SetAttributeRaw("ignore_changes", tokens)
			return true
		}
	}
	return false
}
