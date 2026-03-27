package leaked_credential_check_rule

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_leaked_credential_check_rule from v4 to v5.
// This is a pass-through migration as the resource name and all fields remain unchanged between v4 and v5.
// The only changes are validation-level (username and password changed from Required to Optional in v5).
type V4ToV5Migrator struct {
}

// NewV4ToV5Migrator creates a new migrator for cloudflare_leaked_credential_check_rule v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register the migrator with the internal registry
	// Resource name is identical in v4 and v5
	internal.RegisterMigrator("cloudflare_leaked_credential_check_rule", "v4", "v5", migrator)

	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_leaked_credential_check_rule"
}

// CanHandle determines if this migrator can handle the given resource type.
// Returns true for cloudflare_leaked_credential_check_rule (resource name is the same in v4 and v5).
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_leaked_credential_check_rule"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing is needed for this migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// This resource does not rename, so we return the same name for both old and new.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_leaked_credential_check_rule"}, "cloudflare_leaked_credential_check_rule"
}

// TransformConfig handles configuration file transformations.
// The v4 provider had a bug where it created the detection rule successfully
// but failed to store the returned detection_id in state (id = ""). The v5
// provider's Read() detects the empty ID and removes the resource from state,
// causing Terraform to attempt a re-create on the next apply — which fails
// with error 11003 "custom detection for given username and password already
// exists" if the rule was actually created by the v4 provider.
//
// We emit a MIGRATION WARNING comment so users know they may need to manually
// import the existing rule.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := block.Labels()[1]

	// If the block has no id attribute the v4 bug may be present.
	// Generate an import {} block with placeholder IDs so the user can fill
	// them in without any manual HCL editing. Also emit a terminal diagnostic
	// with the curl command to find the real detection_id.
	if body.GetAttribute("id") == nil {
		// Extract zone_id if it's a literal 32-char hex string; otherwise keep placeholder.
		zoneID := "<zone_id>"
		if zoneAttr := body.GetAttribute("zone_id"); zoneAttr != nil {
			extracted := tfhcl.ExtractStringFromAttribute(zoneAttr)
			if len(extracted) == 32 {
				zoneID = extracted
			}
		}

		importID := fmt.Sprintf("%s/<detection_id>", zoneID)
		listCmd := fmt.Sprintf(
			`curl -s "https://api.cloudflare.com/client/v4/zones/%s/leaked-credential-checks/detections" -H "Authorization: Bearer <token>" | jq '.result[] | {id, username, password}'`,
			zoneID,
		)

		// Generate import {} block with placeholder ID so the user just needs
		// to replace <detection_id> (and <zone_id> if not resolved above).
		importBlock := tfhcl.CreateImportBlock(
			"cloudflare_leaked_credential_check_rule",
			resourceName,
			importID,
		)

		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Action required: fill in detection_id for cloudflare_leaked_credential_check_rule.%s", resourceName),
			Detail: "The v4 provider had a bug where the detection_id was not stored in state.\n" +
				"An import {} block has been generated in:\n" +
				"  " + ctx.FilePath + "\n\n" +
				"Before running terraform plan/apply:\n" +
				"  1. Find the detection_id:\n" +
				"       " + listCmd + "\n\n" +
				"  2. Replace <detection_id> in the import block with the real ID.\n" +
				"     The import block looks like:\n" +
				"       import {\n" +
				"         to = cloudflare_leaked_credential_check_rule." + resourceName + "\n" +
				"         id = \"" + importID + "\"\n" +
				"       }",
		})

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{importBlock, block},
			RemoveOriginal: true, // Must be true for extra blocks (import block) to be written
		}, nil
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
