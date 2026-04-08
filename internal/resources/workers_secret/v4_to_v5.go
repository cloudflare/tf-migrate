package workers_secret

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of worker secrets from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names:
	//   - cloudflare_worker_secret: singular form (requires type rename + moved block)
	//   - cloudflare_workers_secret: plural form (in-place attr rename only)
	internal.RegisterMigrator("cloudflare_worker_secret", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_workers_secret", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_workers_secret"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the singular and plural v4 names
	return resourceType == "cloudflare_worker_secret" || resourceType == "cloudflare_workers_secret"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_worker_secret", "cloudflare_workers_secret"}, "cloudflare_workers_secret"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)
	body := block.Body()

	// Check if this is the singular form (needs rename + moved block)
	wasSingular := resourceType == "cloudflare_worker_secret"

	// Rename resource type if using singular form
	if wasSingular {
		tfhcl.RenameResourceType(block, "cloudflare_worker_secret", "cloudflare_workers_secret")
	}

	// Rename attribute: secret → secret_text
	// v4 used "secret" attribute, v5 uses "secret_text"
	tfhcl.RenameAttribute(body, "secret", "secret_text")

	// Check if account_id is missing - add warning if so
	// v5 requires account_id, v4 didn't
	if body.GetAttribute("account_id") == nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Missing required field: account_id",
			Detail: `The 'account_id' field is required in v5 but was not present in v4.

Please add the account_id attribute to avoid plan errors:
  account_id = "your-account-id"`,
		})

		// Add a warning comment in the HCL
		tfhcl.AppendWarningComment(body, "MIGRATION REQUIRED: Add account_id attribute (required in v5)")
	}

	// Generate moved block if the resource was renamed (singular → plural)
	if wasSingular {
		from := "cloudflare_worker_secret." + resourceName
		to := "cloudflare_workers_secret." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	// Plural form - no moved block needed, just attribute rename
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
