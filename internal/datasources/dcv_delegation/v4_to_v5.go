package dcv_delegation

import (
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of the cloudflare_dcv_delegation
// datasource from v4 to v5.
//
// The only input, zone_id, is unchanged (required in v4, optional in v5), so
// the block itself needs no transformation. What changes are the computed
// attributes, so the migration is about rewriting references to them:
//
//	id       → uuid  (same value: v4 set id to the delegation UUID)
//	hostname → removed in v5
//
// v4 computed hostname as "<uuid>.dcv.cloudflare.com". v5 has no equivalent
// attribute, so references to hostname are reported as warnings with the
// expression to use instead, rather than rewritten.
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for the cloudflare_dcv_delegation
// datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_dcv_delegation", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_dcv_delegation"
}

// GetAttributeRenames returns attribute renames for cross-file reference
// updates. v4 exposed the delegation UUID as id; v5 exposes it as uuid and has
// no id attribute.
func (m *V4ToV5Migrator) GetAttributeRenames() []transform.AttributeRename {
	return []transform.AttributeRename{
		{
			ResourceType: "data.cloudflare_dcv_delegation",
			OldAttribute: "id",
			NewAttribute: "uuid",
		},
	}
}

// GetInvalidAttributeReferences implements the InvalidAttributeReferenceDetector
// interface. hostname has no v5 attribute, so each reference is reported with
// the expression that reproduces the v4 value.
func (m *V4ToV5Migrator) GetInvalidAttributeReferences() []transform.InvalidAttributeReference {
	hostname := `'hostname' is not a valid attribute of the v5 cloudflare_dcv_delegation datasource.

  In v4 it was computed as "<uuid>.dcv.cloudflare.com". Build the same value
  from the uuid attribute:

    "${data.cloudflare_dcv_delegation.<name>.uuid}.dcv.cloudflare.com"`

	return []transform.InvalidAttributeReference{
		{
			ResourceType: "data.cloudflare_dcv_delegation",
			Attribute:    "hostname",
			Suggestion:   hostname,
		},
	}
}

// GetResourceRename implements the ResourceRenamer interface.
// The cloudflare_dcv_delegation datasource keeps its name in v5.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"data.cloudflare_dcv_delegation"}, "data.cloudflare_dcv_delegation"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_dcv_delegation"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for the dcv_delegation datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
//
// zone_id is the only input and is unchanged, so the block body is returned
// as-is. The migration happens through GetAttributeRenames and
// GetInvalidAttributeReferences, which operate on references to this
// datasource.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func init() {
	// Register the migrator on package initialization
	NewV4ToV5Migrator()
}
