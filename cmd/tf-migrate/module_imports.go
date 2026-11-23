package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// childModuleImportSummary marks the per-resource warning that import blocks
// were generated into a child module. Migrators set Subject.Filename to the
// file they migrated, so the warning can be dropped once the blocks are moved.
const childModuleImportSummary = "to the root module"

// moduleCall is a module block whose source is a local path.
type moduleCall struct {
	name     string
	file     *migratedFile
	block    *hclwrite.Block
	childDir string
}

type migratedFile struct {
	path  string
	file  *hclwrite.File
	dirty bool
}

// hoistModuleImports moves import blocks out of local child modules into the
// module that calls them, where Terraform accepts them. For each call
//
//	module "m" { source = "./mod"; zone_id = "abc" }
//
// an import { to = cloudflare_zone_setting.x; id = "${var.zone_id}/ssl" } in
// ./mod becomes, next to the call,
//
//	import { to = module.m.cloudflare_zone_setting.x; id = "${("abc")}/ssl" }
//
// var.<name> is replaced with the call's argument, or the variable's default.
// A child is moved all or nothing: when one of its import blocks can't be
// written in a caller (an id that uses a local or a data source, a caller
// that is itself a module, a call with count, ...), its blocks stay where they
// are and a warning says why.
func hoistModuleImports(log hclog.Logger, cfg config, outputPaths []string, diags hcl.Diagnostics) hcl.Diagnostics {
	files := map[string][]*migratedFile{}
	for _, p := range outputPaths {
		content, err := os.ReadFile(p)
		if err != nil {
			log.Warn("Failed to read file for module import blocks", "file", p, "error", err)
			continue
		}
		f, d := hclwrite.ParseConfig(content, p, hcl.InitialPos)
		if d.HasErrors() {
			continue
		}
		dir := absDir(p)
		files[dir] = append(files[dir], &migratedFile{path: p, file: f})
	}

	callsInto := map[string][]moduleCall{}
	for _, dir := range sortedKeys(files) {
		for _, mf := range files[dir] {
			for _, b := range mf.file.Body().Blocks() {
				if b.Type() != "module" || len(b.Labels()) != 1 {
					continue
				}
				src, ok := tfhcl.JoinQuotedLiteral(attrTokens(b.Body(), "source"))
				if !ok || (!strings.HasPrefix(src, "./") && !strings.HasPrefix(src, "../")) {
					continue
				}
				c := moduleCall{name: b.Labels()[0], file: mf, block: b, childDir: filepath.Clean(filepath.Join(dir, src))}
				callsInto[c.childDir] = append(callsInto[c.childDir], c)
			}
		}
	}

	// Children whose per-resource "move them to the root module" warnings
	// are replaced by one warning or note here.
	resolved := map[string]bool{}
	var out hcl.Diagnostics
	for _, childDir := range sortedKeys(files) {
		var imports []*hclwrite.Block
		var owners []*migratedFile
		for _, mf := range files[childDir] {
			for _, b := range mf.file.Body().Blocks() {
				if b.Type() == "import" {
					imports = append(imports, b)
					owners = append(owners, mf)
				}
			}
		}
		callers := callsInto[childDir]
		if len(imports) == 0 || len(callers) == 0 {
			continue
		}

		reason := ""
		type planned struct {
			call  moduleCall
			block *hclwrite.Block
		}
		var plan []planned
		for _, c := range callers {
			if len(callsInto[absDir(c.file.path)]) > 0 {
				reason = fmt.Sprintf("module.%s is called from %s, which is itself a module; move them by hand to the root module", c.name, filepath.Dir(c.file.path))
				break
			}
			if c.block.Body().GetAttribute("count") != nil {
				reason = fmt.Sprintf("module.%s uses count, which import blocks don't support", c.name)
				break
			}
			for _, imp := range imports {
				nb, why := importForCall(imp, c, files[childDir])
				if why != "" {
					reason = why
					break
				}
				if !hasImportTo(c.file.file, nb) {
					plan = append(plan, planned{c, nb})
				}
			}
			if reason != "" {
				break
			}
		}

		resolved[childDir] = true
		resolved[inputDir(cfg, childDir)] = true
		var addrs []string
		for _, imp := range imports {
			addrs = append(addrs, "  "+strings.TrimSpace(string(attrTokens(imp.Body(), "to").Bytes())))
		}
		if reason != "" {
			out = append(out, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary: fmt.Sprintf("Action required: move %d import block(s) in %s to the root module",
					len(imports), displayPath(cfg, childDir)),
				Detail: fmt.Sprintf("Terraform only accepts import blocks in the root module, and rejects "+
					"these at plan time. tf-migrate could not move them: %s.\n\n%s\n\n"+
					"Move them to the root module and prefix 'to' with the module address, for example "+
					"to = module.<name>.cloudflare_zone_setting.<name>. If the module is called with "+
					"for_each, use module.<name>[each.key] and a for_each import block (Terraform 1.7+).",
					reason, strings.Join(addrs, "\n")),
			})
			continue
		}

		for i, imp := range imports {
			owners[i].file.Body().RemoveBlock(imp)
			owners[i].dirty = true
		}
		movedTo := map[string]int{}
		for _, p := range plan {
			body := p.call.file.file.Body()
			body.AppendNewline()
			body.AppendBlock(p.block)
			p.call.file.dirty = true
			movedTo[p.call.file.path]++
		}
		var where []string
		for _, f := range sortedKeys(movedTo) {
			where = append(where, fmt.Sprintf("  %s: %d", displayPath(cfg, f), movedTo[f]))
		}
		out = append(out, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary: fmt.Sprintf("Moved %d import block(s) from module %s to the files that call it",
				len(imports), displayPath(cfg, childDir)),
			Detail: "Terraform only accepts import blocks in the root module. tf-migrate removed them from " +
				"the module and wrote one per module call, with 'to' prefixed by the module address and " +
				"var.* in 'id' replaced by the module argument:\n\n" + strings.Join(where, "\n") +
				"\n\nCheck the import ids in the plan.",
		})
	}

	for _, mfs := range files {
		for _, mf := range mfs {
			if !mf.dirty {
				continue
			}
			if err := os.WriteFile(mf.path, hclwrite.Format(mf.file.Bytes()), 0o644); err != nil {
				out = append(out, &hcl.Diagnostic{Severity: hcl.DiagError, Summary: "Failed to write " + mf.path, Detail: err.Error()})
			}
		}
	}

	kept := diags[:0:0]
	for _, d := range diags {
		if d.Subject != nil && strings.Contains(d.Summary, childModuleImportSummary) &&
			resolved[absDir(d.Subject.Filename)] {
			continue
		}
		kept = append(kept, d)
	}
	return append(kept, out...)
}

// importForCall builds the import block for one module call, or returns why
// it can't.
func importForCall(imp *hclwrite.Block, c moduleCall, childFiles []*migratedFile) (*hclwrite.Block, string) {
	body := imp.Body()
	to := body.GetAttribute("to")
	id := body.GetAttribute("id")
	if to == nil || id == nil {
		return nil, "an import block has no 'to' or 'id'"
	}
	if body.GetAttribute("provider") != nil {
		return nil, "an import block sets provider"
	}
	callForEach := c.block.Body().GetAttribute("for_each")
	impForEach := body.GetAttribute("for_each")
	if callForEach != nil && impForEach != nil {
		return nil, fmt.Sprintf("both module.%s and an import block in it use for_each", c.name)
	}

	nb := hclwrite.NewBlock("import", nil)
	nbBody := nb.Body()
	switch {
	case callForEach != nil:
		nbBody.SetAttributeRaw("for_each", callForEach.Expr().BuildTokens(nil))
	case impForEach != nil:
		tokens, why := substituteVars(impForEach.Expr().BuildTokens(nil), c, childFiles)
		if why != "" {
			return nil, why
		}
		nbBody.SetAttributeRaw("for_each", tokens)
	}

	index := ""
	if callForEach != nil {
		index = "[each.key]"
	}
	toTokens := hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte("module." + c.name + index + ".")}}
	nbBody.SetAttributeRaw("to", append(toTokens, to.Expr().BuildTokens(nil)...))

	idTokens, why := substituteVars(id.Expr().BuildTokens(nil), c, childFiles)
	if why != "" {
		return nil, why
	}
	nbBody.SetAttributeRaw("id", idTokens)
	return nb, ""
}

// substituteVars replaces var.<name> with the module call's argument (or the
// variable's default) in parentheses. Other references aren't visible from
// the caller, so they make the block unmovable. each.* is kept: it binds to
// the import block's for_each, which is the child's or the call's.
func substituteVars(tokens hclwrite.Tokens, c moduleCall, childFiles []*migratedFile) (hclwrite.Tokens, string) {
	expr, d := hclsyntax.ParseExpression(tokens.Bytes(), "expr", hcl.InitialPos)
	if d.HasErrors() {
		return nil, "an import id could not be parsed"
	}
	for _, tr := range expr.Variables() {
		switch root := tr.RootName(); root {
		case "var", "each":
		default:
			return nil, fmt.Sprintf("an import id or for_each uses %s.*, which isn't visible from the module call", root)
		}
	}

	var out hclwrite.Tokens
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		isVar := t.Type == hclsyntax.TokenIdent && string(t.Bytes) == "var" &&
			(i == 0 || tokens[i-1].Type != hclsyntax.TokenDot) &&
			i+2 < len(tokens) && tokens[i+1].Type == hclsyntax.TokenDot && tokens[i+2].Type == hclsyntax.TokenIdent
		if !isVar {
			out = append(out, t)
			continue
		}
		name := string(tokens[i+2].Bytes)
		value := attrTokens(c.block.Body(), name)
		if value == nil {
			value = variableDefault(childFiles, name)
		}
		if value == nil {
			return nil, fmt.Sprintf("module.%s doesn't set %s and the variable has no default", c.name, name)
		}
		if needsParens(value) {
			out = append(out, &hclwrite.Token{Type: hclsyntax.TokenOParen, Bytes: []byte("("), SpacesBefore: t.SpacesBefore})
			out = append(out, value...)
			out = append(out, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
		} else {
			out = append(out, value...)
		}
		i += 2
	}
	return collapseLiteralInterpolations(out), ""
}

// collapseLiteralInterpolations turns "${"abc"}/ssl" into "abc/ssl".
func collapseLiteralInterpolations(tokens hclwrite.Tokens) hclwrite.Tokens {
	var out hclwrite.Tokens
	for i := 0; i < len(tokens); i++ {
		if i+4 < len(tokens) &&
			tokens[i].Type == hclsyntax.TokenTemplateInterp &&
			tokens[i+1].Type == hclsyntax.TokenOQuote &&
			tokens[i+2].Type == hclsyntax.TokenQuotedLit &&
			tokens[i+3].Type == hclsyntax.TokenCQuote &&
			tokens[i+4].Type == hclsyntax.TokenTemplateSeqEnd {
			out = append(out, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: tokens[i+2].Bytes})
			i += 4
			continue
		}
		out = append(out, tokens[i])
	}
	return out
}

// needsParens reports whether value has to be parenthesised to replace a
// reference, so that var.x.y or an operator next to it keeps its meaning.
func needsParens(value hclwrite.Tokens) bool {
	expr, d := hclsyntax.ParseExpression(value.Bytes(), "value", hcl.InitialPos)
	if d.HasErrors() {
		return true
	}
	switch expr.(type) {
	case *hclsyntax.ScopeTraversalExpr, *hclsyntax.LiteralValueExpr, *hclsyntax.TemplateExpr,
		*hclsyntax.FunctionCallExpr, *hclsyntax.TupleConsExpr, *hclsyntax.ObjectConsExpr:
		return false
	}
	return true
}

func variableDefault(files []*migratedFile, name string) hclwrite.Tokens {
	for _, mf := range files {
		for _, b := range mf.file.Body().Blocks() {
			if b.Type() == "variable" && len(b.Labels()) == 1 && b.Labels()[0] == name {
				return attrTokens(b.Body(), "default")
			}
		}
	}
	return nil
}

func hasImportTo(f *hclwrite.File, nb *hclwrite.Block) bool {
	want := strings.Join(strings.Fields(string(attrTokens(nb.Body(), "to").Bytes())), "")
	for _, b := range f.Body().Blocks() {
		if b.Type() == "import" &&
			strings.Join(strings.Fields(string(attrTokens(b.Body(), "to").Bytes())), "") == want {
			return true
		}
	}
	return false
}

func attrTokens(body *hclwrite.Body, name string) hclwrite.Tokens {
	if a := body.GetAttribute(name); a != nil {
		return a.Expr().BuildTokens(nil)
	}
	return nil
}

func absDir(path string) string {
	d := filepath.Dir(path)
	if a, err := filepath.Abs(d); err == nil {
		return a
	}
	return filepath.Clean(d)
}

// inputDir maps an output directory back to the input directory it was
// migrated from.
func inputDir(cfg config, outDir string) string {
	return absDir(filepath.Join(inputDirForOutput(cfg, filepath.Join(outDir, "x.tf")), "x.tf"))
}

// displayPath shows path relative to the output directory when it's inside.
func displayPath(cfg config, path string) string {
	base, err := filepath.Abs(cfg.outputDir)
	if err == nil {
		if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
