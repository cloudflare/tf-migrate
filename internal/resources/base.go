package resources

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// ResourceMigrator is an interface that concrete migrators implement
// to provide custom transformation behavior for specific resource types
type ResourceMigrator interface {
	CanHandleResource(resourceType string) bool
	TransformResourceConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error)
	TransformResourceState(ctx *transform.Context, json gjson.Result, resourcePath string) (string, error)
	PreprocessResource(content string) string
}

// BaseResourceTransformer provides common functionality for all resource transformers
type BaseResourceTransformer struct {
	ResourceType string
	Migrator     ResourceMigrator // Reference to the concrete migrator
}

// NewBaseResourceTransformer creates a new base transformer for a specific resource type
func NewBaseResourceTransformer(resourceType string, migrator ResourceMigrator) *BaseResourceTransformer {
	return &BaseResourceTransformer{
		ResourceType: resourceType,
		Migrator:     migrator,
	}
}

// CanHandle determines if this transformer can handle the given resource type
func (t *BaseResourceTransformer) CanHandle(resourceType string) bool {
	return t.Migrator.CanHandleResource(resourceType)
}

// GetResourceType returns the primary resource type this transformer handles
func (t *BaseResourceTransformer) GetResourceType() string {
	return t.ResourceType
}

// TransformConfig handles configuration file transformations
func (t *BaseResourceTransformer) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return t.Migrator.TransformResourceConfig(ctx, block)
}

// TransformState handles state file transformations
func (t *BaseResourceTransformer) TransformState(ctx *transform.Context, json gjson.Result, resourcePath string) (string, error) {
	return t.Migrator.TransformResourceState(ctx, json, resourcePath)
}

// Preprocess handles string-level transformations before HCL parsing
func (t *BaseResourceTransformer) Preprocess(content string) string {
	return t.Migrator.PreprocessResource(content)
}
