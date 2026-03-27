package transform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// DiagInfo is an informational diagnostic severity level.
// The hcl package only defines DiagError and DiagWarning; DiagInvalid (0) is
// the zero value and serves as our "info" tier. Diagnostics at this level are
// suppressed by default and only shown with --verbose.
const DiagInfo = hcl.DiagnosticSeverity(0) // same as hcl.DiagInvalid

// Context carries data through the transformation pipeline
type Context struct {
	Content       []byte
	Filename      string
	CFGFile       *hclwrite.File
	CFGFiles      map[string]*hclwrite.File
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
	GetResourceType() string
	// Preprocess for string-level transformations before parsing
	Preprocess(content string) string
}

// PhaseOneTransformer is an optional interface for migrators whose v4 resource
// type has no schema in the v5 provider. These resources require a two-phase
// migration in Atlantis-managed workspaces where `terraform state rm` is disabled:
//
//   - Phase 1: append a `removed {}` block to the file while leaving the rest of
//     the config as v4. Committing and applying this lets Terraform drop the old
//     state entry via its own lifecycle, without any manual state surgery.
//   - Phase 2: run the full migration once the state is clean (the normal
//     `tf-migrate migrate` path).
//
// TransformPhaseOne receives the original v4 block and returns a TransformResult
// whose Blocks contains only the `removed {}` block (RemoveOriginal: false so the
// original resource block is left untouched).
type PhaseOneTransformer interface {
	TransformPhaseOne(ctx *Context, block *hclwrite.Block) (*TransformResult, error)
}

// ResourceRenamer is an optional interface that migrators can implement
// to expose resource type renames. This enables global cross-file reference updates.
//
// For resources with multiple v4 names (e.g., both "cloudflare_tunnel_route" and
// "cloudflare_zero_trust_tunnel_route" mapping to the same v5 name), return all old names
// in the oldTypes slice. This ensures cross-file references using any v4 name are updated.
type ResourceRenamer interface {
	// GetResourceRename returns the old resource type names and the new type name.
	// For resources with multiple v4 names, return all of them in oldTypes.
	// For resources with no rename, return the same name in both oldTypes and newType.
	GetResourceRename() (oldTypes []string, newType string)
}

// AttributeRename represents an attribute name change for a specific resource/datasource type
type AttributeRename struct {
	ResourceType string // The resource/datasource type (e.g., "cloudflare_zones", "data.cloudflare_zones")
	OldAttribute string // The old attribute name (e.g., "zones")
	NewAttribute string // The new attribute name (e.g., "result")
}

// AttributeRenamer is an optional interface that migrators can implement
// to expose attribute renames. This enables global cross-file attribute reference updates.
type AttributeRenamer interface {
	// GetAttributeRenames returns a list of attribute renames for this migrator
	// Each rename specifies the resource type and the old/new attribute names
	GetAttributeRenames() []AttributeRename
}

// MigrationProvider specifies the interface for a migrator provider
// This is used to provide a way to get migrators for a given resource type
// a migrator defines the strategy which a resource uses to migrate the resource
// from a source version to target version
type MigrationProvider interface {
	GetMigrator(resourceType string, sourceVersion string, targetVersion string) ResourceTransformer
	GetAllMigrators(sourceVersion string, targetVersion string, resources ...string) []ResourceTransformer
}

type DefaultMigratorProvider struct {
	getFunc    func(string, string, string) ResourceTransformer
	getAllFunc func(string, string, ...string) []ResourceTransformer
}

func NewMigrationProvider(
	getFunc func(string, string, string) ResourceTransformer,
	getAllFunc func(string, string, ...string) []ResourceTransformer,
) MigrationProvider {
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
