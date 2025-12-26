package zero_trust_access_mtls_certificate

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Register both the deprecated v4 name and the intermediate v5 name
	internal.RegisterMigrator("cloudflare_access_mutual_tls_certificate", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_mtls_certificate", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the current name and the deprecated v4 name
	return resourceType == "cloudflare_access_mutual_tls_certificate" ||
		resourceType == "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations can be done with HCL helpers
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_mutual_tls_certificate", "cloudflare_zero_trust_access_mtls_certificate"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	if resourceType == "cloudflare_access_mutual_tls_certificate" {
		tfhcl.RenameResourceType(block, "cloudflare_access_mutual_tls_certificate", "cloudflare_zero_trust_access_mtls_certificate")
	}

	// No other config transformations needed - all fields remain the same!

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("resources").Exists() {
		return m.transformFullState(result, stateJSON)
	}

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	result = m.transformSingleInstance(result, stateJSON)

	return result, nil
}

func (m *V4ToV5Migrator) transformFullState(result string, stateJSON gjson.Result) (string, error) {
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		if !m.CanHandle(resourceType) {
			return true // continue
		}

		// Rename cloudflare_access_mutual_tls_certificate to cloudflare_zero_trust_access_mtls_certificate
		if resourceType == "cloudflare_access_mutual_tls_certificate" {
			resourcePath := "resources." + key.String() + ".type"
			result, _ = sjson.Set(result, resourcePath, "cloudflare_zero_trust_access_mtls_certificate")
		}

		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			attrs := instance.Get("attributes")
			if attrs.Exists() {
				instJSON := instance.String()
				transformedInst := m.transformSingleInstance(instJSON, instance)
				transformedInstParsed := gjson.Parse(transformedInst)
				result, _ = sjson.SetRaw(result, instPath, transformedInstParsed.Raw)
			}
			return true
		})

		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	attrs := instance.Get("attributes")

	// Add associated_hostnames = [] if missing (v5 has default of empty set)
	if !attrs.Get("associated_hostnames").Exists() {
		result, _ = sjson.Set(result, "attributes.associated_hostnames", []interface{}{})
	}

	// Always set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result
}
