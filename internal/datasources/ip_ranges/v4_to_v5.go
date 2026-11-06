package ip_ranges

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles the migration of the cloudflare_ip_ranges datasource
// from v4 to v5.
//
// The v4 datasource takes no input arguments, so the configuration block itself
// needs no transformation. Everything that changes is on the output side, which
// means the migration is entirely about rewriting cross-file references:
//
//	ipv4_cidr_blocks       → ipv4_cidrs   (verified identical contents)
//	ipv6_cidr_blocks       → ipv6_cidrs   (verified identical contents)
//
// Three v4 attributes have no drop-in v5 replacement and are reported as
// warnings instead of being rewritten:
//
//	cidr_blocks            → concat(ipv4_cidrs, ipv6_cidrs)
//	china_ipv4_cidr_blocks → jdcloud_cidrs, and requires networks = "jdcloud"
//	china_ipv6_cidr_blocks → jdcloud_cidrs, and requires networks = "jdcloud"
//
// The two china_* lists were merged into the single jdcloud_cidrs list in v5,
// which is null unless the datasource sets networks = "jdcloud". That is a
// 2 → 1 merge combined with an input change, so it cannot be rewritten
// mechanically without guessing at intent.
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for the cloudflare_ip_ranges
// datasource v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with "data." prefix to distinguish from resource migration
	internal.RegisterMigrator("data.cloudflare_ip_ranges", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 datasource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_ip_ranges"
}

// GetAttributeRenames returns attribute renames for cross-file reference
// updates. Only the two attributes with an exact v5 equivalent are renamed;
// everything else is handled by GetInvalidAttributeReferences.
func (m *V4ToV5Migrator) GetAttributeRenames() []transform.AttributeRename {
	return []transform.AttributeRename{
		{
			ResourceType: "data.cloudflare_ip_ranges",
			OldAttribute: "ipv4_cidr_blocks",
			NewAttribute: "ipv4_cidrs",
		},
		{
			ResourceType: "data.cloudflare_ip_ranges",
			OldAttribute: "ipv6_cidr_blocks",
			NewAttribute: "ipv6_cidrs",
		},
	}
}

// GetInvalidAttributeReferences implements the InvalidAttributeReferenceDetector
// interface. It declares the v4 attributes that still have no valid v5
// equivalent after the renames above have been applied. The postprocessor emits
// a DiagWarning for each reference it finds.
func (m *V4ToV5Migrator) GetInvalidAttributeReferences() []transform.InvalidAttributeReference {
	combined := `'cidr_blocks' is not a valid attribute of the v5 cloudflare_ip_ranges datasource.

  In v4 this was the combined IPv4 and IPv6 list. v5 exposes the two families
  separately, so build the combined list explicitly:

    concat(
      data.cloudflare_ip_ranges.<name>.ipv4_cidrs,
      data.cloudflare_ip_ranges.<name>.ipv6_cidrs,
    )`

	china := `'%s' is not a valid attribute of the v5 cloudflare_ip_ranges datasource.

  v5 merged the two v4 china_* lists into a single 'jdcloud_cidrs' list that
  contains both IPv4 and IPv6 ranges. It is null unless the datasource opts in:

    data "cloudflare_ip_ranges" "<name>" {
      networks = "jdcloud"
    }

  Then reference data.cloudflare_ip_ranges.<name>.jdcloud_cidrs.

  Note this is a 2 → 1 merge: if you relied on the IPv4 and IPv6 China ranges
  being separate, filter jdcloud_cidrs yourself, for example with
  [for c in ... : c if !strcontains(c, ":")] for the IPv4 entries.`

	return []transform.InvalidAttributeReference{
		{
			ResourceType: "data.cloudflare_ip_ranges",
			Attribute:    "cidr_blocks",
			Suggestion:   combined,
		},
		{
			ResourceType: "data.cloudflare_ip_ranges",
			Attribute:    "china_ipv4_cidr_blocks",
			Suggestion:   fmt.Sprintf(china, "china_ipv4_cidr_blocks"),
		},
		{
			ResourceType: "data.cloudflare_ip_ranges",
			Attribute:    "china_ipv6_cidr_blocks",
			Suggestion:   fmt.Sprintf(china, "china_ipv6_cidr_blocks"),
		},
	}
}

// GetResourceRename implements the ResourceRenamer interface.
// The cloudflare_ip_ranges datasource keeps its name in v5.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"data.cloudflare_ip_ranges"}, "data.cloudflare_ip_ranges"
}

// CanHandle determines if this migrator can handle the given datasource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Only match datasource type (with "data." prefix)
	return resourceType == "data.cloudflare_ip_ranges"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for the ip_ranges datasource migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig handles configuration file transformations.
//
// The v4 datasource has no input arguments — every v4 attribute is computed —
// so the block body is returned unchanged. The entire migration happens through
// GetAttributeRenames and GetInvalidAttributeReferences, which operate on
// references to this datasource in other files.
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
