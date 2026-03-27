package leaked_credential_check_rule

import (
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

	// If the block has no id attribute the v4 bug may be present — warn the user.
	if body.GetAttribute("id") == nil {
		tfhcl.AppendWarningComment(body,
			"The v4 provider had a bug where the detection_id was not stored in state (id = \"\"). "+
				"If this rule was already created in v4, Terraform will fail to recreate it with "+
				"error 11003 on the first apply. Find the existing detection_id with: "+
				"curl -s \"https://api.cloudflare.com/client/v4/zones/<zone_id>/leaked-credential-checks/detections\" "+
				"-H \"Authorization: Bearer <token>\" | jq '.result[].id' "+
				"Then import: terraform import cloudflare_leaked_credential_check_rule."+resourceName+" <zone_id>/<detection_id>",
		)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
