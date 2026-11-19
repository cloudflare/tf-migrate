package zero_trust_access_policy

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// NeedsPhaseOne implements transform.PhaseOneFilter. Only the app-scoped
// policies step 1 recorded (accesspolicyphase.Scan) take part in step 2.
func (m *V4ToV5Migrator) NeedsPhaseOne(filePath string, block *hclwrite.Block) bool {
	if block.Type() != "resource" || len(block.Labels()) != 2 {
		return false
	}
	dir := filepath.Dir(filePath)
	if accesspolicyphase.StateOf(dir) != accesspolicyphase.Recorded {
		return false
	}
	_, ok := accesspolicyphase.Find(dir, block.Labels()[0], block.Labels()[1])
	return ok
}

// TransformPhaseOne implements transform.PhaseOneTransformer: v4 drops the
// policy from state without deleting it.
func (m *V4ToV5Migrator) TransformPhaseOne(_ *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	addr := block.Labels()[0] + "." + block.Labels()[1]
	return &transform.TransformResult{Blocks: []*hclwrite.Block{tfhcl.CreateRemovedBlock(addr)}}, nil
}

// newImportBlock returns the import block for p at address to. A for_each
// policy gets one import block iterating over its record entries.
func newImportBlock(p accesspolicyphase.Policy, to string) *hclwrite.Block {
	b := hclwrite.NewBlock("import", nil)
	if p.ForEach {
		b.Body().SetAttributeRaw("for_each", tokensOf(p.RecordRef()))
		to += "[each.key]"
	}
	b.Body().SetAttributeRaw("to", tokensOf(to))
	b.Body().SetAttributeRaw("id", tokensOf(accesspolicyphase.ImportID(p)))
	return b
}

func tokensOf(expr string) hclwrite.Tokens {
	f, diags := hclwrite.ParseConfig([]byte("x = "+expr+"\n"), "", hcl.InitialPos)
	if diags.HasErrors() {
		return hclwrite.Tokens{{Bytes: []byte(expr)}}
	}
	return f.Body().GetAttribute("x").Expr().BuildTokens(nil)
}
