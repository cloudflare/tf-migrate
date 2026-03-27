package zero_trust_access_mtls_certificate

import (
	"fmt"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register both the deprecated v4 name and the intermediate v5 name
	internal.RegisterMigrator("cloudflare_access_mutual_tls_certificate", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_mtls_certificate", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the current name and the deprecated v4 name
	return resourceType == "cloudflare_access_mutual_tls_certificate" ||
		resourceType == "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations can be done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_access_mutual_tls_certificate", "cloudflare_zero_trust_access_mtls_certificate"}, "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture resource name and original type BEFORE any modifications
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)
	originalType := block.Labels()[0]
	needsMovedBlock := originalType == "cloudflare_access_mutual_tls_certificate"

	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_access_mutual_tls_certificate", "cloudflare_zero_trust_access_mtls_certificate")

	// Check for required certificate attribute - it's write-only so we can't auto-populate
	// If missing, add a placeholder value and lifecycle ignore block since the resource already exists
	body := block.Body()
	if body.GetAttribute("certificate") == nil {
		// Add placeholder certificate value - this won't be used since we ignore changes
		body.SetAttributeValue("certificate", cty.StringVal("PLACEHOLDER - actual certificate already deployed"))

		// Add or update lifecycle block to ignore certificate changes
		tfhcl.AddLifecycleIgnoreChanges(body, "certificate")

		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Added placeholder for write-only 'certificate' in cloudflare_zero_trust_access_mtls_certificate.%s", resourceName),
			Detail: `The 'certificate' attribute is required in v5 but was not found in your configuration.
A placeholder value has been added with lifecycle { ignore_changes = [certificate] }.
The actual certificate is already deployed in Cloudflare and won't be modified.`,
		})
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

