package hyperdrive_config

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// V4ToV5Migrator handles migration of Hyperdrive Config resources from v4 to v5.
// The resource name and all user-configurable attributes are identical between
// v4 and v5. V5 adds new optional attributes (origin.service_id,
// origin_connection_limit, mtls) and new computed attributes (created_on,
// modified_on), but no existing attributes were renamed, removed, or
// restructured. The provider's StateUpgrader handles state migration
// automatically, so TransformState is not needed.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_hyperdrive_config", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_hyperdrive_config"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_hyperdrive_config"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// The resource name is unchanged between v4 and v5.
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_hyperdrive_config"}, "cloudflare_hyperdrive_config"
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// No transformations are needed -- the config is identical between v4 and v5.
// All v4 attributes (origin.database, origin.host, origin.port, origin.scheme,
// origin.user, origin.password, origin.access_client_id,
// origin.access_client_secret, caching.disabled, caching.max_age,
// caching.stale_while_revalidate) are valid in v5 with the same names and types.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
