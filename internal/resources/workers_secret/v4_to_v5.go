package workers_secret

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of worker secrets from v4 to v5
// In v5, cloudflare_workers_secret is removed - secrets are now secret_text bindings
// inside cloudflare_workers_script resources.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names:
	//   - cloudflare_worker_secret: singular form
	//   - cloudflare_workers_secret: plural form
	internal.RegisterMigrator("cloudflare_worker_secret", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_workers_secret", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// In v5, workers_secret doesn't exist as a standalone resource
	// Secrets are now bindings within workers_script
	return "cloudflare_workers_script"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the singular and plural v4 names
	return resourceType == "cloudflare_worker_secret" || resourceType == "cloudflare_workers_secret"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// Note: In v5, workers_secret doesn't exist - it becomes part of workers_script
// This returns empty to indicate no direct rename (cross-resource migration instead)
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	// Return empty to indicate no direct resource rename
	// The migration is handled as cross-resource (secrets -> bindings)
	return nil, ""
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Extract secret information for the warning message
	scriptName := extractAttributeValue(body, "script_name")
	secretName := extractAttributeValue(body, "name")

	// Build a detailed warning message
	var warningDetail string
	if scriptName != "" && secretName != "" {
		warningDetail = fmt.Sprintf(
			"This secret (%s) must be migrated to a secret_text binding in the workers_script resource for '%s'. "+
				"In v5, cloudflare_workers_secret is removed - secrets are now bindings within cloudflare_workers_script. "+
				"Add the following binding to your cloudflare_workers_script resource:\n\n"+
				"  secret_text = {\n"+
				"    name = %q\n"+
				"    text = <secret_value>\n"+
				"  }",
			secretName, scriptName, secretName,
		)
	} else {
		warningDetail = "In v5, cloudflare_workers_secret is removed. Secrets must be migrated to secret_text bindings within cloudflare_workers_script resources."
	}

	// Add warning diagnostic
	ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Manual migration required: workers_secret removed in v5",
		Detail:   warningDetail,
	})

	// Add warning comment directly to the resource block
	tfhcl.AppendWarningComment(body,
		"cloudflare_workers_secret removed in v5. "+
			"Migrate this secret to a 'secret_text' binding in cloudflare_workers_script. "+
			"See: https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade#cloudflare_workers_secret")

	// Keep the resource block but add the warning
	// The user needs to manually migrate by moving the secret to workers_script bindings
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// extractAttributeValue extracts the value of an attribute as a string
// Returns empty string if attribute doesn't exist or value can't be determined
func extractAttributeValue(body *hclwrite.Body, attrName string) string {
	attr := body.GetAttribute(attrName)
	if attr == nil {
		return ""
	}

	// Try to extract a simple string value
	tokens := attr.Expr().BuildTokens(nil)
	if len(tokens) > 0 {
		// For simple string literals like "value" or "${var.name}"
		// Return the raw bytes as a string
		val := string(tokens.Bytes())
		// Remove quotes if present
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			return val[1 : len(val)-1]
		}
		return val
	}

	return ""
}
