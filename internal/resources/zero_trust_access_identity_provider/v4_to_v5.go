package zero_trust_access_identity_provider

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
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
		tfhcl.RenameAndWrapInArray(configBody, "idp_public_cert", "idp_public_certs")
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
	// Create empty config object if it doesn't exist (EnsureAttribute doesn't work for objects)
	if body.GetAttribute("config") == nil {
		body.SetAttributeRaw("config", hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{}))
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
	// Use TransformFieldArrayToObject helper to handle the array-to-object transformation
	options := state.ArrayToObjectOptions{
		SkipFields: []string{"api_token"}, // Remove deprecated field
		RenameFields: map[string]string{
			"idp_public_cert": "idp_public_certs", // Rename field
		},
		FieldTransforms: map[string]func(gjson.Result) interface{}{
			// Note: FieldTransforms uses the NEW field name (after renaming)
			"idp_public_certs": func(value gjson.Result) interface{} {
				// Transform string to array
				if value.Type == gjson.String && value.String() != "" {
					return []string{value.String()}
				}
				return []string{}
			},
		},
		EnsureObjectExists: true, // Config is required in v5, ensure it exists as an object
	}

	result = state.TransformFieldArrayToObject(result, "attributes", attrs, "config", options)

	return result
}

// transformScimConfigField unwraps the scim_config array and handles field transformations
func (m *V4ToV5Migrator) transformScimConfigField(result string, attrs gjson.Result) string {
	// Use TransformFieldArrayToObject helper to handle the array-to-object transformation
	options := state.ArrayToObjectOptions{
		SkipFields: []string{"group_member_deprovision"}, // Remove deprecated field
	}

	result = state.TransformFieldArrayToObject(result, "attributes", attrs, "scim_config", options)

	return result
}
