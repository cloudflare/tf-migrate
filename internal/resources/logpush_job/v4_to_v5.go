package logpush_job

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

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
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_logpush_job", "cloudflare_logpush_job"
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

// TransformState is a no-op for logpush_job migration.
// State transformation is handled by the provider's StateUpgraders (UpgradeState).
// The provider's migration logic automatically transforms:
// - output_options array → object
// - cve20214428 → cve_2021_44228 field rename
// - Empty strings → null (filter, logpull_options, name)
// - Zero values → null (max_upload_*)
// - kind="instant-logs" → removed
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider handles state transformation, not tf-migrate.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
