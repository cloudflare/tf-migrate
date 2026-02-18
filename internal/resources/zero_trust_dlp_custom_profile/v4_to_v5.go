package zero_trust_dlp_custom_profile

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
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
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	// Return both old types that map to this migrator
	// The global postprocessing will handle both cloudflare_dlp_profile and cloudflare_zero_trust_dlp_profile
	
	return "cloudflare_dlp_profile", "cloudflare_zero_trust_dlp_custom_profile"
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
		m.transformCustomEntryBlocks(body)

	case "predefined":
		newType = "cloudflare_zero_trust_dlp_predefined_profile"
		tfhcl.RenameResourceType(block, currentType, newType)
		tfhcl.RemoveAttributes(body, "type")
		m.transformPredefinedEntryBlocks(body)

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

func (m *V4ToV5Migrator) transformCustomEntryBlocks(body *hclwrite.Body) {
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

func (m *V4ToV5Migrator) transformPredefinedEntryBlocks(body *hclwrite.Body) {
	var enabledEntryIDs []string
	for _, block := range body.Blocks() {
		if block.Type() == "entry" {
			entryBody := block.Body()
			enabledAttr := entryBody.GetAttribute("enabled")
			if enabledAttr != nil {
				enabled := tfhcl.ExtractStringFromAttribute(enabledAttr)
				if enabled == "true" {
					idAttr := entryBody.GetAttribute("id")
					if idAttr != nil {
						id := tfhcl.ExtractStringFromAttribute(idAttr)
						if id != "" {
							enabledEntryIDs = append(enabledEntryIDs, id)
						}
					}
				}
			}
		}
	}

	tfhcl.RemoveBlocksByType(body, "entry")

	if len(enabledEntryIDs) > 0 {
		var stringTokens []hclwrite.Tokens
		for _, id := range enabledEntryIDs {
			tokens := hclwrite.Tokens{
				{Type: hclsyntax.TokenOQuote, Bytes: []byte{'"'}},
				{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(id)},
				{Type: hclsyntax.TokenCQuote, Bytes: []byte{'"'}},
			}
			stringTokens = append(stringTokens, tokens)
		}

		arrayTokens := hclwrite.TokensForTuple(stringTokens)
		body.SetAttributeRaw("enabled_entries", arrayTokens)
	}

	if idAttr := body.GetAttribute("id"); idAttr != nil {
		tfhcl.RenameAttribute(body, "id", "profile_id")
	}
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
// via StateUpgraders (MoveState/UpgradeState) rather than tf-migrate state transformation.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

// TransformState is a no-op - state transformation is handled by the provider's StateUpgraders.
// The moved block generated in TransformConfig triggers the provider's migration logic.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}
