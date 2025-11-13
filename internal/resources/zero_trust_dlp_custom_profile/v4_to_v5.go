package zero_trust_dlp_custom_profile

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
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

	typeAttr := body.GetAttribute("type")
	profileType := "custom"
	if typeAttr != nil {
		profileType = tfhcl.ExtractStringFromAttribute(typeAttr)
	}

	currentType := block.Labels()[0]

	switch profileType {
	case "custom":
		if currentType != "cloudflare_zero_trust_dlp_custom_profile" {
			tfhcl.RenameResourceType(block, currentType, "cloudflare_zero_trust_dlp_custom_profile")
		}
		tfhcl.RemoveAttributes(body, "type")
		m.transformCustomEntryBlocks(body)

	case "predefined":
		tfhcl.RenameResourceType(block, currentType, "cloudflare_zero_trust_dlp_predefined_profile")
		tfhcl.RemoveAttributes(body, "type")
		m.transformPredefinedEntryBlocks(body)

	default:
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, fmt.Errorf("unknown DLP profile type: %s", profileType)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) transformCustomEntryBlocks(body *hclwrite.Body) {
	var entryBlocks []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "entry" {
			entryBlocks = append(entryBlocks, block)
		}
	}

	if len(entryBlocks) == 0 {
		return
	}

	var entryObjects []hclwrite.Tokens
	for _, entryBlock := range entryBlocks {
		entryBody := entryBlock.Body()
		tfhcl.RemoveAttributes(entryBody, "id")
		m.transformPatternBlock(entryBody)
		objTokens := hcl.BuildObjectFromBlock(entryBlock)
		entryObjects = append(entryObjects, objTokens)
	}

	arrayTokens := hclwrite.TokensForTuple(entryObjects)
	body.SetAttributeRaw("entries", arrayTokens)
	tfhcl.RemoveBlocksByType(body, "entry")
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

	objTokens := hcl.BuildObjectFromBlock(patternBlock)
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

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("instances").Exists() {
		profileType := "custom"
		instances := stateJSON.Get("instances")
		if instances.IsArray() && len(instances.Array()) > 0 {
			typeField := instances.Array()[0].Get("attributes.type")
			if typeField.Exists() {
				profileType = typeField.String()
			}
		}

		resourceType := stateJSON.Get("type").String()
		if resourceType == "cloudflare_dlp_profile" || resourceType == "cloudflare_zero_trust_dlp_profile" {
			switch profileType {
			case "custom":
				result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_dlp_custom_profile")
			case "predefined":
				result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_dlp_predefined_profile")
			}
		}

		instances = gjson.Get(result, "instances")
		if instances.IsArray() {
			for i, instance := range instances.Array() {
				transformedInstance := m.transformInstance(instance)
				transformedJSON := transformedInstance.String()
				result, _ = sjson.SetRaw(result, fmt.Sprintf("instances.%d", i), transformedJSON)
			}
		}

		return result, nil
	}

	transformedInstance := m.transformInstance(stateJSON)
	return transformedInstance.String(), nil
}

func (m *V4ToV5Migrator) transformInstance(instance gjson.Result) gjson.Result {
	result := instance.String()

	profileType := "custom"
	typeField := gjson.Get(result, "attributes.type")
	if typeField.Exists() {
		profileType = typeField.String()
	}

	result, _ = sjson.Delete(result, "attributes.type")

	switch profileType {
	case "custom":
		entryData := gjson.Get(result, "attributes.entry")
		if entryData.Exists() && entryData.IsArray() {
			var transformedEntries []interface{}

			entryData.ForEach(func(key, value gjson.Result) bool {
				entry := make(map[string]interface{})

				if name := value.Get("name"); name.Exists() {
					entry["name"] = name.String()
				}
				if enabled := value.Get("enabled"); enabled.Exists() {
					entry["enabled"] = enabled.Bool()
				}

				patternArray := value.Get("pattern")
				if patternArray.Exists() && patternArray.IsArray() {
					patternData := patternArray.Array()
					if len(patternData) > 0 {
						pattern := make(map[string]interface{})
						if regex := patternData[0].Get("regex"); regex.Exists() {
							pattern["regex"] = regex.String()
						}
						if validation := patternData[0].Get("validation"); validation.Exists() && validation.String() != "" {
							pattern["validation"] = validation.String()
						}
						entry["pattern"] = pattern
					}
				}

				transformedEntries = append(transformedEntries, entry)
				return true
			})

			result, _ = sjson.Set(result, "attributes.entries", transformedEntries)
			result, _ = sjson.Delete(result, "attributes.entry")
		}

	case "predefined":
		entryData := gjson.Get(result, "attributes.entry")
		if entryData.Exists() && entryData.IsArray() {
			var enabledEntryIDs []string

			entryData.ForEach(func(key, value gjson.Result) bool {
				if enabled := value.Get("enabled"); enabled.Exists() && enabled.Bool() {
					if id := value.Get("id"); id.Exists() {
						enabledEntryIDs = append(enabledEntryIDs, id.String())
					}
				}
				return true
			})

			if len(enabledEntryIDs) > 0 {
				result, _ = sjson.Set(result, "attributes.enabled_entries", enabledEntryIDs)
			}

			result, _ = sjson.Delete(result, "attributes.entry")
		}

		if id := gjson.Get(result, "attributes.id"); id.Exists() {
			result, _ = sjson.Set(result, "attributes.profile_id", id.String())
		}
	}

	if allowedCount := gjson.Get(result, "attributes.allowed_match_count"); allowedCount.Exists() {
		convertedValue := state.ConvertToFloat64(allowedCount)
		result, _ = sjson.Set(result, "attributes.allowed_match_count", convertedValue)
	}

	contextAwareness := gjson.Get(result, "attributes.context_awareness")
	if contextAwareness.Exists() {
		if contextAwareness.IsArray() {
			result, _ = sjson.Delete(result, "attributes.context_awareness")
		} else if contextAwareness.IsObject() {
			skipField := contextAwareness.Get("skip")
			if skipField.Exists() && skipField.IsArray() {
				result, _ = sjson.Delete(result, "attributes.context_awareness")
			}
		}
	}

	result, _ = sjson.Set(result, "schema_version", 0)

	return gjson.Parse(result)
}
