package zero_trust_device_posture_rule

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of device posture rule resources from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	internal.RegisterMigrator("cloudflare_device_posture_rule", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_device_posture_rule", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_posture_rule"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_posture_rule" ||
		resourceType == "cloudflare_zero_trust_device_posture_rule"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done at HCL level
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Returns rename from cloudflare_device_posture_rule to cloudflare_zero_trust_device_posture_rule
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_device_posture_rule", "cloudflare_zero_trust_device_posture_rule"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// CRITICAL: Capture resource name and original type BEFORE any modifications
	// This is required for correct moved block generation
	resourceName := tfhcl.GetResourceName(block)
	originalType := block.Labels()[0]
	needsMovedBlock := originalType == "cloudflare_device_posture_rule"

	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_device_posture_rule", "cloudflare_zero_trust_device_posture_rule")

	body := block.Body()

	// Convert optional input block to attribute, handling nested locations attribute
	if inputBlock := tfhcl.FindBlockByType(body, "input"); inputBlock != nil {
		tfhcl.ConvertBlocksToAttribute(body, "input", "input", func(inputBlock *hclwrite.Block) {
			tfhcl.RemoveAttributes(inputBlock.Body(), "running")
			tfhcl.ConvertBlocksToAttribute(inputBlock.Body(), "locations", "locations", func(locationsBlock *hclwrite.Block) {})
		})
	}

	// Convert optional match blocks to attribute array
	if matchBlocks := tfhcl.FindBlocksByType(body, "match"); len(matchBlocks) > 0 {
		tfhcl.MergeAttributeAndBlocksToObjectArray(body, "", "match", "match", "platform", []string{}, true)
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

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
