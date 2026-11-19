package zero_trust_access_application

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// attachImportedPolicies lists the app-scoped policies restored by step 4 of
// the phased migration (see internal/accesspolicyphase) on the application,
// in v4 precedence order. v4 attached them from the policy side.
func (m *V4ToV5Migrator) attachImportedPolicies(ctx *transform.Context, body *hclwrite.Body, appName string) {
	dir := filepath.Dir(ctx.FilePath)
	policies := accesspolicyphase.ForApp(dir, appName)
	if len(policies) == 0 || accesspolicyphase.StateOf(dir) != accesspolicyphase.Frozen {
		return
	}
	if body.GetAttribute("policies") != nil {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Action required: add imported policies to cloudflare_zero_trust_access_application.%s", appName),
			Detail:   "This application already sets policies, so tf-migrate did not add the policies it imports. Add them to policies in this order:\n\n" + policiesSource(policies),
		})
		return
	}
	f, diags := hclwrite.ParseConfig([]byte("policies = "+policiesSource(policies)+"\n"), "", hcl.InitialPos)
	if diags.HasErrors() {
		return
	}
	body.SetAttributeRaw("policies", f.Body().GetAttribute("policies").Expr().BuildTokens(nil))
}

func policiesSource(policies []accesspolicyphase.Policy) string {
	var b strings.Builder
	b.WriteString("[\n")
	for _, p := range policies {
		fmt.Fprintf(&b, "    {\n      id         = cloudflare_zero_trust_access_policy.%s.id\n      precedence = %s\n    },\n", p.Name, p.Precedence)
	}
	b.WriteString("  ]")
	return b.String()
}
