package authenticated_origin_pulls

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

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
// Note: This returns the default type for resources without hostname.
// Resources with hostname stay as cloudflare_authenticated_origin_pulls (handled in TransformConfig).
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_authenticated_origin_pulls_settings"
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
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_authenticated_origin_pulls", "cloudflare_authenticated_origin_pulls_settings"
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

// TransformState is disabled - state transformation is handled by provider StateUpgraders.
// This method is a no-op and returns the state unchanged.
// Users should use `terraform state mv` or Terraform 1.8+ `moved` blocks for resource renaming,
// which will trigger the provider's StateUpgrader to handle the schema transformation.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// Return state unchanged - provider StateUpgraders will handle transformation
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}

// updateCertificateReference updates certificate references from
// cloudflare_authenticated_origin_pulls_certificate to
// cloudflare_authenticated_origin_pulls_hostname_certificate
// when the referenced certificate has type="per-hostname"
func updateCertificateReference(certTokens hclwrite.Tokens, ctx *transform.Context) hclwrite.Tokens {
	// Convert tokens to string to check if it's a resource reference
	certRefStr := strings.TrimSpace(string(certTokens.Bytes()))

	// Check if this is a reference to cloudflare_authenticated_origin_pulls_certificate
	if !strings.Contains(certRefStr, "cloudflare_authenticated_origin_pulls_certificate.") {
		return certTokens // Not a reference to the cert resource, return unchanged
	}

	// Extract the resource name from the reference
	// Format: cloudflare_authenticated_origin_pulls_certificate.resource_name.id
	// or: cloudflare_authenticated_origin_pulls_certificate.resource_name
	parts := strings.Split(certRefStr, ".")
	if len(parts) < 2 {
		return certTokens // Invalid format, return unchanged
	}

	resourceName := parts[1]

	// Check if this certificate resource has type="per-hostname" by looking at the state
	isPerHostname := false

	if ctx.StateJSON != "" {
		// Parse state to find the certificate resource
		state := gjson.Parse(ctx.StateJSON)
		state.Get("resources").ForEach(func(_, resource gjson.Result) bool {
			if resource.Get("type").String() == "cloudflare_authenticated_origin_pulls_certificate" &&
				resource.Get("name").String() == resourceName {
				// Found the resource - check type attribute in state
				certType := resource.Get("instances.0.attributes.type").String()
				if certType == "per-hostname" {
					isPerHostname = true
				}
				return false // Stop iteration
			}
			return true
		})
	}

	// If we couldn't determine from state, check if the resource name contains "hostname"
	// This is a fallback heuristic matching the certificate migrator's Postprocess logic
	if !isPerHostname {
		lowerName := strings.ToLower(resourceName)
		if strings.Contains(lowerName, "hostname") || strings.Contains(lowerName, "host") {
			isPerHostname = true
		}
	}

	// Update the reference if it's a per-hostname certificate
	if isPerHostname {
		newRefStr := strings.Replace(certRefStr,
			"cloudflare_authenticated_origin_pulls_certificate.",
			"cloudflare_authenticated_origin_pulls_hostname_certificate.",
			1)

		// Parse the updated string back to tokens
		// We need to create new tokens from the updated reference
		newTokens := hclwrite.Tokens{
			{Type: hclsyntax.TokenIdent, Bytes: []byte(newRefStr)},
		}
		return newTokens
	}

	return certTokens
}
