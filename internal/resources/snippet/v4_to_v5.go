package snippet

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_snippet", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_snippet"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_snippet"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Rename "name" to "snippet_name"
	if nameAttr := body.GetAttribute("name"); nameAttr != nil {
		tokens := nameAttr.Expr().BuildTokens(nil)
		body.SetAttributeRaw("snippet_name", tokens)
		body.RemoveAttribute("name")
	}

	// 2. Move "main_module" to nested "metadata" object
	if mainModuleAttr := body.GetAttribute("main_module"); mainModuleAttr != nil {
		// Get the raw tokens from the original expression to preserve variable references
		mainModuleTokens := mainModuleAttr.Expr().BuildTokens(nil)

		// Build the metadata object with raw tokens to preserve the original expression
		var metadataTokens hclwrite.Tokens
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte("{"),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("    main_module"),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(" "),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte("="),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(" "),
		})
		// Append the original expression tokens (preserves variables, literals, etc.)
		metadataTokens = append(metadataTokens, mainModuleTokens...)
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("  "),
		})
		metadataTokens = append(metadataTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte("}"),
		})

		// Set metadata attribute using raw tokens
		body.SetAttributeRaw("metadata", metadataTokens)
		body.RemoveAttribute("main_module")
	}

	// 3. Convert "files" blocks to list attribute
	filesBlocks := body.Blocks()
	var blocksToRemove []*hclwrite.Block
	var hasFiles bool

	// First pass: check if we have files blocks
	for _, filesBlock := range filesBlocks {
		if filesBlock.Type() == "files" {
			hasFiles = true
			blocksToRemove = append(blocksToRemove, filesBlock)
		}
	}

	// Remove all files blocks
	for _, filesBlock := range blocksToRemove {
		body.RemoveBlock(filesBlock)
	}

	// Add files as a list attribute if we have any files
	if hasFiles {
		// Build the files list manually using raw tokens to preserve formatting
		var filesTokens hclwrite.Tokens
		filesTokens = append(filesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte("["),
		})
		filesTokens = append(filesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})

		// Process each files block
		for i, filesBlock := range blocksToRemove {
			if filesBlock.Type() == "files" {
				// Add indentation
				filesTokens = append(filesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("    "),
				})
				filesTokens = append(filesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenOBrace,
					Bytes: []byte("{"),
				})
				filesTokens = append(filesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})

				fileBody := filesBlock.Body()

				// Add name attribute
				if nameAttr := fileBody.GetAttribute("name"); nameAttr != nil {
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte("      name"),
					})
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte("    "),
					})
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenEqual,
						Bytes: []byte("="),
					})
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte(" "),
					})
					// Append the raw tokens from the name expression
					filesTokens = append(filesTokens, nameAttr.Expr().BuildTokens(nil)...)
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenNewline,
						Bytes: []byte("\n"),
					})
				}

				// Add content attribute - preserve heredoc format
				if contentAttr := fileBody.GetAttribute("content"); contentAttr != nil {
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte("      content"),
					})
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte(" "),
					})
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenEqual,
						Bytes: []byte("="),
					})
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte(" "),
					})
					// Append the raw tokens from the content expression (preserves heredoc)
					filesTokens = append(filesTokens, contentAttr.Expr().BuildTokens(nil)...)
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenNewline,
						Bytes: []byte("\n"),
					})
				}

				// Close object
				filesTokens = append(filesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("    "),
				})
				filesTokens = append(filesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenCBrace,
					Bytes: []byte("}"),
				})

				// Add comma if not the last item
				if i < len(blocksToRemove)-1 {
					filesTokens = append(filesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenComma,
						Bytes: []byte(","),
					})
				}
				filesTokens = append(filesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
		}

		// Close list
		filesTokens = append(filesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("  "),
		})
		filesTokens = append(filesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte("]"),
		})

		// Set the files attribute using raw tokens
		body.SetAttributeRaw("files", filesTokens)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("type").Exists() && stateJSON.Get("instances").Exists() {
		return m.transformFullResource(result, stateJSON)
	}

	if !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	return m.transformSingleInstance(result, stateJSON), nil
}

func (m *V4ToV5Migrator) transformFullResource(result string, resource gjson.Result) (string, error) {
	resourceType := resource.Get("type").String()
	if resourceType != "cloudflare_snippet" {
		return result, nil
	}

	instances := resource.Get("instances")
	instances.ForEach(func(key, instance gjson.Result) bool {
		instPath := "instances." + key.String()
		instJSON := instance.String()
		transformedInst := m.transformSingleInstance(instJSON, instance)
		result, _ = sjson.SetRaw(result, instPath, transformedInst)
		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	if !instance.Get("attributes").Exists() {
		return result
	}

	// 1. Rename "name" to "snippet_name"
	if instance.Get("attributes.name").Exists() {
		nameValue := instance.Get("attributes.name").String()
		result, _ = sjson.Set(result, "attributes.snippet_name", nameValue)
		result, _ = sjson.Delete(result, "attributes.name")
	}

	// 2. Move "main_module" to nested "metadata.main_module"
	if instance.Get("attributes.main_module").Exists() {
		mainModuleValue := instance.Get("attributes.main_module").String()
		result, _ = sjson.Set(result, "attributes.metadata.main_module", mainModuleValue)
		result, _ = sjson.Delete(result, "attributes.main_module")
	}

	return result
}
