package certificate_pack

import (
	"fmt"
	"strings"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// V4ToV5Migrator handles migration of certificate_pack resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_certificate_pack", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_certificate_pack doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_certificate_pack"}, "cloudflare_certificate_pack"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Check for deprecated fields and warn
	// Check both attribute syntax (validation_records = [...]) and block syntax (validation_records { })
	var removedFields []string
	if body.GetAttribute("wait_for_active_status") != nil {
		removedFields = append(removedFields, "wait_for_active_status")
	}
	if body.GetAttribute("validation_records") != nil || len(tfhcl.FindBlocksByType(body, "validation_records")) > 0 {
		removedFields = append(removedFields, "validation_records")
	}
	if body.GetAttribute("validation_errors") != nil || len(tfhcl.FindBlocksByType(body, "validation_errors")) > 0 {
		removedFields = append(removedFields, "validation_errors")
	}

	if len(removedFields) > 0 {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: transform.DiagInfo,
			Summary:  fmt.Sprintf("Deprecated fields removed: cloudflare_certificate_pack.%s", resourceName),
			Detail: fmt.Sprintf(`The following fields have been removed during migration:
  %s

These fields are now Computed-only or removed in the v5 provider:
  - wait_for_active_status: No longer needed (provider waits automatically)
  - validation_records: Now Computed-only, cannot be set in config
  - validation_errors: Now Computed-only, cannot be set in config`, strings.Join(removedFields, ", ")),
		})
	}

	// wait_for_active_status: removed in v5
	tfhcl.RemoveAttributes(body, "wait_for_active_status")

	// validation_records, validation_errors: were Optional+Computed in v4, only Computed in v5
	// Remove both attribute syntax and block syntax since v4 supported both
	tfhcl.RemoveAttributes(body, "validation_records", "validation_errors")
	tfhcl.RemoveBlocksByType(body, "validation_records")
	tfhcl.RemoveBlocksByType(body, "validation_errors")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
