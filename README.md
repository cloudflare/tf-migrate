# tf-migrate

A Terraform configuration and state migration tool that transforms HCL configuration files and JSON state files through a customizable pipeline architecture.

## Overview

tf-migrate is a command-line tool that provides a framework for migrating Terraform configurations between different provider versions or resource schemas. It processes both configuration files (.tf) and state files (terraform.tfstate) using a pipeline of transformation handlers.


## Usage

### Basic Commands

```bash
# Migrate all .tf files in current directory
tf-migrate migrate

# Migrate specific directory
tf-migrate --config-dir ./terraform migrate

# Migrate with state file
tf-migrate --config-dir ./terraform --state-file terraform.tfstate migrate

# Preview changes without modifying files
tf-migrate --dry-run migrate

# Output to different directory
tf-migrate migrate --output-dir ./migrated --output-state ./migrated/terraform.tfstate

# Skip backup creation
tf-migrate migrate --backup=false
```

### Command-Line Flags

#### Global Flags
- `--config-dir` - Directory containing Terraform configuration files (default: current directory)
- `--state-file` - Path to Terraform state file (optional)
- `--resources` - Comma-separated list of resources to migrate (not yet implemented)
- `--dry-run` - Preview changes without modifying files
- `-v, --verbose` - Enable verbose output
- `--debug` - Enable debug output
- `-q, --quiet` - Suppress all output except errors

#### Migrate Command Flags
- `--output-dir` - Output directory for migrated configuration files (default: in-place)
- `--output-state` - Output path for migrated state file (default: in-place)
- `--backup` - Create backup of original files before migration (default: true)

## Architecture

### Pipeline Structure

The tool uses a pipeline architecture with two separate pipelines:

#### Configuration Pipeline (HCL Files)
1. **PreprocessHandler** - String-level transformations before parsing
2. **ParseHandler** - Parse HCL text into Abstract Syntax Tree (AST)
3. **ResourceTransformHandler** - Transform resource blocks in the AST
4. **FormatterHandler** - Format AST back to HCL text

#### State Pipeline (JSON Files)
1. **StateTransformHandler** - Transform resources in state JSON
2. **StateFormatterHandler** - Pretty-print JSON output

### Key Components

- **Handlers** - Implement the Chain of Responsibility pattern for pipeline stages
- **ResourceTransformer** - Strategy pattern interface for resource-specific transformations
- **StrategyRegistry** - Thread-safe registry for transformation strategies
- **PipelineBuilder** - Fluent builder for custom pipeline construction

## Extending tf-migrate

### Creating a Custom Resource Transformer

Implement the `ResourceTransformer` interface:

```go
type MyResourceTransformer struct{}

func (t *MyResourceTransformer) CanHandle(resourceType string) bool {
    return resourceType == "my_resource_type"
}

func (t *MyResourceTransformer) GetResourceType() string {
    return "my_resource_type"
}

func (t *MyResourceTransformer) Preprocess(content string) string {
    // Optional: String-level transformations
    return content
}

func (t *MyResourceTransformer) TransformConfig(block *hclwrite.Block) (*interfaces.TransformResult, error) {
    // Transform HCL configuration
    // Options:
    // - In-place: return {Blocks: [modifiedBlock], RemoveOriginal: false}
    // - Split: return {Blocks: [block1, block2], RemoveOriginal: true}
    // - Remove: return {Blocks: [], RemoveOriginal: true}
    return &interfaces.TransformResult{
        Blocks:         []*hclwrite.Block{block},
        RemoveOriginal: false,
    }, nil
}

func (t *MyResourceTransformer) TransformState(json gjson.Result, resourcePath string) (string, error) {
    // Transform state JSON
    return modifiedJSON, nil
}
```

### Registering Transformers

```go
func init() {
    resources.RegisterResourceFactory("my_resource_type", func() interfaces.ResourceTransformer {
        return &MyResourceTransformer{}
    })
}
```

### Creating Custom Pipelines

```go
pipeline := pipeline.NewPipelineBuilder(registry).
    With(pipeline.Preprocess).
    WithHandler(customHandler).
    With(pipeline.Parse).
    With(pipeline.TransformResources).
    With(pipeline.Format).
    Build()
```

## Project Structure

```
tf-migrate/
├── cmd/tf-migrate/         # CLI application
│   ├── main.go
│   └── root/
│       ├── root.go         # Command definitions
│       └── migrate.go      # Migration logic
├── internal/
│   ├── handlers/           # Pipeline handlers
│   ├── interfaces/         # Core interfaces
│   ├── pipeline/           # Pipeline orchestration
│   ├── registry/           # Strategy registry
│   ├── resources/          # Resource transformers
│   ├── hcl/               # HCL utilities
│   └── logger/            # Logging
└── go.mod
```

## Getting Started

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o tf-migrate cmd/tf-migrate/main.go
```

## Contributing

### Adding New Resource Transformers

1. Create a new file in `internal/resources/` for your resource type
2. Implement the `ResourceTransformer` interface
3. Register your transformer in an `init()` function
4. Add tests in `internal/resources/` with `_test.go` suffix
5. Submit a pull request with your changes

### Development Guidelines

- Follow Go conventions and idioms
- Add tests for new functionality
- Use the existing handler pattern for new pipeline stages
- Update documentation for new features
- Ensure all tests pass before submitting PRs
