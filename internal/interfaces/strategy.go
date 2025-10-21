package interfaces

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type TransformResult struct {
	Blocks         []*hclwrite.Block
	RemoveOriginal bool
}

// ResourceTransformer defines the strategy pattern interface for resource-specific transformations
type ResourceTransformer interface {
	CanHandle(resourceType string) bool
	// TransformConfig handles transformations:
	// - In-place: return {Blocks: [modifiedBlock], RemoveOriginal: false}
	// - Split: return {Blocks: newBlocks, RemoveOriginal: true}
	// - Remove: return {Blocks: nil, RemoveOriginal: true}
	TransformConfig(block *hclwrite.Block) (*TransformResult, error)
	TransformState(json gjson.Result, resourcePath string) (string, error)
	GetResourceType() string
	// Preprocess for string-level transformations before parsing
	Preprocess(content string) string
}
