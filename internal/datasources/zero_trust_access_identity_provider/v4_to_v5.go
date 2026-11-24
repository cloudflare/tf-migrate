package zero_trust_access_identity_provider

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

const (
	v4Type     = "cloudflare_access_identity_provider"
	v5Type     = "cloudflare_zero_trust_access_identity_provider"
	pluralType = "cloudflare_zero_trust_access_identity_providers"
)

// V4ToV5Migrator handles the cloudflare_zero_trust_access_identity_provider
// data source (v4 also: cloudflare_access_identity_provider).
//
// v4 looked the identity provider up by name: it listed them and took the
// first whose name matched. v5 looks it up by identity_provider_id, and name
// is read-only. The migration keeps the data source, so references to it
// keep working, and adds a plural data source to find the ID:
//
//	data "cloudflare_zero_trust_access_identity_providers" "idp_lookup" {
//	  account_id = var.account_id
//	}
//	data "cloudflare_zero_trust_access_identity_provider" "idp" {
//	  account_id           = var.account_id
//	  identity_provider_id = [for p in data.cloudflare_zero_trust_access_identity_providers.idp_lookup.result : p.id if p.name == "MyIdentity"][0]
//	}
//
// [0] takes the first match, like v4, and fails when nothing matches, as v4
// did ("no Access Identity Provider matching name").
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("data."+v4Type, "v4", "v5", migrator)
	internal.RegisterMigrator("data."+v5Type, "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return v5Type
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "data."+v4Type || resourceType == "data."+v5Type
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename renames references to the deprecated v4 name.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"data." + v4Type, "data." + v5Type}, "data." + v5Type
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	tfhcl.RenameResourceType(block, v4Type, v5Type)
	body := block.Body()

	nameAttr := body.GetAttribute("name")
	if nameAttr == nil || body.GetAttribute("identity_provider_id") != nil {
		return &transform.TransformResult{Blocks: []*hclwrite.Block{block}}, nil
	}

	labels := block.Labels()
	lookupName := uniqueLookupName(ctx, labels[1]+"_lookup")
	lookup := hclwrite.NewBlock("data", []string{pluralType, lookupName})
	lookupBody := lookup.Body()

	// The list is the same for every instance unless the account or zone
	// depends on the instance, so the lookup only repeats in that case.
	index := ""
	scope := []string{"account_id", "zone_id"}
	if usesInstance(body, scope, "each") {
		if fe := body.GetAttribute("for_each"); fe != nil {
			lookupBody.SetAttributeRaw("for_each", fe.Expr().BuildTokens(nil))
			index = "[each.key]"
		}
	} else if usesInstance(body, scope, "count") {
		if c := body.GetAttribute("count"); c != nil {
			lookupBody.SetAttributeRaw("count", c.Expr().BuildTokens(nil))
			index = "[count.index]"
		}
	}
	for _, name := range scope {
		if attr := body.GetAttribute(name); attr != nil {
			lookupBody.SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
		}
	}

	nameTokens := nameAttr.Expr().BuildTokens(nil)
	idTokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(fmt.Sprintf(
			"for p in data.%s.%s%s.result : p.id if p.name == ", pluralType, lookupName, index))},
	}
	idTokens = append(idTokens, nameTokens...)
	idTokens = append(idTokens,
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
		&hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		&hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte("0")},
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
	)
	body.RemoveAttribute("name")
	body.SetAttributeRaw("identity_provider_id", idTokens)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{lookup, block},
		RemoveOriginal: true,
	}, nil
}

// usesInstance reports whether any of the attributes refers to each.* or
// count.*.
func usesInstance(body *hclwrite.Body, attrs []string, root string) bool {
	for _, name := range attrs {
		attr := body.GetAttribute(name)
		if attr == nil {
			continue
		}
		tokens := attr.Expr().BuildTokens(nil)
		for i := 0; i+1 < len(tokens); i++ {
			if tokens[i].Type == hclsyntax.TokenIdent && string(tokens[i].Bytes) == root &&
				tokens[i+1].Type == hclsyntax.TokenDot {
				return true
			}
		}
	}
	return false
}

// uniqueLookupName returns name, or name with a numeric suffix when the file
// already has a plural identity provider data source with that name.
func uniqueLookupName(ctx *transform.Context, name string) string {
	taken := map[string]bool{}
	if ctx != nil && ctx.CFGFile != nil {
		for _, b := range ctx.CFGFile.Body().Blocks() {
			if l := b.Labels(); b.Type() == "data" && len(l) == 2 && l[0] == pluralType {
				taken[l[1]] = true
			}
		}
	}
	candidate := name
	for i := 2; taken[candidate]; i++ {
		candidate = fmt.Sprintf("%s_%d", name, i)
	}
	return candidate
}
