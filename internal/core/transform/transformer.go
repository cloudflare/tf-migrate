package transform

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

// TransformResult defines the output of a resource transformation
type TransformResult struct {
	Blocks         []*hclwrite.Block
	RemoveOriginal bool
}

// Transformer defines the interface for resource-specific transformations
type Transformer interface {
	CanHandle(resourceType string) bool
	// Config handles configuration transformations:
	// - In-place: return {Blocks: [modifiedBlock], RemoveOriginal: false}
	// - Split: return {Blocks: newBlocks, RemoveOriginal: true}
	// - Remove: return {Blocks: nil, RemoveOriginal: true}
	Config(block *hclwrite.Block) (*TransformResult, error)
	State(json gjson.Result, resourcePath string) (string, error)
	GetResourceType() string
	// GetTargetResourceType returns the new resource type if it changes, or empty string if unchanged
	GetTargetResourceType() string
	// Preprocess for string-level transformations before parsing
	Preprocess(content string) string
}