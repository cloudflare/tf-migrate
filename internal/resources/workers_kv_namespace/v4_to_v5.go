package workers_kv_namespace

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_workers_kv_namespace", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_kv_namespace"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_kv_namespace"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For workers_kv_namespace, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_workers_kv_namespace", "cloudflare_workers_kv_namespace"
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For workers_kv_namespace, the config is identical between v4 and v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - config is identical
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the JSON state from v4 to v5.
// For workers_kv_namespace, the state is identical between v4 and v5.
// The id field exists in both v4 (implicit) and v5 (explicit in schema).
// The supports_url_encoding field is new in v5 and will be populated by the provider on first refresh.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	// No transformations needed - state structure is identical
	// The id field already exists in v4 state and will continue to exist in v5
	// The supports_url_encoding field (new in v5) will be added by the provider on first refresh

	return result, nil
}
