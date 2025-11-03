package workers_route

import (
	"regexp"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of Workers Route resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with both old (singular) and new (plural) resource names
	// This is necessary because preprocessing renames the resource type,
	// and the framework looks up migrators by the post-preprocessing name
	internal.RegisterMigrator("cloudflare_worker_route", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_workers_route", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the new (plural) resource type name
	return "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both old (singular) and new (plural) names
	return resourceType == "cloudflare_worker_route" || resourceType == "cloudflare_workers_route"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Rename cloudflare_worker_route to cloudflare_workers_route
	// Match the resource declaration and rename it
	pattern := regexp.MustCompile(`(resource\s+)"cloudflare_worker_route"`)
	content = pattern.ReplaceAllString(content, `${1}"cloudflare_workers_route"`)
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename script_name attribute to script
	tfhcl.RenameAttribute(block.Body(), "script_name", "script")

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
		// Full resource - transform all instances and update resource type
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
	if resourceType != "cloudflare_worker_route" && resourceType != "cloudflare_workers_route" {
		return result, nil
	}

	// Update resource type from singular to plural
	if resourceType == "cloudflare_worker_route" {
		result, _ = sjson.Set(result, "type", "cloudflare_workers_route")
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

// transformSingleInstance transforms a single instance's attributes
func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	if !instance.Get("attributes").Exists() {
		return result
	}

	// Rename script_name to script
	attrs := instance.Get("attributes")
	result = state.RenameField(result, "attributes", attrs, "script_name", "script")

	return result
}
