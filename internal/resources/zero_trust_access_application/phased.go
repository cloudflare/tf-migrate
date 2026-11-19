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

// attachImportedPolicies lists the app-scoped policies restored by step 3 of
// the phased migration (see internal/accesspolicyphase) on the application.
// v4 attached them from the policy side.
//
// v5 expects policies in precedence order, or every plan shows a diff. With
// only single policies the precedences are in config and the list is written
// sorted. A for_each policy's precedences are only in the record, so the
// entries go into a locals map (returned, to be written next to the
// application) and policies sorts them by precedence.
func (m *V4ToV5Migrator) attachImportedPolicies(ctx *transform.Context, body *hclwrite.Body, appName string) *hclwrite.Block {
	dir := filepath.Dir(ctx.FilePath)
	policies := accesspolicyphase.ForApp(dir, appName)
	if len(policies) == 0 || accesspolicyphase.StateOf(dir) != accesspolicyphase.Frozen {
		return nil
	}
	local := "tf_migrate_" + appName + "_policies"
	value, locals := staticPolicies(policies), ""
	if hasForEach(policies) {
		locals = policiesMap(policies)
		value = fmt.Sprintf("[for s in sort([for k, v in local.%s : format(\"%%09d|%%s\", v.precedence, k)]) : local.%s[substr(s, 10, -1)]]", local, local)
	}
	if body.GetAttribute("policies") != nil {
		detail := "This application already sets policies, so tf-migrate did not add the policies it imports. Add them to policies, in precedence order:\n\n" + value
		if locals != "" {
			detail += "\n\nwith\n\nlocals {\n  " + local + " = " + locals + "\n}"
		}
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Action required: add imported policies to cloudflare_zero_trust_access_application.%s", appName),
			Detail:   detail,
		})
		return nil
	}
	body.SetAttributeRaw("policies", tokensOf("policies", value))
	if locals == "" {
		return nil
	}
	b := hclwrite.NewBlock("locals", nil)
	b.Body().SetAttributeRaw(local, tokensOf(local, locals))
	return b
}

func hasForEach(policies []accesspolicyphase.Policy) bool {
	for _, p := range policies {
		if p.ForEach {
			return true
		}
	}
	return false
}

// staticPolicies renders single policies, already in precedence order.
func staticPolicies(policies []accesspolicyphase.Policy) string {
	var b strings.Builder
	b.WriteString("[\n")
	for _, p := range policies {
		fmt.Fprintf(&b, "    {\n      id         = cloudflare_zero_trust_access_policy.%s.id\n      precedence = %s\n    },\n", p.Name, p.Precedence)
	}
	b.WriteString("  ]")
	return b.String()
}

// policiesMap renders every policy instance as key => { id, precedence },
// with the recorded precedence.
func policiesMap(policies []accesspolicyphase.Policy) string {
	var single []string
	var parts []string
	for _, p := range policies {
		if p.ForEach {
			parts = append(parts, fmt.Sprintf(`{ for k, v in %s : "%s/${k}" => { id = cloudflare_zero_trust_access_policy.%s[k].id, precedence = v.precedence } }`, p.RecordRef(), p.Name, p.Name))
			continue
		}
		single = append(single, fmt.Sprintf("      %s = { id = cloudflare_zero_trust_access_policy.%s.id, precedence = %s.precedence }", p.Name, p.Name, p.RecordRef()))
	}
	if len(single) > 0 {
		parts = append([]string{"{\n" + strings.Join(single, "\n") + "\n    }"}, parts...)
	}
	return "merge(\n    " + strings.Join(parts, ",\n    ") + ",\n  )"
}

func tokensOf(name, expr string) hclwrite.Tokens {
	f, diags := hclwrite.ParseConfig([]byte(name+" = "+expr+"\n"), "", hcl.InitialPos)
	if diags.HasErrors() {
		return hclwrite.Tokens{{Bytes: []byte(expr)}}
	}
	return f.Body().GetAttribute(name).Expr().BuildTokens(nil)
}
