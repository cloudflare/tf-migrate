package logpush_job

import (
	"github.com/hashicorp/hcl/v2"

	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name - same as v5 in this case
	internal.RegisterMigrator("cloudflare_logpush_job", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_logpush_job"}, "cloudflare_logpush_job"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Convert output_options block to attribute (block → attribute syntax)
	// This handles: output_options { ... } → output_options = { ... }
	if outputBlock := tfhcl.FindBlockByType(body, "output_options"); outputBlock != nil {
		outputBody := outputBlock.Body()

		// Rename cve20214428 → cve_2021_44228 BEFORE conversion
		tfhcl.RenameAttribute(outputBody, "cve20214428", "cve_2021_44228")

		// Add v4 schema defaults if not already present (to preserve v4 behavior in v5)
		// v5 does not have defaults for these fields, so we must make them explicit
		m.ensureV4SchemaDefaults(outputBody)

		tfhcl.ConvertSingleBlockToAttribute(body, "output_options", "output_options")
	}

	// 2. Handle kind = "instant-logs" → remove attribute
	// "instant-logs" is no longer valid in v5, remove the attribute entirely
	if kindAttr := body.GetAttribute("kind"); kindAttr != nil {
		kindValue := tfhcl.ExtractStringFromAttribute(kindAttr)
		if kindValue == "instant-logs" {
			// Remove the attribute entirely since instant-logs is not valid in v5
			body.RemoveAttribute("kind")

			// Removing kind turns an instant-logs job into an ordinary logpush
			// job, which is a different thing: it needs a destination and will
			// be created rather than adopted.
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary: fmt.Sprintf(
					"kind = \"instant-logs\" removed from cloudflare_logpush_job.%s",
					tfhcl.GetResourceName(block)),
				Detail: "v5 has no instant-logs job kind, so the kind attribute has been " +
					"removed.\n\n" +
					"What remains is an ordinary logpush job, not an instant-logs " +
					"session, so review this resource before applying: it needs a valid " +
					"destination_conf, and the original instant-logs job is not managed " +
					"by it.",
			})
		}
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// ensureV4SchemaDefaults adds v4 schema defaults to output_options if not present
// This preserves v4 behavior in v5, which has no defaults for these fields
func (m *V4ToV5Migrator) ensureV4SchemaDefaults(body *hclwrite.Body) {
	// Use a slice to ensure deterministic ordering of defaults
	type defaultPair struct {
		field string
		value interface{}
	}

	v4Defaults := []defaultPair{
		{"cve_2021_44228", false},
		{"field_delimiter", ","},
		{"record_prefix", "{"},
		{"record_suffix", "}\n"},
		{"timestamp_format", "unixnano"},
		{"sample_rate", 1},
	}

	for _, pair := range v4Defaults {
		if body.GetAttribute(pair.field) == nil {
			// Field not present, add the v4 default
			tokens := tfhcl.TokensForSimpleValue(pair.value)
			if tokens != nil {
				body.SetAttributeRaw(pair.field, tokens)
			}
		}
	}
}
