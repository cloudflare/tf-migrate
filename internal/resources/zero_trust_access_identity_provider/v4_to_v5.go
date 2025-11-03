package zero_trust_access_identity_provider

import (
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both old and new names (dual registration)
	internal.RegisterMigrator("cloudflare_access_identity_provider", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_identity_provider", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_identity_provider"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_access_identity_provider" ||
		resourceType == "cloudflare_zero_trust_access_identity_provider"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Rename resource type from v4 to v5
	content = strings.ReplaceAll(content,
		`resource "cloudflare_access_identity_provider"`,
		`resource "cloudflare_zero_trust_access_identity_provider"`)

	// Convert idp_public_cert = "CERT" to idp_public_certs = ["CERT"]
	// This regex looks for idp_public_cert with a quoted string value
	certPattern := regexp.MustCompile(`idp_public_cert\s*=\s*("([^"\\]|\\.)*")`)
	content = certPattern.ReplaceAllString(content, `idp_public_certs = [$1]`)

	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert config block to object attribute
	m.convertBlockToObject(block, "config")

	// Convert scim_config block to object attribute
	m.convertBlockToObject(block, "scim_config")

	// Ensure config exists (required in v5, even if empty for OneTimePin)
	if body.GetAttribute("config") == nil {
		// Check if there's no config block either
		hasConfigBlock := false
		for _, b := range body.Blocks() {
			if b.Type() == "config" {
				hasConfigBlock = true
				break
			}
		}
		if !hasConfigBlock {
			// Create empty config object
			tokens := hclwrite.Tokens{
				{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
				{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
			}
			body.SetAttributeRaw("config", tokens)
		}
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) convertBlockToObject(parentBlock *hclwrite.Block, blockName string) {
	body := parentBlock.Body()
	blocks := body.Blocks()

	for _, b := range blocks {
		if b.Type() == blockName {
			// Convert block attributes to object items
			attrs := b.Body().Attributes()
			if len(attrs) == 0 {
				// Empty block -> empty object
				tokens := hclwrite.Tokens{
					{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
					{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
				}
				body.SetAttributeRaw(blockName, tokens)
			} else {
				// Create object expression from block attributes
				var objTokens hclwrite.Tokens
				objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
				objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

				// Sort attribute names for consistent output
				var attrNames []string
				for name := range attrs {
					attrNames = append(attrNames, name)
				}
				sort.Strings(attrNames)

				for _, name := range attrNames {
					attr := attrs[name]
					// Add attribute name
					objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    " + name)})

					// Calculate padding for alignment (align to longest attribute name)
					padding := 18 - len(name)
					if padding < 1 {
						padding = 1
					}
					for i := 0; i < padding; i++ {
						objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
					}

					objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
					objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})

					// Add attribute value tokens
					objTokens = append(objTokens, attr.Expr().BuildTokens(hclwrite.Tokens{})...)
					objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
				}

				objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")})
				objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
				body.SetAttributeRaw(blockName, objTokens)
			}

			// Remove the original block
			body.RemoveBlock(b)
			break
		}
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformation needed
	return stateJSON.String(), nil
}
