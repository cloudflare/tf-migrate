package zero_trust_access_policy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
)

func TestApplicationScopedPolicyAliases(t *testing.T) {
	for _, resourceType := range []string{"cloudflare_access_policy", "cloudflare_zero_trust_access_policy"} {
		t.Run(resourceType, func(t *testing.T) {
			input := fmt.Sprintf(`resource %q "app_policy" {
  for_each       = var.policies
  account_id     = var.account_id
  application_id = cloudflare_access_application.app[each.key].id
  name           = each.value.name
  decision       = "allow"
  precedence     = index(local.order, each.key) + 3
  session_duration = "18h"
  /* Keep all conditions, not just include. */
  include { email = ["employee@example.com"] }
  exclude { email = ["contractor@example.com"] }
  require { auth_method = "mfa" }
  purpose_justification_prompt = <<-EOT
    Explain why access is needed. */
  EOT
}`, resourceType)
			file, diags := hclwrite.ParseConfig([]byte(input), "access.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors(), diags.Error())
			original := file.Body().Blocks()[0]
			originalText := strings.TrimSpace(string(original.BuildTokens(nil).Bytes()))
			ctx := &transform.Context{FilePath: "modules/access/access.tf"}
			result, err := NewV4ToV5Migrator().TransformConfig(ctx, original)
			require.NoError(t, err)
			require.True(t, result.RemoveOriginal)
			require.Len(t, result.Blocks, 1)
			require.Equal(t, "removed", result.Blocks[0].Type())
			require.Equal(t, resourceType+".app_policy", strings.TrimSpace(string(result.Blocks[0].Body().GetAttribute("from").Expr().BuildTokens(nil).Bytes())))

			output := string(hclwrite.Format(result.Blocks[0].BuildTokens(nil).Bytes()))
			for _, line := range strings.Split(originalText, "\n") {
				require.Contains(t, output, "# "+line+"\n", "original source must be preserved as line comments")
			}
			require.Contains(t, output, "MIGRATION WARNING")
			require.Contains(t, output, "BEFORE running terraform apply")
			parsed, outputDiags := hclwrite.ParseConfig([]byte(output), "access.tf", hcl.InitialPos)
			require.False(t, outputDiags.HasErrors(), outputDiags.Error())
			require.Len(t, parsed.Body().Blocks(), 1, "commented policy must not be active HCL")
			require.Len(t, ctx.Diagnostics, 1)
			require.Equal(t, hcl.DiagWarning, ctx.Diagnostics[0].Severity)
			require.Contains(t, ctx.Diagnostics[0].Detail, resourceType+".app_policy  (modules/access/access.tf)")
			require.Contains(t, ctx.Diagnostics[0].Detail, "empty policies list")
			require.NotContains(t, ctx.Diagnostics[0].Detail, "precedence = 1")
		})
	}
}
