package d1_database

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// V4ToV5Migrator handles migration of D1 Database resources from v4 to v5.
// The resource name and all user-configurable attributes are identical between
// v4 and v5. V5 adds new optional attributes (jurisdiction, primary_location_hint,
// read_replication) and new computed attributes (uuid, created_at, file_size,
// num_tables), but no existing attributes were renamed, removed, or restructured.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_d1_database", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_d1_database"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_d1_database"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// The resource name is unchanged between v4 and v5.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_d1_database"}, "cloudflare_d1_database"
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// The only required change is adding `read_replication = { mode = "disabled" }`
// to match the API default. The D1 API returns read_replication={mode:"disabled"}
// on read but rejects null on update. Without this, the v5 provider's state
// upgrader initializes read_replication as null, which causes a 400 Bad Request
// on the next apply.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Set read_replication default if not already present.
	if !tfhcl.HasAttribute(body, "read_replication") {
		body.SetAttributeRaw("read_replication", hclwrite.TokensForObject([]hclwrite.ObjectAttrTokens{
			{
				Name:  hclwrite.TokensForIdentifier("mode"),
				Value: hclwrite.TokensForValue(cty.StringVal("disabled")),
			},
		}))
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
