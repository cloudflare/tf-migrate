package zero_trust_access_identity_provider

import (
	"github.com/hashicorp/hcl/v2"

	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"

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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_access_identity_provider", "cloudflare_zero_trust_access_identity_provider"}, "cloudflare_zero_trust_access_identity_provider"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// CRITICAL: Capture resource name and original type BEFORE any modifications
	// This is required for correct moved block generation
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)
	originalType := block.Labels()[0]
	needsMovedBlock := originalType == m.oldType

	// Rename cloudflare_access_identity_provider to cloudflare_zero_trust_access_identity_provider
	tfhcl.RenameResourceType(block, m.oldType, m.newType)

	body := block.Body()

	// 1. Convert config block to attribute (with preprocessing).
	//
	// v4 accepted repeated config blocks, but the provider read config.0 and
	// nothing else, so any later block never reached the API. Keep the first
	// to preserve behaviour: keeping the last would activate settings that
	// were previously inert.
	droppedConfig := tfhcl.ConvertBlocksToAttributeKeepFirst(body, "config", "config", func(configBlock *hclwrite.Block) {
		configBody := configBlock.Body()

		// Remove deprecated api_token field
		tfhcl.RemoveAttributes(configBody, "api_token")

		// Rename idp_public_cert to idp_public_certs and wrap in array
		tfhcl.RenameAndWrapInArray(configBody, "idp_public_cert", "idp_public_certs")
	})

	// 2. Convert scim_config block to attribute (with preprocessing).
	// Same as config: v4 only ever read scim_config.0.
	droppedScim := tfhcl.ConvertBlocksToAttributeKeepFirst(body, "scim_config", "scim_config", func(scimBlock *hclwrite.Block) {
		scimBody := scimBlock.Body()

		// Remove deprecated group_member_deprovision field
		tfhcl.RemoveAttributes(scimBody, "group_member_deprovision")

		// Remove secret field (it's now computed-only in v5, not user-settable)
		tfhcl.RemoveAttributes(scimBody, "secret")
	})

	for _, dropped := range []struct {
		blockType string
		count     int
	}{
		{"config", droppedConfig},
		{"scim_config", droppedScim},
	} {
		if dropped.count == 0 {
			continue
		}
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"Extra %s block(s) dropped from cloudflare_zero_trust_access_identity_provider.%s",
				dropped.blockType, tfhcl.GetResourceName(block)),
			Detail: fmt.Sprintf(
				"%d additional %s block(s) were present. v5 accepts a single %s object, "+
					"and the first block has been kept.\n\n"+
					"This does not change the identity provider's behaviour: the v4 "+
					"provider read only the first %s block, so the others were never "+
					"sent to the API.\n\n"+
					"Confirm the first block is the one you intended, since the others "+
					"are now gone from the configuration.",
				dropped.count, dropped.blockType, dropped.blockType, dropped.blockType),
		})
	}

	// 3. Ensure config attribute exists (required in v5)
	if body.GetAttribute("config") == nil {
		body.SetAttributeRaw("config", hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{}))
	}

	// Build result blocks
	resultBlocks := []*hclwrite.Block{block}

	// Generate moved block for state migration (only when renaming from old type)
	if needsMovedBlock {
		_, newType := m.GetResourceRename()
		from := originalResourceType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)
		resultBlocks = append(resultBlocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         resultBlocks,
		RemoveOriginal: true,
	}, nil
}
