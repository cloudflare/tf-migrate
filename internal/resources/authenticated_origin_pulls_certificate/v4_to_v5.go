package authenticated_origin_pulls_certificate

import (
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

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
	// Return empty string because this resource splits into TWO different types
	// based on the type field value. The actual type is determined dynamically in TransformState.
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
	// Handle cross-file reference updates for resource split
	// We need to update references like:
	// cloudflare_authenticated_origin_pulls_certificate.resource_name
	// to:
	// cloudflare_authenticated_origin_pulls_hostname_certificate.resource_name
	// for resources that were migrated to the hostname type

	// This is a best-effort approach - we look for patterns that suggest
	// a hostname resource based on common naming conventions
	re := regexp.MustCompile(`cloudflare_authenticated_origin_pulls_certificate\.([a-zA-Z0-9_]+)`)

	content = re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the resource name
		parts := strings.Split(match, ".")
		if len(parts) != 2 {
			return match
		}
		resourceName := parts[1]

		// Check if this looks like a hostname resource (contains "hostname" or "host" in name)
		lowerName := strings.ToLower(resourceName)
		if strings.Contains(lowerName, "hostname") || strings.Contains(lowerName, "host") {
			// Update to hostname certificate resource
			return "cloudflare_authenticated_origin_pulls_hostname_certificate." + resourceName
		}

		// Keep as per-zone resource
		return match
	})

	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource splits into two different types based on the type field:
// - per-zone -> cloudflare_authenticated_origin_pulls_certificate
// - per-hostname -> cloudflare_authenticated_origin_pulls_hostname_certificate
// We return the v4 name for both to indicate no simple 1:1 rename
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_authenticated_origin_pulls_certificate", "cloudflare_authenticated_origin_pulls_certificate"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Determine target resource type by reading from state (more reliable than config)
	// Config might have variables or expressions for type field
	var targetType string
	var typeFromState string

	if ctx.StateJSON != "" {
		// Parse state to find this resource
		state := gjson.Parse(ctx.StateJSON)
		state.Get("resources").ForEach(func(_, resource gjson.Result) bool {
			if resource.Get("type").String() == "cloudflare_authenticated_origin_pulls_certificate" &&
				resource.Get("name").String() == resourceName {
				// Found the resource - check type attribute in state
				typeFromState = resource.Get("instances.0.attributes.type").String()
				return false // Stop iteration
			}
			return true
		})
	}

	// If we couldn't determine from state, fall back to config
	if typeFromState == "" {
		typeAttr := body.GetAttribute("type")
		if typeAttr != nil {
			typeFromState = tfhcl.ExtractStringFromAttribute(typeAttr)
		}
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

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// Complete pass-through - all transformations handled by:
	// 1. Moved blocks (generated in TransformConfig) handle resource type routing
	// 2. Provider StateUpgraders handle field transformations and schema version bumps
	//
	// This is the Terraform-native approach: moved blocks + provider state upgraders
	return stateJSON.String(), nil
}
