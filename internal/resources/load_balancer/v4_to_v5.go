package load_balancer

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_load_balancer", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_load_balancer"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - config transformations handled by provider's migrate tool
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Config transformations are complex and handled by the provider's migrate tool
	// No transformations needed here
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	// Rename fallback_pool_id to fallback_pool
	fallbackPoolIDPath := "attributes.fallback_pool_id"
	fallbackPoolPath := "attributes.fallback_pool"
	fallbackPoolID := stateJSON.Get(fallbackPoolIDPath)
	if fallbackPoolID.Exists() {
		result, _ = sjson.Set(result, fallbackPoolPath, fallbackPoolID.Value())
		result, _ = sjson.Delete(result, fallbackPoolIDPath)
	}

	// Rename default_pool_ids to default_pools
	defaultPoolIDsPath := "attributes.default_pool_ids"
	defaultPoolsPath := "attributes.default_pools"
	defaultPoolIDs := stateJSON.Get(defaultPoolIDsPath)
	if defaultPoolIDs.Exists() {
		result, _ = sjson.Set(result, defaultPoolsPath, defaultPoolIDs.Value())
		result, _ = sjson.Delete(result, defaultPoolIDsPath)
	}

	// Remove empty arrays for single-object attributes
	singleObjectAttrs := []string{
		"adaptive_routing",
		"location_strategy",
		"random_steering",
		"session_affinity_attributes",
	}
	for _, attr := range singleObjectAttrs {
		attrPath := "attributes." + attr
		attrValue := gjson.Get(result, attrPath)
		if attrValue.Exists() && attrValue.IsArray() {
			if len(attrValue.Array()) == 0 {
				result, _ = sjson.Delete(result, attrPath)
			}
		}
	}

	// Convert empty arrays to empty maps for map attributes
	mapAttrs := []string{
		"country_pools",
		"pop_pools",
		"region_pools",
	}
	for _, attr := range mapAttrs {
		attrPath := "attributes." + attr
		attrValue := gjson.Get(result, attrPath)
		if attrValue.Exists() && attrValue.IsArray() {
			if len(attrValue.Array()) == 0 {
				result, _ = sjson.Set(result, attrPath, map[string]interface{}{})
			}
		}
	}

	return result, nil
}
