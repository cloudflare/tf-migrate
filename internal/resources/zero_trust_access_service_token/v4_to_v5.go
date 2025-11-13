package zero_trust_access_service_token

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

	// Deprecated v4 Name
	internal.RegisterMigrator("cloudflare_access_service_token", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_service_token", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the current name and the deprecated v4 name
	return resourceType == "cloudflare_access_service_token" || resourceType == "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_access_service_token", "cloudflare_zero_trust_access_service_token"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)
	if resourceType == "cloudflare_access_service_token" {
		tfhcl.RenameResourceType(block, "cloudflare_access_service_token", "cloudflare_zero_trust_access_service_token")
	}

	body := block.Body()

	// Remove deprecated field: min_days_for_renewal
	tfhcl.RemoveAttributes(body, "min_days_for_renewal")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
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

		// Rename cloudflare_access_service_token to cloudflare_zero_trust_access_service_token
		if resourceType == "cloudflare_access_service_token" {
			resourcePath := "resources." + key.String() + ".type"
			result, _ = sjson.Set(result, resourcePath, "cloudflare_zero_trust_access_service_token")
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

	// Remove deprecated field: min_days_for_renewal
	result = state.RemoveFields(result, "attributes", attrs, "min_days_for_renewal")

	// Convert client_secret_version from int to float64
	clientSecretVersion := instance.Get("attributes.client_secret_version")
	if clientSecretVersion.Exists() && clientSecretVersion.Type == gjson.Number {
		result, _ = sjson.Set(result, "attributes.client_secret_version", clientSecretVersion.Float())
	} else {
		// Set default 1.0
		result, _ = sjson.Set(result, "attributes.client_secret_version", 1.0)
	}

	return result
}
