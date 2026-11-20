package zone_dnssec

import (
	"fmt"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// V4ToV5Migrator handles the migration of cloudflare_zone_dnssec from v4 to v5.
// This migration requires converting flags and key_tag from TypeInt (v4) to Float64 (v5).
type V4ToV5Migrator struct {
}

// NewV4ToV5Migrator creates a new migrator for cloudflare_zone_dnssec v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register the migrator with the internal registry
	internal.RegisterMigrator("cloudflare_zone_dnssec", "v4", "v5", migrator)

	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone_dnssec"
}

// CanHandle determines if this migrator can handle the given resource type.
// Returns true for cloudflare_zone_dnssec (resource name is the same in v4 and v5).
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone_dnssec"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing is needed for this migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_zone_dnssec"}, "cloudflare_zone_dnssec"
}

// TransformConfig handles configuration file transformations.
//  1. Sets status = "active" when absent. status was computed-only in v4, where
//     the resource always enabled DNSSEC. In v5 it is optional and not computed,
//     so leaving it out makes every plan show status = "active" -> null.
//  2. Removes modified_on attribute if present (changed from optional+computed to computed-only in v5)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	if body.GetAttribute("status") == nil {
		body.SetAttributeValue("status", cty.StringVal("active"))
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"status = \"active\" added to cloudflare_zone_dnssec.%s", tfhcl.GetResourceName(block)),
			Detail: "v4 always enabled DNSSEC and status was computed-only. In v5 status is " +
				"set in config, so tf-migrate wrote \"active\" to match v4.\n\n" +
				"If DNSSEC was turned off outside Terraform, the first plan shows " +
				"status = \"disabled\" -> \"active\" and applying turns DNSSEC back on. " +
				"Set status = \"disabled\" to keep it off.",
		})
	}

	// Remove modified_on attribute if present (changed from optional+computed to computed-only)
	// RemoveAttribute is safe to call even if the attribute doesn't exist
	body.RemoveAttribute("modified_on")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
