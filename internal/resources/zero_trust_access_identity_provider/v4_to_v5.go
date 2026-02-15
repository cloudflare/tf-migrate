package zero_trust_access_identity_provider

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

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
	// Register BOTH v4 resource names (deprecated and current)
	internal.RegisterMigrator("cloudflare_access_identity_provider", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_identity_provider", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType || resourceType == m.newType
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
	// CRITICAL: Capture resource name and original type BEFORE any modifications
	// This is required for correct moved block generation
	resourceName := tfhcl.GetResourceName(block)
	originalType := block.Labels()[0]
	needsMovedBlock := originalType == m.oldType

	// Rename cloudflare_access_identity_provider to cloudflare_zero_trust_access_identity_provider
	tfhcl.RenameResourceType(block, m.oldType, m.newType)

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
	if body.GetAttribute("config") == nil {
		body.SetAttributeRaw("config", hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{}))
	}

	// Build result blocks
	resultBlocks := []*hclwrite.Block{block}

	// Generate moved block for state migration (only when renaming from old type)
	if needsMovedBlock {
		oldType, newType := m.GetResourceRename()
		from := oldType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)
		resultBlocks = append(resultBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         resultBlocks,
		RemoveOriginal: true,
	}, nil
}

// TransformState is a no-op - state transformation is handled by the provider's StateUpgraders.
// The moved block generated in TransformConfig triggers the provider's migration logic.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
