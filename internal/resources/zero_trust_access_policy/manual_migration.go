package zero_trust_access_policy

import (
	"path/filepath"
	"strings"

	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const ManualMigrationSummary = "Application-scoped access policy must be inlined"
const ManualMigrationGuide = "https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-migration#application-scoped-access-policies"

func policySourcePath(ctx *transform.Context) string {
	if ctx.FilePath != "" {
		return filepath.Base(ctx.FilePath)
	}
	return ctx.Filename
}

// IsApplicationScopedPolicy recognizes both resource names supported by v4.
// These policies must be migrated inline, not moved to an account-level resource.
func IsApplicationScopedPolicy(block *hclwrite.Block) bool {
	if block.Type() != "resource" || len(block.Labels()) != 2 {
		return false
	}
	resourceType := block.Labels()[0]
	return (resourceType == "cloudflare_access_policy" || resourceType == "cloudflare_zero_trust_access_policy") &&
		block.Body().GetAttribute("application_id") != nil
}

func commentOriginalPolicy(block *hclwrite.Block) hclwrite.Tokens {
	var comment strings.Builder
	comment.WriteString("\n# Original v4 policy — convert to inline policies before applying:\n")
	for _, line := range strings.Split(strings.TrimSpace(string(block.BuildTokens(nil).Bytes())), "\n") {
		comment.WriteByte('#')
		if line != "" {
			comment.WriteByte(' ')
			comment.WriteString(line)
		}
		comment.WriteByte('\n')
	}
	return hclwrite.Tokens{{Type: hclsyntax.TokenComment, Bytes: []byte(comment.String())}}
}
