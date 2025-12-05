package zero_trust_access_applications

import (
	"strconv"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register both old v4 name and current v4/v5 name
	// v4 had both cloudflare_access_application and cloudflare_zero_trust_access_application
	internal.RegisterMigrator("cloudflare_access_application", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_application", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the v5 resource name
	return "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated v4 name and the current name
	return resourceType == "cloudflare_access_application" ||
		resourceType == "cloudflare_zero_trust_access_application"
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_application", "cloudflare_zero_trust_access_application"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Preprocessing will be added if needed for block→attribute syntax conversion
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename resource type if it's the old name
	resourceType := tfhcl.GetResourceType(block)
	if resourceType == "cloudflare_access_application" {
		tfhcl.RenameResourceType(block, "cloudflare_access_application", "cloudflare_zero_trust_access_application")
	}

	body := block.Body()

	tfhcl.RemoveAttributes(body, "domain_type")

	tfhcl.ConvertBlocksToAttribute(body, "cors_headers", "cors_headers", nil)
	tfhcl.ConvertBlocksToAttributeList(body, "destinations", nil)
	tfhcl.ConvertBlocksToAttributeList(body, "footer_links", nil)
	tfhcl.ConvertBlocksToAttribute(body, "landing_page_design", "landing_page_design", nil)

	tfhcl.ConvertArrayAttributeToObjectArray(body, "policies", func(element hclwrite.Tokens, index int) map[string]hclwrite.Tokens {
		return map[string]hclwrite.Tokens{
			"id": element,
			"precedence": {
				&hclwrite.Token{
					Type:  hclsyntax.TokenNumberLit,
					Bytes: []byte(strconv.Itoa(index + 1)),
				},
			},
		}
	})

	m.transformSaasAppBlock(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) transformSaasAppBlock(body *hclwrite.Body) {
	saasAppBlocks := tfhcl.FindBlocksByType(body, "saas_app")
	if len(saasAppBlocks) == 0 {
		return
	}

	for _, saasAppBlock := range saasAppBlocks {
		saasAppBody := saasAppBlock.Body()

		// Process custom_attribute blocks before converting to list
		customAttrBlocks := tfhcl.FindBlocksByType(saasAppBody, "custom_attribute")
		for _, customAttrBlock := range customAttrBlocks {
			customAttrBody := customAttrBlock.Body()
			// Convert source block
			if sourceBlock := tfhcl.FindBlockByType(customAttrBody, "source"); sourceBlock != nil {
				sourceBody := sourceBlock.Body()
				// Convert source.name_by_idp from map to object array (SAML)
				tfhcl.ConvertMapAttributeToObjectArray(sourceBody, "name_by_idp", func(key hclwrite.Tokens, value hclwrite.Tokens) map[string]hclwrite.Tokens {
					return map[string]hclwrite.Tokens{
						"idp_id":      key,
						"source_name": value,
					}
				})
			}

			tfhcl.ConvertSingleBlockToAttribute(customAttrBody, "source", "source")
		}

		// Process custom_claim blocks before converting to list
		customClaimBlocks := tfhcl.FindBlocksByType(saasAppBody, "custom_claim")
		for _, customClaimBlock := range customClaimBlocks {
			customClaimBody := customClaimBlock.Body()
			// Convert source block to attribute
			// NOTE: For custom_claims (OIDC), name_by_idp stays as a map, so no transformation needed
			tfhcl.ConvertSingleBlockToAttribute(customClaimBody, "source", "source")
		}

		tfhcl.ConvertBlocksToAttributeList(saasAppBody, "custom_attribute", nil)
		tfhcl.RenameAttribute(saasAppBody, "custom_attribute", "custom_attributes")

		tfhcl.ConvertBlocksToAttributeList(saasAppBody, "custom_claim", nil)
		tfhcl.RenameAttribute(saasAppBody, "custom_claim", "custom_claims")

		tfhcl.ConvertSingleBlockToAttribute(saasAppBody, "hybrid_and_implicit_options", "hybrid_and_implicit_options")
		tfhcl.ConvertSingleBlockToAttribute(saasAppBody, "refresh_token_options", "refresh_token_options")
	}

	tfhcl.ConvertSingleBlockToAttribute(body, "saas_app", "saas_app")
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Handle both full state and single instance transformation
	if stateJSON.Get("resources").Exists() {
		return m.transformFullState(result, stateJSON)
	}

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	// Rename resource type if it's the old name (for single instance tests)
	resourceType := stateJSON.Get("type").String()
	if resourceType == "cloudflare_access_application" {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_access_application")
	}

	result = m.transformSingleInstance(result, stateJSON)

	return result, nil
}

func (m *V4ToV5Migrator) transformFullState(result string, stateJSON gjson.Result) (string, error) {
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		if !m.CanHandle(resourceType) {
			return true // continue
		}

		// Rename cloudflare_access_application to cloudflare_zero_trust_access_application
		if resourceType == "cloudflare_access_application" {
			resourcePath := "resources." + key.String() + ".type"
			result, _ = sjson.Set(result, resourcePath, "cloudflare_zero_trust_access_application")
		}

		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			attrs := instance.Get("attributes")
			if attrs.Exists() {
				instJSON := instance.String()
				transformedInst := m.transformSingleInstance(instJSON, instance)
				transformedInstParsed := gjson.Parse(transformedInst)
				result, _ = sjson.SetRaw(result, instPath, transformedInstParsed.Raw)
			}
			return true
		})

		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result
	}

	// TODO: Add state transformations
	// 1. MaxItems:1 array → object (saas_app, landing_page_design, scim_config, cors_headers, etc.)
	// 2. policies: string array → object array
	// 3. target_criteria[].target_attributes: list → map
	// 4. scim_config.authentication: multi-item → single
	// 5. Type conversions (cors_headers.max_age: int → float64)
	// 6. Add v5 defaults (destinations[].type, landing_page_design.title, saas_app.auth_type)
	// 7. Remove deprecated fields

	// Placeholder: Type conversion example
	// if maxAge := attrs.Get("cors_headers.max_age"); maxAge.Exists() {
	//     floatVal := state.ConvertToFloat64(maxAge)
	//     result, _ = sjson.Set(result, "attributes.cors_headers.max_age", floatVal)
	// }

	_ = state.ConvertToFloat64 // Prevent unused import error during initial development

	// Always set schema_version
	result, _ = sjson.Set(result, "schema_version", 0)

	return result
}
