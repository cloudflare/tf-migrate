package resources

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/vaishak/tf-migrate/internal/interfaces"
)

// BaseResourceTransformer provides common functionality for all resource transformers
type BaseResourceTransformer struct {
	resourceType  string
	canHandleFunc func(resourceType string) bool

	// Transformation functions (optional - can be set via builder pattern)
	configTransformer func(*hclwrite.Block) (*interfaces.TransformResult, error)
	stateTransformer  func(gjson.Result, string) (string, error)
	preprocessor      func(string) string
}

// NewBaseResourceTransformer creates a new base transformer for a specific resource type
func NewBaseResourceTransformer(resourceType string) *BaseResourceTransformer {
	return &BaseResourceTransformer{
		resourceType: resourceType,
	}
}

// CanHandle determines if this transformer can handle the given resource type
func (t *BaseResourceTransformer) CanHandle(resourceType string) bool {
	if t.canHandleFunc != nil {
		return t.canHandleFunc(resourceType)
	}
	return resourceType == t.resourceType
}

// GetResourceType returns the primary resource type this transformer handles
func (t *BaseResourceTransformer) GetResourceType() string {
	return t.resourceType
}

// TransformConfig handles configuration file transformations
func (t *BaseResourceTransformer) TransformConfig(block *hclwrite.Block) (*interfaces.TransformResult, error) {
	if t.configTransformer != nil {
		return t.configTransformer(block)
	}
	// Default: return unchanged
	return &interfaces.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations
func (t *BaseResourceTransformer) TransformState(json gjson.Result, resourcePath string) (string, error) {
	if t.stateTransformer != nil {
		return t.stateTransformer(json, resourcePath)
	}
	// Default: return unchanged
	return json.String(), nil
}

// Preprocess handles string-level transformations before HCL parsing
func (t *BaseResourceTransformer) Preprocess(content string) string {
	if t.preprocessor != nil {
		return t.preprocessor(content)
	}
	// Default: return unchanged
	return content
}

// SetCanHandleFunc sets a custom function to determine if this transformer can handle a resource type
func (t *BaseResourceTransformer) SetCanHandleFunc(f func(string) bool) *BaseResourceTransformer {
	t.canHandleFunc = f
	return t
}

// SetConfigTransformer sets the config transformation function
func (t *BaseResourceTransformer) SetConfigTransformer(f func(*hclwrite.Block) (*interfaces.TransformResult, error)) *BaseResourceTransformer {
	t.configTransformer = f
	return t
}

// SetStateTransformer sets the state transformation function
func (t *BaseResourceTransformer) SetStateTransformer(f func(gjson.Result, string) (string, error)) *BaseResourceTransformer {
	t.stateTransformer = f
	return t
}

// SetPreprocessor sets the preprocessing function
func (t *BaseResourceTransformer) SetPreprocessor(f func(string) string) *BaseResourceTransformer {
	t.preprocessor = f
	return t
}