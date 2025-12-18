package zero_trust_access_identity_provider

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust Access Identity Provider resources from v4 to v5
type V4ToV5Migrator struct {
	oldType string
	newType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType: "cloudflare_access_identity_provider",
		newType: "cloudflare_zero_trust_access_identity_provider",
	}
	// Register with OLD resource name (v4 name)
	internal.RegisterMigrator("cloudflare_access_identity_provider", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType
}

// Preprocess - no preprocessing needed, transformation happens in TransformConfig
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_identity_provider", "cloudflare_zero_trust_access_identity_provider"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename cloudflare_access_identity_provider to cloudflare_zero_trust_access_identity_provider
	tfhcl.RenameResourceType(block, "cloudflare_access_identity_provider", "cloudflare_zero_trust_access_identity_provider")

	body := block.Body()

	// 1. Convert config block to attribute (with preprocessing)
	tfhcl.ConvertBlocksToAttribute(body, "config", "config", func(configBlock *hclwrite.Block) {
		configBody := configBlock.Body()

		// Remove deprecated api_token field
		tfhcl.RemoveAttributes(configBody, "api_token")

		// Rename idp_public_cert to idp_public_certs and wrap in array
		if certAttr := configBody.GetAttribute("idp_public_cert"); certAttr != nil {
			// Get the certificate value
			certTokens := certAttr.Expr().BuildTokens(nil)

			// Create array with the certificate value
			arrayTokens := hclwrite.Tokens{
				{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
			}
			arrayTokens = append(arrayTokens, certTokens...)
			arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

			// Set as idp_public_certs
			configBody.SetAttributeRaw("idp_public_certs", arrayTokens)

			// Remove old attribute
			configBody.RemoveAttribute("idp_public_cert")
		}
	})

	// 2. Convert scim_config block to attribute (with preprocessing)
	tfhcl.ConvertBlocksToAttribute(body, "scim_config", "scim_config", func(scimBlock *hclwrite.Block) {
		scimBody := scimBlock.Body()

		// Remove deprecated group_member_deprovision field
		tfhcl.RemoveAttributes(scimBody, "group_member_deprovision")

		// Remove secret field (it's now computed-only in v5, not user-settable)
		tfhcl.RemoveAttributes(scimBody, "secret")
	})

	// 3. Ensure config attribute exists (required in v5)
	// If no config block was found and no config attribute exists, add empty config
	if body.GetAttribute("config") == nil {
		emptyObjTokens := hclwrite.Tokens{
			{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
			{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
		}
		body.SetAttributeRaw("config", emptyObjTokens)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// This function receives a single instance and needs to return the transformed instance JSON
	result := stateJSON.String()

	// Get attributes from the instance
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Set schema_version even for instances without attributes
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// 1. Transform config array to object
	result = m.transformConfigField(result, attrs)

	// 2. Transform empty values to null for config fields not explicitly set in user's HCL
	// Re-parse attrs after config transformation
	attrs = gjson.Parse(result).Get("attributes")
	configField := attrs.Get("config")
	if configField.Exists() && configField.IsObject() {
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        "attributes.config",
			FieldResult:      configField,
			ResourceName:     resourceName,
			HCLAttributePath: "config",
			CanHandle:        m.CanHandle,
		})
	}

	// 3. Transform scim_config array to object
	result = m.transformScimConfigField(result, attrs)

	// Always set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformConfigField unwraps the config array and handles field transformations
func (m *V4ToV5Migrator) transformConfigField(result string, attrs gjson.Result) string {
	configField := attrs.Get("config")

	// Handle missing config (e.g., onetimepin type)
	if !configField.Exists() || configField.Type == gjson.Null {
		// Config is required in v5, set to empty object
		result, _ = sjson.Set(result, "attributes.config", map[string]interface{}{})
		return result
	}

	// Unwrap array to object [{}] → {}
	if configField.IsArray() {
		configArray := configField.Array()
		if len(configArray) > 0 {
			// Get the first element (config is MaxItems:1)
			configObj := configArray[0]

			// Convert to map for manipulation
			configMap := make(map[string]interface{})
			configObj.ForEach(func(key, value gjson.Result) bool {
				fieldName := key.String()

				// Skip deprecated api_token field
				if fieldName == "api_token" {
					return true
				}

				// Handle idp_public_cert → idp_public_certs transformation
				if fieldName == "idp_public_cert" {
					if value.Type == gjson.String && value.String() != "" {
						// Wrap string in array
						configMap["idp_public_certs"] = []string{value.String()}
					}
					return true
				}

				// Keep all other fields
				configMap[fieldName] = value.Value()
				return true
			})

			result, _ = sjson.Set(result, "attributes.config", configMap)
		} else {
			// Empty array, set to empty object
			result, _ = sjson.Set(result, "attributes.config", map[string]interface{}{})
		}
	} else if configField.IsObject() {
		// Already an object (shouldn't happen in v4, but handle gracefully)
		// Still need to check for deprecated fields and transformations
		configMap := make(map[string]interface{})
		configField.ForEach(func(key, value gjson.Result) bool {
			fieldName := key.String()

			if fieldName == "api_token" {
				return true
			}

			if fieldName == "idp_public_cert" {
				if value.Type == gjson.String && value.String() != "" {
					configMap["idp_public_certs"] = []string{value.String()}
				}
				return true
			}

			configMap[fieldName] = value.Value()
			return true
		})
		result, _ = sjson.Set(result, "attributes.config", configMap)
	}

	return result
}

// transformScimConfigField unwraps the scim_config array and handles field transformations
func (m *V4ToV5Migrator) transformScimConfigField(result string, attrs gjson.Result) string {
	scimConfigField := attrs.Get("scim_config")

	// scim_config is optional, if missing just return
	if !scimConfigField.Exists() || scimConfigField.Type == gjson.Null {
		return result
	}

	// Unwrap array to object [{}] → {}
	if scimConfigField.IsArray() {
		scimArray := scimConfigField.Array()
		if len(scimArray) > 0 {
			// Get the first element (scim_config is MaxItems:1)
			scimObj := scimArray[0]

			// Convert to map for manipulation
			scimMap := make(map[string]interface{})
			scimObj.ForEach(func(key, value gjson.Result) bool {
				fieldName := key.String()

				// Skip deprecated group_member_deprovision field
				if fieldName == "group_member_deprovision" {
					return true
				}

				// Keep all other fields (including secret - it's computed but preserve existing value)
				scimMap[fieldName] = value.Value()
				return true
			})

			result, _ = sjson.Set(result, "attributes.scim_config", scimMap)
		} else {
			// Empty array, delete scim_config (it's optional)
			result, _ = sjson.Delete(result, "attributes.scim_config")
		}
	} else if scimConfigField.IsObject() {
		// Already an object (shouldn't happen in v4, but handle gracefully)
		scimMap := make(map[string]interface{})
		scimConfigField.ForEach(func(key, value gjson.Result) bool {
			fieldName := key.String()

			if fieldName == "group_member_deprovision" {
				return true
			}

			scimMap[fieldName] = value.Value()
			return true
		})
		result, _ = sjson.Set(result, "attributes.scim_config", scimMap)
	}

	return result
}
