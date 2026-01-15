package workers_for_platforms_dispatch_namespace

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_workers_for_platforms_dispatch_namespace", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_workers_for_platforms_dispatch_namespace"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_for_platforms_dispatch_namespace"
}

// Preprocess performs any string-level transformations before HCL parsing.
// For workers_for_platforms_dispatch_namespace, no preprocessing is needed.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_workers_for_platforms_dispatch_namespace", "cloudflare_workers_for_platforms_dispatch_namespace"
}

// TransformConfig transforms the HCL configuration from v4 to v5.
// For workers_for_platforms_dispatch_namespace, the config is identical between v4 and v5.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No transformations needed - config is identical
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState transforms the JSON state from v4 to v5.
// The v5 provider requires namespace_name to be set in state for Read/Delete operations.
// In v5, both id and namespace_name contain the same value (the namespace identifier).
// This transformation copies the id value to namespace_name to match v5's expectations.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	// The v5 provider expects namespace_name to be set in state.
	// In v4, the id field contains the namespace identifier.
	// In v5, both id and namespace_name must contain the same value.
	// This matches the v5 provider's behavior in Create (line 94: data.ID = data.NamespaceName)
	// and ImportState (line 187: data.NamespaceName = types.StringValue(path_dispatch_namespace))
	if id := attrs.Get("id"); id.Exists() && id.String() != "" {
		var err error
		result, err = sjson.Set(result, "attributes.namespace_name", id.String())
		if err != nil {
			return result, err
		}
	}

	return result, nil
}
