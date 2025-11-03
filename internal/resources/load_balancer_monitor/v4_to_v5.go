package load_balancer_monitor

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of Load Balancer Monitor resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_load_balancer_monitor", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_load_balancer_monitor"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// No config transformation needed
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// This function can receive either:
	// 1. A full resource (in unit tests) - has "type", "name", "instances"
	// 2. A single instance (in actual migration framework) - has "attributes"
	// We need to handle both cases

	result := stateJSON.String()

	// Check if this is a full resource (has "type" and "instances") or a single instance
	if stateJSON.Get("type").Exists() && stateJSON.Get("instances").Exists() {
		// Full resource - transform all instances
		return m.transformFullResource(result, stateJSON)
	}

	// Single instance - transform just the attributes
	if !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	return m.transformSingleInstance(result, stateJSON), nil
}

// transformFullResource handles transformation of a full resource with instances
func (m *V4ToV5Migrator) transformFullResource(result string, resource gjson.Result) (string, error) {
	resourceType := resource.Get("type").String()
	if resourceType != "cloudflare_load_balancer_monitor" {
		return result, nil
	}

	// Transform all instances
	instances := resource.Get("instances")
	instances.ForEach(func(key, instance gjson.Result) bool {
		instPath := "instances." + key.String()
		instJSON := instance.String()
		transformedInst := m.transformSingleInstance(instJSON, instance)
		result, _ = sjson.SetRaw(result, instPath, transformedInst)
		return true
	})

	return result, nil
}

// transformSingleInstance transforms a single instance's header attribute
func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	header := instance.Get("attributes.header")
	if !header.Exists() || !header.IsArray() {
		return result
	}

	// Transform header from array format to map format
	// v4: header = [{"header": "Host", "values": ["example.com"]}]
	// v5: header = {"Host": ["example.com"]}
	headerMap := make(map[string][]string)
	for _, item := range header.Array() {
		if !item.IsObject() {
			continue
		}

		headerName := item.Get("header").String()
		values := item.Get("values")
		if headerName == "" || !values.Exists() {
			continue
		}

		// Convert values to string array
		var stringValues []string
		if values.IsArray() {
			for _, v := range values.Array() {
				stringValues = append(stringValues, v.String())
			}
		}
		headerMap[headerName] = stringValues
	}

	// Set the transformed header or remove if empty
	if len(headerMap) > 0 {
		result, _ = sjson.Set(result, "attributes.header", headerMap)
	} else {
		// Empty array -> remove the attribute
		result, _ = sjson.Delete(result, "attributes.header")
	}

	return result
}
