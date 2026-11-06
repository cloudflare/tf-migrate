package managed_transforms

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the OLD (v4) resource name
	internal.RegisterMigrator("cloudflare_managed_headers", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_managed_headers"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_managed_headers"}, "cloudflare_managed_transforms"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original resource type before any modifications (for moved block generation)
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_managed_headers", "cloudflare_managed_transforms")

	body := block.Body()

	// v5 requires these fields, so an absent one is written as an empty array.
	// A dynamic block must be converted to a for expression first: it is not
	// matched by the static block conversion, and leaving it in place would
	// pair it with an empty array that declares "no managed headers".
	var unconverted []string
	for _, blockType := range []string{"managed_request_headers", "managed_response_headers"} {
		converted := false

		if dynBlock := tfhcl.FindDynamicBlockFor(body, blockType); dynBlock != nil {
			switch {
			case len(tfhcl.FindBlocksByType(body, blockType)) > 0:
				// Splicing a for expression into a literal list would need
				// concat(); refuse rather than emit something subtly wrong.
				unconverted = append(unconverted,
					blockType+" (it is mixed with static "+blockType+" block(s))")
			default:
				if ok, reason := tfhcl.ConvertDynamicBlockToForList(body, dynBlock, blockType); ok {
					converted = true
				} else {
					unconverted = append(unconverted, blockType+" ("+reason+")")
				}
			}
		}

		// Skip the static conversion once a dynamic block has produced the
		// attribute, or the empty-array default would overwrite it.
		if !converted {
			tfhcl.ConvertBlocksToArrayAttribute(body, blockType, true)
		}
	}

	for _, detail := range unconverted {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"Dynamic block not migrated on cloudflare_managed_transforms.%s", resourceName),
			Detail: fmt.Sprintf(
				"A dynamic block for %s could not be converted automatically.\n\n"+
					"It has been left in place, but v5 expects a list attribute rather "+
					"than repeated blocks, so the configuration will not validate until "+
					"it is rewritten by hand, for example:\n\n"+
					"  managed_request_headers = [for k, v in <for_each> : {\n"+
					"    id      = k\n"+
					"    enabled = v\n"+
					"  }]",
				detail),
		})
	}

	// Generate moved block for state migration if resource type changed
	_, newType := m.GetResourceRename()
	if originalResourceType != newType {
		from := originalResourceType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
