# tf-migrate

A command-line tool for migrating Terraform configurations and state files between different provider versions or resource schemas.

## What it does

tf-migrate helps you update your Terraform files when:
- A provider changes resource names or attributes
- You need to split one resource into multiple resources
- You need to update deprecated resource configurations
- You need to transform both `.tf` configuration files and `terraform.tfstate` files

## Installation

```bash
go build -o tf-migrate cmd/tf-migrate/main.go
```

## Usage

### Basic Migration

```bash
# Migrate all .tf files in current directory
tf-migrate migrate

# Migrate a specific directory
tf-migrate --config-dir ./terraform migrate

# Migrate with state file
tf-migrate --config-dir ./terraform --state-file terraform.tfstate migrate

# Preview changes without modifying files (dry run)
tf-migrate --dry-run migrate

# Output to different directory instead of in-place
tf-migrate migrate --output-dir ./migrated

# Skip backup creation (not recommended)
tf-migrate migrate --backup=false
```

### Command-Line Options

**Global Flags:**
- `--config-dir` - Directory containing Terraform files (default: current directory)
- `--state-file` - Path to terraform.tfstate file (optional)
- `--dry-run` - Preview changes without modifying files
- `-v, --verbose` - Enable verbose output
- `--debug` - Enable debug output
- `-q, --quiet` - Suppress output except errors

**Migrate Command Flags:**
- `--output-dir` - Output directory for migrated files (default: in-place)
- `--output-state` - Output path for migrated state (default: in-place)
- `--backup` - Create `.backup` files before migration (default: true)

## How it Works

The tool processes files through two pipelines:

### Configuration Pipeline (for .tf files)
1. **Preprocess** - Apply string transformations before parsing
2. **Parse** - Convert HCL to Abstract Syntax Tree (AST)
3. **Transform** - Modify resources in the AST
4. **Format** - Convert AST back to formatted HCL

### State Pipeline (for .tfstate files)
1. **Transform** - Modify resource instances in JSON
2. **Format** - Pretty-print the JSON output

## Adding Resource Transformers

**Note:** No resource transformers are implemented yet. This tool provides the framework for transformations.

### Step 1: Create Your Transformer

Create a new file in `internal/resources/` for your resource type:

```go
package resources

import (
    "github.com/hashicorp/hcl/v2/hclwrite"
    "github.com/tidwall/gjson"
    "github.com/cloudflare/tf-migrate/internal/transforms"
)

type DNSRecordTransformer struct {
    BaseResourceTransformer
}

func NewDNSRecordTransformer() *DNSRecordTransformer {
    return &DNSRecordTransformer{
        BaseResourceTransformer: BaseResourceTransformer{
            ResourceType: "cloudflare_record", // Old resource name
            
            // Transform the HCL configuration
            ConfigTransformer: func(block *hclwrite.Block) (*transforms.TransformResult, error) {
                // Example: Rename resource type
                newBlock := hclwrite.NewBlock("resource", []string{"cloudflare_dns_record", block.Labels()[1]})
                
                // Copy and transform attributes
                body := block.Body()
                newBody := newBlock.Body()
                
                for name, attr := range body.Attributes() {
                    // Example: Rename 'domain' to 'zone_id'
                    if name == "domain" {
                        newBody.SetAttributeRaw("zone_id", attr.Expr().BuildTokens(nil))
                    } else {
                        newBody.SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
                    }
                }
                
                return &transform.TransformResult{
                    Blocks:         []*hclwrite.Block{newBlock},
                    RemoveOriginal: true, // Remove old block, add new one
                }, nil
            },
            
            // Transform the state file
            StateTransformer: func(json gjson.Result, path string) (string, error) {
                // Transform the JSON representation of the resource
                // Return the modified JSON string
                return json.String(), nil
            },
        },
    }
}

// CanHandle determines if this transformer handles the given resource type
func (t *DNSRecordTransformer) CanHandle(resourceType string) bool {
    return resourceType == "cloudflare_record" || resourceType == "cloudflare_dns_record"
}
```

### Step 2: Register Your Transformer

Add your transformer to `internal/transformers/registry.go`:

```go
func CreateRegistry(resourceFilter ...string) *registry.Registry {
    reg := registry.NewRegistry()
    
    // Add your transformer here
    reg.Register(NewDNSRecordTransformer())
    // reg.Register(NewLoadBalancerTransformer())
    // reg.Register(NewFirewallTransformer())
    
    return reg
}
```

That's it! Your transformer will now be used when processing files.

## Project Structure

```
tf-migrate/
├── cmd/tf-migrate/         # CLI application
│   └── main.go            # Entry point with command definitions
├── internal/
│   ├── migrate/           # Core migration logic
│   ├── processor/         # Processing logic
│   ├── registry/          # Transformer registry
│   ├── resources/         # Resource transformers (add yours here)
│   ├── transform/         # Core transformation interfaces
│   ├── transformers/      # Base transformer implementations
│   └── logger/            # Logging utilities
└── go.mod
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o tf-migrate cmd/tf-migrate/main.go
```

### Adding a New Transformer - Complete Example

Let's say Cloudflare changes `cloudflare_load_balancer` to separate resources. Here's how you'd handle it:

```go
// internal/resources/load_balancer.go
package resources

type LoadBalancerTransformer struct {
    transformers.BaseTransformer
}

func NewLoadBalancerTransformer() *LoadBalancerTransformer {
    return &LoadBalancerTransformer{
        BaseTransformer: transformers.BaseTransformer{
            ResourceType: "cloudflare_load_balancer",
            
            ConfigTransformer: func(block *hclwrite.Block) (*transform.TransformResult, error) {
                // Split into two resources: pool and load_balancer
                poolBlock := hclwrite.NewBlock("resource", []string{"cloudflare_load_balancer_pool", block.Labels()[1] + "_pool"})
                lbBlock := hclwrite.NewBlock("resource", []string{"cloudflare_load_balancer", block.Labels()[1]})
                
                // Move pool-related attributes to pool block
                // Keep load balancer attributes in lb block
                // ... transformation logic ...
                
                return &transform.TransformResult{
                    Blocks:         []*hclwrite.Block{poolBlock, lbBlock},
                    RemoveOriginal: true, // Remove original, add two new blocks
                }, nil
            },
        },
    }
}
```

## Contributing

1. Create your transformer in `internal/resources/`
2. Register it in `internal/transformers/registry.go`
3. Add tests for your transformer
4. Submit a pull request

## License

[Your License Here]