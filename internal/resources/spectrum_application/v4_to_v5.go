package spectrum_application

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the v4 resource name
	internal.RegisterMigrator("cloudflare_spectrum_application", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Resource name doesn't change
	return "cloudflare_spectrum_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_spectrum_application"
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_spectrum_application"}, "cloudflare_spectrum_application"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For spectrum_application, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig transforms the HCL configuration from v4 to v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Remove id attribute if present (defensive cleanup - id is computed-only in both versions)
	tfhcl.RemoveAttributes(body, "id")

	// 2. Convert MaxItems:1 blocks to attributes
	// dns: required block -> attribute
	tfhcl.ConvertBlocksToAttribute(body, "dns", "dns", func(block *hclwrite.Block) {})

	// origin_dns: optional block -> attribute
	tfhcl.ConvertBlocksToAttribute(body, "origin_dns", "origin_dns", func(block *hclwrite.Block) {})

	// edge_ips: optional block -> attribute
	tfhcl.ConvertBlocksToAttribute(body, "edge_ips", "edge_ips", func(block *hclwrite.Block) {})

	// 3. Convert origin_port_range block to origin_port string
	m.convertOriginPortRange(body)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// convertOriginPortRange converts origin_port_range block to origin_port string attribute
func (m *V4ToV5Migrator) convertOriginPortRange(body *hclwrite.Body) {
	// Find origin_port_range blocks
	for _, block := range body.Blocks() {
		if block.Type() == "origin_port_range" {
			blockBody := block.Body()

			// Extract start and end attributes
			startAttr := blockBody.GetAttribute("start")
			endAttr := blockBody.GetAttribute("end")

			if startAttr != nil && endAttr != nil {
				startTokens := startAttr.Expr().BuildTokens(nil)
				endTokens := endAttr.Expr().BuildTokens(nil)

				if len(startTokens) >= 1 && len(endTokens) >= 1 {
					// Build the "start-end" range. Either side may be an
					// expression rather than a literal port number, so each is
					// interpolated unless it is a bare number. Reading just the
					// first token here used to turn "var.port_start" into "var",
					// producing a well-formed but meaningless "var-var" that
					// terraform validate accepts.
					body.SetAttributeRaw("origin_port", portRangeTokens(startTokens, endTokens))
				}
			}

			// Remove the origin_port_range block
			body.RemoveBlock(block)
			break // Only handle the first one (MaxItems: 1)
		}
	}
}

// portRangeTokens renders the v5 origin_port value for a v4 origin_port_range
// block as the string "start-end".
//
// v5 expresses the range as a single string, so a side that is not a literal
// port number has to be interpolated to survive. The literal text around the
// interpolations is emitted as one token per run, matching how HCL parses the
// equivalent hand-written string.
func portRangeTokens(startTokens, endTokens hclwrite.Tokens) hclwrite.Tokens {
	startLit, startIsLit := tfhcl.LiteralNumber(startTokens)
	endLit, endIsLit := tfhcl.LiteralNumber(endTokens)

	// Both sides literal: a plain quoted string, identical to what the
	// pre-existing literal path produced.
	if startIsLit && endIsLit {
		return hclwrite.TokensForValue(cty.StringVal(startLit + "-" + endLit))
	}

	tokens := hclwrite.Tokens{{Type: hclsyntax.TokenOQuote, Bytes: []byte{'"'}}}
	if startIsLit {
		// The separator joins the leading literal run: "1000-${...}".
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(startLit + "-")})
	} else {
		tokens = append(tokens, interpolation(startTokens)...)
	}

	if endIsLit {
		// Mirrored: the separator joins the trailing literal run, "${...}-2000".
		if !startIsLit {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte("-" + endLit)})
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(endLit)})
		}
	} else {
		if startIsLit {
			tokens = append(tokens, interpolation(endTokens)...)
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte("-")})
			tokens = append(tokens, interpolation(endTokens)...)
		}
	}

	return append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte{'"'}})
}

// interpolation wraps an expression as "${expr}" for embedding in a template.
func interpolation(tokens hclwrite.Tokens) hclwrite.Tokens {
	out := hclwrite.Tokens{{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")}}
	out = append(out, tfhcl.TrimLeadingSpace(tokens)...)
	return append(out, &hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte{'}'}})
}
