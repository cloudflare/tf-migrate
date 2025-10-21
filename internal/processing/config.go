package processing

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	
	"github.com/cloudflare/tf-migrate/internal/core"
)

// ProcessConfig transforms Terraform configuration files
func ProcessConfig(content []byte, filename string, registry *core.Registry) ([]byte, error) {
	// Step 1: Preprocess - Apply string-level transformations
	processed := preprocess(string(content), registry)
	
	// Step 2: Parse - Convert to AST
	ast, err := parse([]byte(processed), filename)
	if err != nil {
		return nil, core.NewError(core.ParseError).
			WithFile(filename).
			WithOperation("parsing HCL configuration").
			WithCause(err).
			Build()
	}
	
	// Step 3: Transform - Apply resource transformations
	if err := transformResources(ast, registry, filename); err != nil {
		// Error already wrapped by transformResources
		return nil, err
	}
	
	// Step 4: Format - Convert back to HCL
	return format(ast), nil
}

// preprocess applies string-level transformations before parsing
func preprocess(content string, registry *core.Registry) string {
	// Apply all registered preprocessors
	for _, transformer := range registry.GetAll() {
		content = transformer.Preprocess(content)
	}
	return content
}

// parse converts HCL text to AST
func parse(content []byte, filename string) (*hclwrite.File, error) {
	file, diags := hclwrite.ParseConfig(content, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, diags
	}
	return file, nil
}

// transformResources applies transformations to resource blocks
func transformResources(ast *hclwrite.File, registry *core.Registry, filename string) error {
	body := ast.Body()
	blocks := body.Blocks()
	
	var blocksToRemove []*hclwrite.Block
	var blocksToAdd []*hclwrite.Block
	
	// Collect all errors to report them together
	errorList := core.NewErrorList(10)
	
	for _, block := range blocks {
		if block.Type() != "resource" {
			continue
		}
		
		labels := block.Labels()
		if len(labels) < 2 {
			// Resource blocks should have type and name
			errorList.Add(core.ValidationErrorf(
				"resource block missing required labels (expected 2, got %d) in %s",
				len(labels), filename))
			continue
		}
		
		resourceType := labels[0]
		resourceName := labels[1]
		transformer := registry.Find(resourceType)
		if transformer == nil {
			continue // No transformer for this resource type - this is ok
		}
		
		result, err := transformer.Config(block)
		if err != nil {
			errorList.Add(core.NewError(core.TransformError).
				WithResource(resourceType).
				WithFile(filename).
				WithOperation("transforming resource").
				WithContext("resource_name", resourceName).
				WithCause(err).
				Build())
			continue // Try to transform other resources
		}
		
		// Check if result is not nil and handle appropriately
		if result != nil && result.RemoveOriginal {
			blocksToRemove = append(blocksToRemove, block)
			if len(result.Blocks) > 0 {
				blocksToAdd = append(blocksToAdd, result.Blocks...)
			}
		}
	}
	
	// If there were errors, return them
	if errorList.HasErrors() {
		return errorList
	}
	
	// Apply changes
	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}
	for _, block := range blocksToAdd {
		body.AppendBlock(block)
	}
	
	return nil
}

// format converts AST back to formatted HCL
func format(ast *hclwrite.File) []byte {
	return hclwrite.Format(ast.Bytes())
}