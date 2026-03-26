package authenticated_origin_pulls

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// certMigratorLookup is a local interface satisfied by the
// authenticated_origin_pulls_certificate migrator. Using an interface avoids
// an import cycle between the two sibling packages.
type certMigratorLookup interface {
	IsPerHostname(resourceName string) bool
}

// V4ToV5Migrator handles migration of Authenticated Origin Pulls from v4 to v5
// The v4 resource handled three modes in one resource:
// 1. Global AOP: zone_id + enabled
// 2. Per-Zone AOP: zone_id + authenticated_origin_pulls_certificate + enabled
// 3. Per-Hostname AOP: zone_id + hostname + authenticated_origin_pulls_certificate + enabled
//
// In v5, this is split into TWO resources:
// - cloudflare_authenticated_origin_pulls_settings: Handles modes 1 & 2 (Global/Per-Zone)
// - cloudflare_authenticated_origin_pulls: Handles mode 3 (Per-Hostname)
//
// Migration routing:
// - Resources WITHOUT hostname → cloudflare_authenticated_origin_pulls_settings
// - Resources WITH hostname → cloudflare_authenticated_origin_pulls (restructured to nested config)
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_authenticated_origin_pulls", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the resource type this migrator handles (v5 name).
// Returns empty string because this resource routes to TWO different types based on hostname presence:
// - WITHOUT hostname → cloudflare_authenticated_origin_pulls_settings (via moved blocks in TransformConfig)
// - WITH hostname → cloudflare_authenticated_origin_pulls (no type change)
// The actual type is determined dynamically in TransformConfig based on configuration attributes.
func (m *V4ToV5Migrator) GetResourceType() string {
	return ""
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_authenticated_origin_pulls"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for authenticated_origin_pulls migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// Note: This returns the default rename for resources without hostname.
// Resources with hostname don't rename (handled conditionally in TransformConfig).
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_authenticated_origin_pulls"}, "cloudflare_authenticated_origin_pulls_settings"
}

// TransformConfig handles configuration file transformations.
// Routes resources based on hostname presence:
// - WITHOUT hostname → cloudflare_authenticated_origin_pulls_settings (Global/Per-Zone AOP)
// - WITH hostname → cloudflare_authenticated_origin_pulls (Per-Hostname AOP, restructured)
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Get the resource name before any modifications
	resourceName := block.Labels()[1]

	// Check if hostname attribute exists
	hasHostname := body.GetAttribute("hostname") != nil

	if hasHostname {
		// Per-Hostname AOP: Keep as cloudflare_authenticated_origin_pulls
		// Restructure from flat to nested config

		// Get attribute values before transformation
		hostname := body.GetAttribute("hostname")
		cert := body.GetAttribute("authenticated_origin_pulls_certificate")
		enabled := body.GetAttribute("enabled")

		// Remove old flat attributes
		tfhcl.RemoveAttributes(body, "hostname", "authenticated_origin_pulls_certificate", "enabled")

		// Create nested config attribute (list of objects)
		// v5 expects: config = [{ hostname = "..." cert_id = "..." enabled = true }]
		// Build a temporary block to construct the object, then convert to tokens
		configBlock := hclwrite.NewBlock("temp", nil)
		configBody := configBlock.Body()

		if hostname != nil {
			configBody.SetAttributeRaw("hostname", hostname.Expr().BuildTokens(nil))
		}
		if cert != nil {
			// Get the certificate reference tokens
			certTokens := cert.Expr().BuildTokens(nil)

			// Update certificate reference if it points to a per-hostname certificate
			// that will be migrated to cloudflare_authenticated_origin_pulls_hostname_certificate
			certTokens = updateCertificateReference(certTokens, ctx)

			configBody.SetAttributeRaw("cert_id", certTokens)
		}
		if enabled != nil {
			configBody.SetAttributeRaw("enabled", enabled.Expr().BuildTokens(nil))
		}

		// Convert block body to object tokens
		objTokens := tfhcl.BuildObjectFromBlock(configBlock)

		// Wrap in list (tuple)
		listTokens := hclwrite.TokensForTuple([]hclwrite.Tokens{objTokens})

		// Set as attribute (not block)
		body.SetAttributeRaw("config", listTokens)

		// No resource rename needed (stays cloudflare_authenticated_origin_pulls)
		// No moved block needed (resource type unchanged)
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: true,
		}, nil
	} else {
		// Global/Per-Zone AOP: Rename to cloudflare_authenticated_origin_pulls_settings

		// Rename the resource type
		tfhcl.RenameResourceType(block, "cloudflare_authenticated_origin_pulls", "cloudflare_authenticated_origin_pulls_settings")

		// Remove authenticated_origin_pulls_certificate if present (Per-Zone mode)
		tfhcl.RemoveAttributes(body, "authenticated_origin_pulls_certificate")

		// Generate moved block for Terraform 1.8+ automatic state migration
		fromRef := "cloudflare_authenticated_origin_pulls." + resourceName
		toRef := "cloudflare_authenticated_origin_pulls_settings." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(fromRef, toRef)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}
}

// updateCertificateReference updates certificate references from
// cloudflare_authenticated_origin_pulls_certificate to
// cloudflare_authenticated_origin_pulls_hostname_certificate
// when the referenced certificate was migrated from type="per-hostname".
//
// It looks up the cert migrator via the registry to check whether the referenced
// resource name was recorded as per-hostname during its own TransformConfig call.
// This avoids the previous name-based heuristic which was unreliable and also
// avoids the over-eager blanket rename that GetResourceRename() caused.
func updateCertificateReference(certTokens hclwrite.Tokens, ctx *transform.Context) hclwrite.Tokens {
	// Convert tokens to string to check if it's a resource reference
	certRefStr := strings.TrimSpace(string(certTokens.Bytes()))

	// Check if this is a reference to cloudflare_authenticated_origin_pulls_certificate
	if !strings.Contains(certRefStr, "cloudflare_authenticated_origin_pulls_certificate.") {
		return certTokens // Not a reference to the cert resource, return unchanged
	}

	// Extract the resource name from the reference
	// Format: cloudflare_authenticated_origin_pulls_certificate.resource_name.id
	// or with for_each: cloudflare_authenticated_origin_pulls_certificate.resource_name[each.key].id
	after := strings.TrimPrefix(certRefStr, "cloudflare_authenticated_origin_pulls_certificate.")
	// resource name is the next dot-separated segment (may be followed by . or [)
	resourceName := after
	if idx := strings.IndexAny(after, ".["); idx >= 0 {
		resourceName = after[:idx]
	}
	if resourceName == "" {
		return certTokens
	}

	// Ask the cert migrator whether this resource was per-hostname.
	// We use a local interface to avoid an import cycle between sibling packages.
	certMigrator := internal.GetMigrator("cloudflare_authenticated_origin_pulls_certificate", "v4", "v5")
	if certMigrator == nil {
		return certTokens
	}
	lookup, ok := certMigrator.(certMigratorLookup)
	if !ok {
		return certTokens
	}

	if !lookup.IsPerHostname(resourceName) {
		return certTokens
	}

	// Replace only the type prefix, preserving the rest of the reference expression
	newRefStr := strings.Replace(certRefStr,
		"cloudflare_authenticated_origin_pulls_certificate.",
		"cloudflare_authenticated_origin_pulls_hostname_certificate.",
		1)

	newTokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(newRefStr)},
	}
	return newTokens
}
