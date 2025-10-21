package factory

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal/core/transform"
)

// Default is the default transformer factory instance
var Default = &TransformerFactory{}

// TransformerFactory creates transformers with common patterns
type TransformerFactory struct{}

// Rename creates a transformer that renames a resource type
func (f *TransformerFactory) Rename(oldType, newType string) transform.Transformer {
	return &transform.BaseTransformer{
		ResourceType:       oldType,
		TargetResourceType: newType,
		Preprocessor: func(content string) string {
			// Replace resource type declarations
			content = strings.ReplaceAll(content, 
				fmt.Sprintf(`resource "%s"`, oldType),
				fmt.Sprintf(`resource "%s"`, newType))
			
			// Replace references to the resource
			content = strings.ReplaceAll(content,
				fmt.Sprintf("%s.", oldType),
				fmt.Sprintf("%s.", newType))
			
			return content
		},
		StateTransformer: func(json gjson.Result, resourcePath string) (string, error) {
			// No instance-level changes needed for simple rename
			// The resource type change is handled by the state processor
			return json.String(), nil
		},
	}
}

// RenameAttribute creates a transformer that renames a single attribute
func (f *TransformerFactory) RenameAttribute(resourceType, oldAttr, newAttr string) transform.Transformer {
	return &transform.BaseTransformer{
		ResourceType: resourceType,
		ConfigTransformer: func(block *hclwrite.Block) (*transform.TransformResult, error) {
			body := block.Body()
			if attr := body.GetAttribute(oldAttr); attr != nil {
				tokens := attr.Expr().BuildTokens(nil)
				body.SetAttributeRaw(newAttr, tokens)
				body.RemoveAttribute(oldAttr)
			}
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
		StateTransformer: func(json gjson.Result, resourcePath string) (string, error) {
			modifiedJSON := json.String()
			
			// Get old attribute value
			oldPath := "attributes." + oldAttr
			if json.Get(oldPath).Exists() {
				value := json.Get(oldPath).Value()
				
				// Set new attribute
				newPath := "attributes." + newAttr
				modifiedJSON, _ = sjson.Set(modifiedJSON, newPath, value)
				
				// Delete old attribute
				modifiedJSON, _ = sjson.Delete(modifiedJSON, oldPath)
			}
			
			return modifiedJSON, nil
		},
	}
}

// RenameAttributes creates a transformer that renames multiple attributes
func (f *TransformerFactory) RenameAttributes(resourceType string, renames map[string]string) transform.Transformer {
	return &transform.BaseTransformer{
		ResourceType: resourceType,
		ConfigTransformer: func(block *hclwrite.Block) (*transform.TransformResult, error) {
			body := block.Body()
			for oldName, newName := range renames {
				if attr := body.GetAttribute(oldName); attr != nil {
					tokens := attr.Expr().BuildTokens(nil)
					body.SetAttributeRaw(newName, tokens)
					body.RemoveAttribute(oldName)
				}
			}
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
		StateTransformer: func(json gjson.Result, resourcePath string) (string, error) {
			modifiedJSON := json.String()
			
			for oldAttr, newAttr := range renames {
				oldPath := "attributes." + oldAttr
				if json.Get(oldPath).Exists() {
					value := json.Get(oldPath).Value()
					
					newPath := "attributes." + newAttr
					modifiedJSON, _ = sjson.Set(modifiedJSON, newPath, value)
					modifiedJSON, _ = sjson.Delete(modifiedJSON, oldPath)
				}
			}
			
			return modifiedJSON, nil
		},
	}
}

// RemoveAttributes creates a transformer that removes specified attributes
func (f *TransformerFactory) RemoveAttributes(resourceType string, attrs ...string) transform.Transformer {
	return &transform.BaseTransformer{
		ResourceType: resourceType,
		ConfigTransformer: func(block *hclwrite.Block) (*transform.TransformResult, error) {
			body := block.Body()
			for _, attr := range attrs {
				body.RemoveAttribute(attr)
			}
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
		StateTransformer: func(json gjson.Result, resourcePath string) (string, error) {
			modifiedJSON := json.String()
			
			for _, attr := range attrs {
				path := "attributes." + attr
				modifiedJSON, _ = sjson.Delete(modifiedJSON, path)
			}
			
			return modifiedJSON, nil
		},
	}
}

// Composite creates a transformer that applies multiple transformations
func (f *TransformerFactory) Composite(resourceType string, transformers ...transform.Transformer) transform.Transformer {
	return &transform.BaseTransformer{
		ResourceType: resourceType,
		ConfigTransformer: func(block *hclwrite.Block) (*transform.TransformResult, error) {
			// Apply each transformer in sequence
			currentBlock := block
			for _, t := range transformers {
				if t.CanHandle(resourceType) {
					result, err := t.Config(currentBlock)
					if err != nil {
						return nil, err
					}
					if result != nil && len(result.Blocks) > 0 {
						currentBlock = result.Blocks[0]
					}
				}
			}
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{currentBlock},
				RemoveOriginal: false,
			}, nil
		},
		StateTransformer: func(json gjson.Result, resourcePath string) (string, error) {
			modifiedJSON := json.String()
			
			// Apply each transformer's state transformation
			for _, t := range transformers {
				if t.CanHandle(resourceType) {
					result, err := t.State(gjson.Parse(modifiedJSON), resourcePath)
					if err != nil {
						return "", err
					}
					modifiedJSON = result
				}
			}
			
			return modifiedJSON, nil
		},
		Preprocessor: func(content string) string {
			// Apply all preprocessors
			for _, t := range transformers {
				content = t.Preprocess(content)
			}
			return content
		},
	}
}

// RenameAndModifyAttributes combines renaming a resource and modifying attributes
func (f *TransformerFactory) RenameAndModifyAttributes(oldType, newType string, attrRenames map[string]string) transform.Transformer {
	// Create a single transformer that handles both the rename and attribute changes
	return &transform.BaseTransformer{
		ResourceType:       oldType,
		TargetResourceType: newType,
		CanHandleFunc: func(resourceType string) bool {
			// Handle both old and new resource types since preprocessing changes the type
			return resourceType == oldType || resourceType == newType
		},
		Preprocessor: func(content string) string {
			// Replace resource type declarations
			content = strings.ReplaceAll(content, 
				fmt.Sprintf(`resource "%s"`, oldType),
				fmt.Sprintf(`resource "%s"`, newType))
			
			// Replace references to the resource
			content = strings.ReplaceAll(content,
				fmt.Sprintf("%s.", oldType),
				fmt.Sprintf("%s.", newType))
			
			return content
		},
		ConfigTransformer: func(block *hclwrite.Block) (*transform.TransformResult, error) {
			body := block.Body()
			// Rename attributes
			for oldName, newName := range attrRenames {
				if attr := body.GetAttribute(oldName); attr != nil {
					tokens := attr.Expr().BuildTokens(nil)
					body.SetAttributeRaw(newName, tokens)
					body.RemoveAttribute(oldName)
				}
			}
			return &transform.TransformResult{
				Blocks:         []*hclwrite.Block{block},
				RemoveOriginal: false,
			}, nil
		},
		StateTransformer: func(json gjson.Result, resourcePath string) (string, error) {
			modifiedJSON := json.String()
			
			// Rename attributes in state
			for oldAttr, newAttr := range attrRenames {
				if json.Get("attributes." + oldAttr).Exists() {
					value := json.Get("attributes." + oldAttr).Value()
					modifiedJSON, _ = sjson.Set(modifiedJSON, "attributes."+newAttr, value)
					modifiedJSON, _ = sjson.Delete(modifiedJSON, "attributes."+oldAttr)
				}
			}
			
			return modifiedJSON, nil
		},
	}
}