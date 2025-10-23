package transform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

// Context carries data through the transformation pipeline
type Context struct {
	Content       []byte
	Filename      string
	AST           *hclwrite.File
	StateJSON     string
	Diagnostics   hcl.Diagnostics
	Metadata      map[string]interface{}
	Resources     []string
	SourceVersion string // Source provider version (e.g., "v4")
	TargetVersion string // Target provider version (e.g., "v5")
}

// TransformResult represents the result of a resource transformation
type TransformResult struct {
	Blocks         []*hclwrite.Block
	RemoveOriginal bool
}

// ResourceTransformer defines the interface for resource-specific transformations
// Each resource implements the ResourceTransformer interface which defines how that resource
// handles the migration between major versions
type ResourceTransformer interface {
	CanHandle(resourceType string) bool
	// TransformConfig handles transformations:
	// - In-place: return {Blocks: [modifiedBlock], RemoveOriginal: false}
	// - Split: return {Blocks: newBlocks, RemoveOriginal: true}
	// - Remove: return {Blocks: nil, RemoveOriginal: true}
	TransformConfig(ctx *Context, block *hclwrite.Block) (*TransformResult, error)
	TransformState(ctx *Context, json gjson.Result, resourcePath string) (string, error)
	GetResourceType() string
	// Preprocess for string-level transformations before parsing
	Preprocess(content string) string
}

// Provider specifies the interface for a migrator provider
// This is used to provide a way to get migrators for a given resource type
// a migrator defines the strategy which a resource uses to migrate the resource
// from a source version to target version
type Provider interface {
	GetMigrator(resourceType string, sourceVersion string, targetVersion string) ResourceTransformer
	GetAllMigrators(sourceVersion string, targetVersion string, resources ...string) []ResourceTransformer
}

type DefaultMigratorProvider struct {
	getFunc    func(string, string, string) ResourceTransformer
	getAllFunc func(string, string, ...string) []ResourceTransformer
}

func NewMigratorProvider(
	getFunc func(string, string, string) ResourceTransformer,
	getAllFunc func(string, string, ...string) []ResourceTransformer,
) Provider {
	return &DefaultMigratorProvider{
		getFunc:    getFunc,
		getAllFunc: getAllFunc,
	}
}

func (p *DefaultMigratorProvider) GetMigrator(resourceType string, sourceVersion string, targetVersion string) ResourceTransformer {
	if p.getFunc != nil {
		return p.getFunc(resourceType, sourceVersion, targetVersion)
	}
	return nil
}

func (p *DefaultMigratorProvider) GetAllMigrators(sourceVersion string, targetVersion string, resources ...string) []ResourceTransformer {
	if p.getAllFunc != nil {
		return p.getAllFunc(sourceVersion, targetVersion, resources...)
	}
	return []ResourceTransformer{}
}
