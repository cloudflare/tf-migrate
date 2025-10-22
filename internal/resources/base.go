package resources

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// BaseResourceTransformer provides common functionality for all resource transformers
type BaseResourceTransformer struct {
	ResourceType  string
	CanHandleFunc func(resourceType string) bool

	// Transformation functions (optional - can be set directly)
	ConfigTransformer func(*transform.Context, *hclwrite.Block) (*transform.TransformResult, error)
	StateTransformer  func(*transform.Context, gjson.Result, string) (string, error)
	Preprocessor      func(string) string
}

// NewBaseResourceTransformer creates a new base transformer for a specific resource type
func NewBaseResourceTransformer(resourceType string) *BaseResourceTransformer {
	return &BaseResourceTransformer{
		ResourceType: resourceType,
	}
}

// CanHandle determines if this transformer can handle the given resource type
func (t *BaseResourceTransformer) CanHandle(resourceType string) bool {
	if t.CanHandleFunc != nil {
		return t.CanHandleFunc(resourceType)
	}
	return resourceType == t.ResourceType
}

// GetResourceType returns the primary resource type this transformer handles
func (t *BaseResourceTransformer) GetResourceType() string {
	return t.ResourceType
}

// TransformConfig handles configuration file transformations
// returns the original content if no ConfigTransformer is set
func (t *BaseResourceTransformer) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	if t.ConfigTransformer != nil {
		return t.ConfigTransformer(ctx, block)
	}
	return &transform.TransformResult{
		Blocks: []*hclwrite.Block{block},
	}, nil
}

// TransformState handles state file transformations
// returns the original content if no StateTransformer is set
func (t *BaseResourceTransformer) TransformState(ctx *transform.Context, json gjson.Result, resourcePath string) (string, error) {
	if t.StateTransformer != nil {
		return t.StateTransformer(ctx, json, resourcePath)
	}
	return json.String(), nil
}

// Preprocess handles string-level transformations before HCL parsing
// returns the original content if no Preprocessor is set
func (t *BaseResourceTransformer) Preprocess(content string) string {
	if t.Preprocessor != nil {
		return t.Preprocessor(content)
	}
	return content
}
