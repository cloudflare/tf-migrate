package zero_trust_access_mtls_hostname_settings

import (
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
	internal.RegisterMigrator("cloudflare_access_mutual_tls_hostname_settings", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_mtls_hostname_settings", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_access_mutual_tls_hostname_settings" ||
		resourceType == "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Look for settings blocks (both static and dynamic)
	var settingsBlocks []*hclwrite.Block
	var dynamicSettingsBlock *hclwrite.Block

	for _, childBlock := range body.Blocks() {
		if childBlock.Type() == "settings" {
			settingsBlocks = append(settingsBlocks, childBlock)
		} else if childBlock.Type() == "dynamic" && len(childBlock.Labels()) > 0 && childBlock.Labels()[0] == "settings" {
			dynamicSettingsBlock = childBlock
		}
	}

	// Handle dynamic blocks - convert to for expressions
	if dynamicSettingsBlock != nil {
		m.convertDynamicBlockToForExpression(body, dynamicSettingsBlock)
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// Handle static settings blocks
	if len(settingsBlocks) > 0 {
		m.convertStaticBlocksToList(body, settingsBlocks)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) convertDynamicBlockToForExpression(parentBody *hclwrite.Body, dynBlock *hclwrite.Block) {
	// Extract the for_each expression
	forEachAttr := dynBlock.Body().GetAttribute("for_each")
	if forEachAttr == nil {
		return
	}

	// Get the iterator name (defaults to "settings")
	iteratorName := "settings"
	if iteratorAttr := dynBlock.Body().GetAttribute("iterator"); iteratorAttr != nil {
		tokens := iteratorAttr.Expr().BuildTokens(nil)
		if len(tokens) > 0 {
			iteratorName = string(tokens[0].Bytes)
		}
	}

	// Extract content block
	var contentBlock *hclwrite.Block
	for _, cb := range dynBlock.Body().Blocks() {
		if cb.Type() == "content" {
			contentBlock = cb
			break
		}
	}

	if contentBlock == nil {
		return
	}

	// Extract field values from content block
	hostnameTokens := hclwrite.Tokens{}
	chinaNetworkTokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte("false")},
	}
	clientCertForwardingTokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte("false")},
	}

	if attr := contentBlock.Body().GetAttribute("hostname"); attr != nil {
		hostnameTokens = attr.Expr().BuildTokens(nil)
	}

	if attr := contentBlock.Body().GetAttribute("china_network"); attr != nil {
		chinaNetworkTokens = attr.Expr().BuildTokens(nil)
	}

	if attr := contentBlock.Body().GetAttribute("client_certificate_forwarding"); attr != nil {
		clientCertForwardingTokens = attr.Expr().BuildTokens(nil)
	}

	// Replace iterator references (e.g., "settings.value" with "value")
	hostnameTokens = m.replaceIteratorReferences(hostnameTokens, iteratorName)

	// Build the for expression: [for value in <for_each_expr> : { ... }]
	var tokens hclwrite.Tokens
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("for")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("value")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("in")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})

	// Add the for_each expression
	tokens = append(tokens, forEachAttr.Expr().BuildTokens(nil)...)

	// Add the colon and object
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenColon, Bytes: []byte(":")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	// Add hostname field
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    hostname")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("                      ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, hostnameTokens...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	// Add china_network field
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    china_network")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("                 ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, chinaNetworkTokens...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	// Add client_certificate_forwarding field
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    client_certificate_forwarding")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
	tokens = append(tokens, clientCertForwardingTokens...)
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

	// Close object and list
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

	// Set the settings attribute
	parentBody.SetAttributeRaw("settings", tokens)

	// Remove the dynamic block
	parentBody.RemoveBlock(dynBlock)
}

func (m *V4ToV5Migrator) replaceIteratorReferences(tokens hclwrite.Tokens, iteratorName string) hclwrite.Tokens {
	// Look for patterns like "settings.value" and replace with just "value"
	var result hclwrite.Tokens
	i := 0
	for i < len(tokens) {
		if i+2 < len(tokens) &&
			tokens[i].Type == hclsyntax.TokenIdent &&
			string(tokens[i].Bytes) == iteratorName &&
			tokens[i+1].Type == hclsyntax.TokenDot &&
			tokens[i+2].Type == hclsyntax.TokenIdent &&
			string(tokens[i+2].Bytes) == "value" {
			// Replace "iterator.value" with just "value"
			result = append(result, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("value")})
			i += 3
		} else {
			result = append(result, tokens[i])
			i++
		}
	}
	return result
}

func (m *V4ToV5Migrator) convertStaticBlocksToList(parentBody *hclwrite.Body, settingsBlocks []*hclwrite.Block) {
	// Build list of settings objects
	var objectTokens hclwrite.Tokens
	objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})

	for idx, settingsBlock := range settingsBlocks {
		if idx > 0 {
			// Add comma between objects
			objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
			objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
		}

		objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
		objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})

		// Get attributes in sorted order
		attrs := settingsBlock.Body().Attributes()
		attrNames := []string{"china_network", "client_certificate_forwarding", "hostname"}

		for _, attrName := range attrNames {
			var attrTokens hclwrite.Tokens

			if attr, exists := attrs[attrName]; exists {
				attrTokens = attr.Expr().BuildTokens(nil)
			} else {
				// Provide defaults
				if attrName == "china_network" || attrName == "client_certificate_forwarding" {
					attrTokens = hclwrite.Tokens{
						{Type: hclsyntax.TokenIdent, Bytes: []byte("false")},
					}
				} else {
					continue // Skip if no default
				}
			}

			// Add attribute with proper spacing
			objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    " + attrName)})

			// Calculate padding for alignment
			padding := 33 - len(attrName)
			if padding < 1 {
				padding = 1
			}
			for i := 0; i < padding; i++ {
				objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
			}

			objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
			objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
			objectTokens = append(objectTokens, attrTokens...)
			objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		}

		objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")})
		objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
	}

	objectTokens = append(objectTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

	// Set the settings attribute
	parentBody.SetAttributeRaw("settings", objectTokens)

	// Remove all settings blocks
	for _, settingsBlock := range settingsBlocks {
		parentBody.RemoveBlock(settingsBlock)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformation needed
	return stateJSON.String(), nil
}
