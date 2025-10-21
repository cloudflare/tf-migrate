package transform

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

)

// BaseTransformer provides common functionality for all resource transformers
type BaseTransformer struct {
	ResourceType       string
	TargetResourceType string // New resource type if renamed, empty otherwise

	// Optional custom functions - set these directly
	CanHandleFunc     func(resourceType string) bool
	ConfigTransformer func(*hclwrite.Block) (*TransformResult, error)
	StateTransformer  func(gjson.Result, string) (string, error)
	Preprocessor      func(string) string
}

// NewBaseTransformer creates a new base transformer for a specific resource type
func NewBaseTransformer(resourceType string) *BaseTransformer {
	return &BaseTransformer{
		ResourceType: resourceType,
	}
}

// CanHandle determines if this transformer can handle the given resource type
func (t *BaseTransformer) CanHandle(resourceType string) bool {
	if t.CanHandleFunc != nil {
		return t.CanHandleFunc(resourceType)
	}
	return resourceType == t.ResourceType
}

// GetResourceType returns the primary resource type this transformer handles
func (t *BaseTransformer) GetResourceType() string {
	return t.ResourceType
}

// GetTargetResourceType returns the new resource type if it changes, or empty string if unchanged
func (t *BaseTransformer) GetTargetResourceType() string {
	return t.TargetResourceType
}

// TransformConfig handles configuration file transformations
func (t *BaseTransformer) Config(block *hclwrite.Block) (*TransformResult, error) {
	if t.ConfigTransformer != nil {
		return t.ConfigTransformer(block)
	}
	
	// No transformer set - return the block unchanged
	return &TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations
func (t *BaseTransformer) State(json gjson.Result, resourcePath string) (string, error) {
	if t.StateTransformer != nil {
		return t.StateTransformer(json, resourcePath)
	}
	
	// No transformer set - return the JSON unchanged
	return json.String(), nil
}

// Preprocess handles string-level transformations before HCL parsing
func (t *BaseTransformer) Preprocess(content string) string {
	if t.Preprocessor != nil {
		return t.Preprocessor(content)
	}
	
	// No preprocessor set - return content unchanged
	return content
}