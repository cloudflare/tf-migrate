package zero_trust_access_application

import (
	"fmt"
	"path/filepath"
	"strconv"
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
// v5 expects policies in precedence order, or every plan shows a diff. When
// every precedence is an integer literal, the list is written sorted. Otherwise
// (a for_each policy, or a precedence from a variable or local) the values are
// only known to Terraform, so the entries go into a locals map (returned, to be
// written next to the application) and policies sorts them by the recorded
// precedence.
//
// accesspolicyphase.Scan leaves applications that already set policies on the
// manual path, so there is nothing to merge with here.
func (m *V4ToV5Migrator) attachImportedPolicies(ctx *transform.Context, body *hclwrite.Body, appName string) *hclwrite.Block {
	dir := filepath.Dir(ctx.FilePath)
	policies := accesspolicyphase.ForApp(dir, appName)
	if len(policies) == 0 || accesspolicyphase.StateOf(dir) != accesspolicyphase.Frozen {
		return nil
	}
	if body.GetAttribute("policies") != nil {
		return nil
	}
	local := "tf_migrate_" + appName + "_policies"
	value, locals := staticPolicies(policies), ""
	if !allLiteralPrecedence(policies) {
		locals = policiesMap(policies)
		value = fmt.Sprintf("[for s in sort([for k, v in local.%s : format(\"%%09d|%%s\", v.precedence, k)]) : local.%s[substr(s, 10, -1)]]", local, local)
	}
	body.SetAttributeRaw("policies", tokensOf("policies", value))
	if locals == "" {
		return nil
	}
	b := hclwrite.NewBlock("locals", nil)
	b.Body().SetAttributeRaw(local, tokensOf(local, locals))
	return b
}

// allLiteralPrecedence reports whether every policy is single and has an
// integer literal precedence, so the list order is known now.
func allLiteralPrecedence(policies []accesspolicyphase.Policy) bool {
	for _, p := range policies {
		if _, err := strconv.Atoi(p.Precedence); p.ForEach || err != nil {
			return false
		}
	}
	return true
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
