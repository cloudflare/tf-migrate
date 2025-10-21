# Resource Transformers

This directory contains specific resource transformation implementations for different Terraform providers.

## Structure

Each transformer implements the `transform.Transformer` interface and handles the migration of a specific resource type.

## Example: CloudflareDNSTransformer

The `cloudflare_dns.go` file demonstrates how to implement a resource transformer:

1. **Preprocess**: String-level transformations before HCL parsing (e.g., renaming resource types)
2. **Config**: AST-level transformations for configuration files (e.g., renaming attributes)
3. **State**: JSON transformations for state files (e.g., updating resource metadata)

## Creating New Transformers

To create a new transformer:

1. Create a new file: `provider_resource.go`
2. Implement the `transform.Transformer` interface
3. Embed `transformers.BaseTransformer` for default implementations
4. Register it in `internal/transformers/registry.go`

Example structure:
```go
type MyResourceTransformer struct {
    transformers.BaseTransformer
}

func NewMyResourceTransformer() *MyResourceTransformer {
    return &MyResourceTransformer{
        BaseTransformer: transformers.BaseTransformer{
            ResourceType: "provider_old_resource",
        },
    }
}

// Implement required methods...
```

## Common Transformation Patterns

### Resource Renaming
```go
func (t *Transformer) Preprocess(content string) string {
    return strings.ReplaceAll(content, "old_resource", "new_resource")
}
```

### Attribute Migration
```go
func (t *Transformer) Config(block *hclwrite.Block) (*transform.TransformResult, error) {
    body := block.Body()
    if attr := body.GetAttribute("old_attr"); attr != nil {
        tokens := attr.Expr().BuildTokens(nil)
        body.SetAttributeRaw("new_attr", tokens)
        body.RemoveAttribute("old_attr")
    }
    return &transform.TransformResult{
        Blocks: []*hclwrite.Block{block},
        RemoveOriginal: false,
    }, nil
}
```

### Resource Splitting
```go
func (t *Transformer) Config(block *hclwrite.Block) (*transform.TransformResult, error) {
    // Create multiple blocks from one
    block1 := hclwrite.NewBlock("resource", []string{"new_type_1", "name_1"})
    block2 := hclwrite.NewBlock("resource", []string{"new_type_2", "name_2"})
    
    return &transform.TransformResult{
        Blocks: []*hclwrite.Block{block1, block2},
        RemoveOriginal: true,
    }, nil
}
```