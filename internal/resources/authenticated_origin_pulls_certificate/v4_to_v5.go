package authenticated_origin_pulls_certificate

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of authenticated origin pulls certificate resources from v4 to v5
// The v4 cloudflare_authenticated_origin_pulls_certificate resource with a type field
// is split into two separate resources in v5:
// - cloudflare_authenticated_origin_pulls_certificate (for type="per-zone")
// - cloudflare_authenticated_origin_pulls_hostname_certificate (for type="per-hostname")
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_authenticated_origin_pulls_certificate", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Returns empty string because this resource routes to TWO different types based on type field:
	// - type="per-hostname" → cloudflare_authenticated_origin_pulls_hostname_certificate (via moved blocks in TransformConfig)
	// - type="per-zone" (or default) → cloudflare_authenticated_origin_pulls_certificate (no type change)
	// The actual type is determined dynamically in TransformConfig based on config attributes.
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_authenticated_origin_pulls_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	// No longer used — reference updates are handled via GetResourceRenameForName
	// which tracks per-hostname resources during TransformConfig.
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// In v5, all per-hostname certificate references (used in cert_id of
// cloudflare_authenticated_origin_pulls) must point to
// cloudflare_authenticated_origin_pulls_hostname_certificate.
// Per-zone certificates keep the same name but are not referenced via cert_id,
// so renaming all cross-file references to the hostname type is correct.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_authenticated_origin_pulls_certificate"}, "cloudflare_authenticated_origin_pulls_hostname_certificate"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Determine target resource type by reading from config
	var targetType string
	var typeFromState string

	typeAttr := body.GetAttribute("type")
	if typeAttr != nil {
		typeFromState = tfhcl.ExtractStringFromAttribute(typeAttr)
	}

	// Determine target type based on type value
	if typeFromState == "per-hostname" {
		targetType = "cloudflare_authenticated_origin_pulls_hostname_certificate"
	} else {
		// Default to per-zone (includes "per-zone", empty, or any other value)
		targetType = "cloudflare_authenticated_origin_pulls_certificate"
	}

	// Rename the resource type
	tfhcl.RenameResourceType(block,
		"cloudflare_authenticated_origin_pulls_certificate",
		targetType)

	// Remove type field from v5 configuration (not present in v5 schemas)
	tfhcl.RemoveAttributes(body, "type")

	// Generate moved block for per-hostname resources (per-zone keeps same name)
	var blocks []*hclwrite.Block
	blocks = append(blocks, block)

	if typeFromState == "per-hostname" {
		movedBlock := tfhcl.CreateMovedBlock(
			"cloudflare_authenticated_origin_pulls_certificate."+resourceName,
			"cloudflare_authenticated_origin_pulls_hostname_certificate."+resourceName,
		)
		blocks = append(blocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: true, // Must be true for blocks to be added
	}, nil
}
