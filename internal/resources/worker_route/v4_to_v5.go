package worker_route

import (
	"bytes"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both v4 resource names (plural and singular forms)
	internal.RegisterMigrator("cloudflare_workers_route", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_worker_route", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return v5 resource name (plural form)
	return "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both v4 resource names (plural and singular)
	return resourceType == "cloudflare_workers_route" || resourceType == "cloudflare_worker_route"
}

// GetResourceRename implements the ResourceRenamer interface
// Handles both cloudflare_worker_route (singular) and cloudflare_workers_route (plural)
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_workers_route", "cloudflare_worker_route"}, "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for this simple migration
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original resource type before any modifications (for moved block generation)
	originalResourceType := tfhcl.GetResourceType(block)

	body := block.Body()
	resourceName := tfhcl.GetResourceName(block)

	// Check if this is the singular form (needs moved block)
	wasSingular := originalResourceType == "cloudflare_worker_route"

	// Handle resource rename: cloudflare_worker_route → cloudflare_workers_route
	tfhcl.RenameResourceType(block, "cloudflare_worker_route", "cloudflare_workers_route")

	// Rename field: script_name → script
	tfhcl.RenameAttribute(body, "script_name", "script")

	// Replace cloudflare_workers_script.*.name -> cloudflare_workers_script.*.id
	if err := replaceScriptReferenceAttribute(body); err != nil {
		return nil, err
	}

	// Generate moved block if the resource was renamed (singular → plural)
	if wasSingular {
		_, newType := m.GetResourceRename()
		from := originalResourceType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock := tfhcl.CreateMovedBlock(from, to)

		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block, movedBlock},
			RemoveOriginal: true,
		}, nil
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// replaceScriptReferenceAttribute updates script traversal references from
// cloudflare_worker(s)_script.*.name to cloudflare_worker(s)_script.*.id.
func replaceScriptReferenceAttribute(blockBody *hclwrite.Body) error {
	scriptAttr := blockBody.GetAttribute("script")
	if scriptAttr == nil {
		return nil
	}

	original := scriptAttr.Expr().BuildTokens(nil)
	updated, changed := rewriteScriptReferenceTokens(original)
	if !changed {
		return nil
	}

	blockBody.SetAttributeRaw("script", updated)
	return nil
}

func rewriteScriptReferenceTokens(tokens hclwrite.Tokens) (hclwrite.Tokens, bool) {
	updated := cloneTokens(tokens)
	changed := false

	for i, tok := range updated {
		if tok.Type != hclsyntax.TokenIdent || !bytes.Equal(tok.Bytes, []byte("name")) {
			continue
		}

		if !isWorkerScriptNameTraversal(updated, i) {
			continue
		}

		updated[i].Bytes = []byte("id")
		changed = true
	}

	return updated, changed
}

func isWorkerScriptNameTraversal(tokens hclwrite.Tokens, nameIdx int) bool {
	if nameIdx < 2 {
		return false
	}

	prev := previousSignificantToken(tokens, nameIdx-1)
	if prev < 0 || tokens[prev].Type != hclsyntax.TokenDot {
		return false
	}

	for i := prev - 1; i >= 0; i-- {
		i = previousSignificantToken(tokens, i)
		if i < 0 {
			return false
		}

		tok := tokens[i]

		// Skip over bracket-enclosed content (e.g., [0], [each.key], ["alpha"])
		if tok.Type == hclsyntax.TokenCBrack {
			i = skipBracketBackward(tokens, i)
			if i < 0 {
				return false
			}
			continue
		}

		if tok.Type == hclsyntax.TokenIdent && (bytes.Equal(tok.Bytes, []byte("cloudflare_worker_script")) || bytes.Equal(tok.Bytes, []byte("cloudflare_workers_script"))) {
			return isValidScriptTraversal(tokens, i, nameIdx)
		}

		if traversalDelimiter(tok.Type) {
			return false
		}
	}

	return false
}

// skipBracketBackward scans backward from a closing bracket to find its matching
// opening bracket, returning the index of the TokenOBrack. Returns -1 if not found.
func skipBracketBackward(tokens hclwrite.Tokens, closeBrackIdx int) int {
	depth := 1
	for i := closeBrackIdx - 1; i >= 0; i-- {
		switch tokens[i].Type {
		case hclsyntax.TokenCBrack:
			depth++
		case hclsyntax.TokenOBrack:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isValidScriptTraversal(tokens hclwrite.Tokens, rootIdx, nameIdx int) bool {
	if rootIdx+2 >= nameIdx {
		return false
	}

	firstDot := nextSignificantToken(tokens, rootIdx+1)
	if firstDot < 0 || tokens[firstDot].Type != hclsyntax.TokenDot {
		return false
	}

	resourceName := nextSignificantToken(tokens, firstDot+1)
	if resourceName < 0 || tokens[resourceName].Type != hclsyntax.TokenIdent {
		return false
	}

	dotCount := 0
	for i := rootIdx + 1; i < nameIdx; i++ {
		i = nextSignificantToken(tokens, i)
		if i < 0 || i >= nameIdx {
			break
		}

		// Skip over bracket-enclosed content (e.g., [0], [each.key], ["alpha"])
		if tokens[i].Type == hclsyntax.TokenOBrack {
			i = skipBracketForward(tokens, i)
			if i < 0 {
				return false
			}
			continue
		}

		if traversalDelimiter(tokens[i].Type) {
			return false
		}

		if tokens[i].Type == hclsyntax.TokenDot {
			dotCount++
		}
	}

	return dotCount >= 2
}

// skipBracketForward scans forward from an opening bracket to find its matching
// closing bracket, returning the index of the TokenCBrack. Returns -1 if not found.
func skipBracketForward(tokens hclwrite.Tokens, openBrackIdx int) int {
	depth := 1
	for i := openBrackIdx + 1; i < len(tokens); i++ {
		switch tokens[i].Type {
		case hclsyntax.TokenOBrack:
			depth++
		case hclsyntax.TokenCBrack:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func previousSignificantToken(tokens hclwrite.Tokens, start int) int {
	for i := start; i >= 0; i-- {
		t := tokens[i].Type
		if t == hclsyntax.TokenNewline || t == hclsyntax.TokenComment {
			continue
		}
		return i
	}
	return -1
}

func nextSignificantToken(tokens hclwrite.Tokens, start int) int {
	for i := start; i < len(tokens); i++ {
		t := tokens[i].Type
		if t == hclsyntax.TokenNewline || t == hclsyntax.TokenComment {
			continue
		}
		return i
	}
	return -1
}

func traversalDelimiter(t hclsyntax.TokenType) bool {
	switch t {
	case hclsyntax.TokenEOF, hclsyntax.TokenComma, hclsyntax.TokenQuestion, hclsyntax.TokenColon, hclsyntax.TokenEqual,
		hclsyntax.TokenOr, hclsyntax.TokenAnd, hclsyntax.TokenPlus, hclsyntax.TokenMinus, hclsyntax.TokenStar,
		hclsyntax.TokenSlash, hclsyntax.TokenPercent, hclsyntax.TokenGreaterThan, hclsyntax.TokenGreaterThanEq,
		hclsyntax.TokenLessThan, hclsyntax.TokenLessThanEq, hclsyntax.TokenEqualOp, hclsyntax.TokenNotEqual,
		hclsyntax.TokenOQuote, hclsyntax.TokenQuotedLit, hclsyntax.TokenCQuote:
		return true
	default:
		return false
	}
}

func cloneTokens(tokens hclwrite.Tokens) hclwrite.Tokens {
	out := make(hclwrite.Tokens, len(tokens))
	for i, tok := range tokens {
		out[i] = &hclwrite.Token{Type: tok.Type, Bytes: append([]byte(nil), tok.Bytes...)}
	}
	return out
}
