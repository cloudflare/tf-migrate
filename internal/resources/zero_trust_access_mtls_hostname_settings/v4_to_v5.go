package zero_trust_access_mtls_hostname_settings

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register the resource name (same in v4 and v5)
	internal.RegisterMigrator("cloudflare_zero_trust_access_mtls_hostname_settings", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the resource name (same in v4 and v5)
	return "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the resource name
	return resourceType == "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - ConvertBlocksToArrayAttribute handles block to attribute conversion
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_zero_trust_access_mtls_hostname_settings doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_zero_trust_access_mtls_hostname_settings", "cloudflare_zero_trust_access_mtls_hostname_settings"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Convert settings blocks to array attribute
	// v4: settings { hostname = "..." } (ListNestedBlock - multiple blocks)
	// v5: settings = [{ hostname = "..." }] (ListNestedAttribute - array of objects)
	// emptyIfNone: false - settings is required and must have at least 1 item
	tfhcl.ConvertBlocksToArrayAttribute(body, "settings", false)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := stateJSON.String()

	// Get the attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		return result, nil
	}

	// Handle settings array - add defaults for Optional→Required fields
	// In v4: china_network and client_certificate_forwarding were Optional
	// In v5: both became Required
	if settings := attrs.Get("settings"); settings.Exists() && settings.IsArray() {
		settingsArray := settings.Array()
		for i, setting := range settingsArray {
			// Add china_network default if missing
			if !setting.Get("china_network").Exists() {
				result, _ = sjson.Set(result, fmt.Sprintf("attributes.settings.%d.china_network", i), false)
			}
			// Add client_certificate_forwarding default if missing
			if !setting.Get("client_certificate_forwarding").Exists() {
				result, _ = sjson.Set(result, fmt.Sprintf("attributes.settings.%d.client_certificate_forwarding", i), false)
			}
		}
	}

	// Set schema_version to 0 for v5
	result = state.SetSchemaVersion(result, 0)

	return result, nil
}
