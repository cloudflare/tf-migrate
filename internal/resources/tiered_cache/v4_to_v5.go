package tiered_cache

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// V4ToV5Migrator handles the migration of cloudflare_tiered_cache from v4 to v5.
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_tiered_cache v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name
	internal.RegisterMigrator("cloudflare_tiered_cache", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_tiered_cache"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_tiered_cache"
}

// Preprocess handles string-level transformations before HCL parsing.
// This transforms cache_type values to prepare for HCL transformation.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_tiered_cache doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_tiered_cache"}, "cloudflare_tiered_cache"
}

// TransformConfig handles configuration file transformations.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Idempotency guard: if cache_type is absent the block has already been
	// migrated (it uses the v5 "value" attribute). Return the block unchanged
	// so re-running tf-migrate does not duplicate resources.
	if body.GetAttribute("cache_type") == nil {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	blocks := make([]*hclwrite.Block, 0)

	// rename cache_type to value in the original block
	tfhcl.RenameAttribute(body, "cache_type", "value")

	resourceName := tfhcl.GetResourceName(block)
	valueAttr := body.GetAttribute("value")

	// Get the literal value (variable references cannot be resolved without state)
	value := tfhcl.ExtractStringFromAttribute(valueAttr)

	if value == "smart" {
		// cache_type="smart" → value="on" for both resources
		// Create tiered_cache resource with value="on"
		tfhcl.SetAttribute(body, "value", "on")
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)

		// Create argo_tiered_caching resource with value="on"
		argoBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "on"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, argoBlock)
		blocks = append(blocks, m.argoImportBlock(ctx, body, resourceName))
	} else if value == "off" {
		// cache_type="off" → value="off" for both resources
		// Create tiered_cache resource with value="off"
		tfhcl.SetAttribute(body, "value", "off")
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)

		// Create argo_tiered_caching resource with value="off"
		argoBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "off"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, argoBlock)
		blocks = append(blocks, m.argoImportBlock(ctx, body, resourceName))
	} else if value == "generic" {
		// cache_type="generic" → tiered_cache value="off", argo_tiered_caching value="on"
		// Create tiered_cache resource with value="off"
		tfhcl.SetAttribute(body, "value", "off")
		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)

		// Create argo_tiered_caching resource with value="on"
		argoBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_argo_tiered_caching",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id"},
				Set:               map[string]interface{}{"value": "on"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, argoBlock)
		blocks = append(blocks, m.argoImportBlock(ctx, body, resourceName))
	} else {
		// cache_type is an expression, so its value cannot be resolved without
		// state and the resource cannot be split. Only cloudflare_tiered_cache
		// is produced; the matching cloudflare_argo_tiered_caching is not.
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"Action required: tiered cache not split for cloudflare_tiered_cache.%s", resourceName),
			Detail: fmt.Sprintf(
				"In v5 cloudflare_tiered_cache covers only smart tiered caching; generic "+
					"tiered caching moves to cloudflare_argo_tiered_caching, so a v4 "+
					"resource normally becomes two.\n\n"+
					"Here cache_type is an expression rather than a literal, so its value "+
					"cannot be resolved and the split cannot be made automatically. Only "+
					"cloudflare_tiered_cache.%s has been written.\n\n"+
					"If the expression can evaluate to \"generic\" or \"smart\", add the "+
					"second resource and an import block for it by hand:\n\n"+
					"  resource \"cloudflare_argo_tiered_caching\" \"%s\" {\n"+
					"    zone_id = <zone_id>\n"+
					"    value   = \"on\"\n"+
					"  }\n\n"+
					"Without it, generic tiered caching stays at whatever the API currently "+
					"has and is no longer tracked by Terraform.",
				resourceName, resourceName),
		})

		tieredCacheBlock := tfhcl.CreateDerivedBlock(
			block,
			"cloudflare_tiered_cache",
			resourceName,
			tfhcl.AttributeTransform{
				Copy:              []string{"zone_id", "value"},
				CopyMetaArguments: true,
			},
		)
		blocks = append(blocks, tieredCacheBlock)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: true,
	}, nil
}

// argoImportBlock builds the import block for the cloudflare_argo_tiered_caching
// resource derived from a v4 cloudflare_tiered_cache.
//
// The derived resource is new to the configuration but already exists at the
// API, so without an import block the first apply would try to create it.
func (m *V4ToV5Migrator) argoImportBlock(ctx *transform.Context, body *hclwrite.Body, resourceName string) *hclwrite.Block {
	// Resolve zone_id when it is a literal so the user has nothing to fill in.
	zoneID := "<zone_id>"
	if zoneAttr := body.GetAttribute("zone_id"); zoneAttr != nil {
		if extracted := tfhcl.ExtractStringFromAttribute(zoneAttr); len(extracted) == 32 {
			zoneID = extracted
		}
	}

	if zoneID == "<zone_id>" {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf(
				"Action required: complete the import block for cloudflare_argo_tiered_caching.%s",
				resourceName),
			Detail: fmt.Sprintf(
				"zone_id is an expression, so it could not be resolved into the generated "+
					"import block for cloudflare_argo_tiered_caching.%s.\n\n"+
					"Replace the <zone_id> placeholder with the zone ID before running "+
					"terraform apply, or the apply will fail.",
				resourceName),
		})
	}

	return tfhcl.CreateImportBlock("cloudflare_argo_tiered_caching", resourceName, zoneID)
}
