package pages_project

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Pages Project resources from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with the OLD (v4) resource name
	internal.RegisterMigrator("cloudflare_pages_project", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name (unchanged in this case)
	return "cloudflare_pages_project"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Check for the OLD (v4) resource name
	return resourceType == "cloudflare_pages_project"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for pages_project
	// Complex transformations will be handled in TransformConfig and state transformation
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Resource name doesn't change (cloudflare_pages_project in both v4 and v5)
	body := block.Body()

	// Important: Process nested blocks BEFORE converting parent blocks
	// This ensures we can access and transform the nested structure

	// Step 1: Process source block and its nested config
	if sourceBlock := tfhcl.FindBlockByType(body, "source"); sourceBlock != nil {
		sourceBody := sourceBlock.Body()

		// Process config block within source first
		if configBlock := tfhcl.FindBlockByType(sourceBody, "config"); configBlock != nil {
			configBody := configBlock.Body()
			// Rename field before converting block to attribute
			tfhcl.RenameAttribute(configBody, "production_deployment_enabled", "production_deployments_enabled")
		}

		// Now convert config block to attribute
		tfhcl.ConvertSingleBlockToAttribute(sourceBody, "config", "config")
	}

	// Step 2: Process deployment_configs and its nested structure
	if deploymentConfigsBlock := tfhcl.FindBlockByType(body, "deployment_configs"); deploymentConfigsBlock != nil {
		deploymentConfigsBody := deploymentConfigsBlock.Body()

		// Process preview deployment config (deepest first)
		if previewBlock := tfhcl.FindBlockByType(deploymentConfigsBody, "preview"); previewBlock != nil {
			previewBody := previewBlock.Body()
			// Add v4 defaults if missing (CRITICAL for behavior preservation)
			tfhcl.EnsureAttribute(previewBody, "usage_model", "bundled")
			tfhcl.EnsureAttribute(previewBody, "fail_open", false)
			// Convert placement block to attribute
			tfhcl.ConvertSingleBlockToAttribute(previewBody, "placement", "placement")
		}

		// Process production deployment config (deepest first)
		if productionBlock := tfhcl.FindBlockByType(deploymentConfigsBody, "production"); productionBlock != nil {
			productionBody := productionBlock.Body()
			// Add v4 defaults if missing (CRITICAL for behavior preservation)
			tfhcl.EnsureAttribute(productionBody, "usage_model", "bundled")
			tfhcl.EnsureAttribute(productionBody, "fail_open", false)
			// Convert placement block to attribute
			tfhcl.ConvertSingleBlockToAttribute(productionBody, "placement", "placement")
		}

		// Now convert preview and production blocks to attributes
		tfhcl.ConvertSingleBlockToAttribute(deploymentConfigsBody, "preview", "preview")
		tfhcl.ConvertSingleBlockToAttribute(deploymentConfigsBody, "production", "production")
	}

	// Step 3: Convert top-level blocks to attributes (after processing nested structure)
	tfhcl.ConvertSingleBlockToAttribute(body, "build_config", "build_config")
	tfhcl.ConvertSingleBlockToAttribute(body, "source", "source")
	tfhcl.ConvertSingleBlockToAttribute(body, "deployment_configs", "deployment_configs")

	// Note: Complex transformations that can't be easily done in HCL are left as-is
	// and will be handled in state transformation:
	// 1. Merging environment_variables + secrets → env_vars (with type/value structure)
	// 2. Converting service_binding blocks → services map (extract name as key)
	// 3. Converting TypeMap → MapNestedAttribute (wrap string values in objects)
	//
	// The HCL will still have the v4 structure for these fields, but the state
	// transformation will produce the correct v5 state. The provider will reconcile
	// the differences on the next apply.

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string) (string, error) {
	result := instance.String()

	// Get attributes
	attrs := instance.Get("attributes")
	if !attrs.Exists() {
		return result, nil
	}

	// Step 1: Convert TypeList MaxItems:1 arrays to objects (deepest first)
	result = m.convertListToObject(result, "attributes.build_config", attrs.Get("build_config"))
	result = m.convertListToObject(result, "attributes.source", attrs.Get("source"))
	result = m.convertListToObject(result, "attributes.deployment_configs", attrs.Get("deployment_configs"))

	// Refresh attrs after conversions
	attrs = gjson.Parse(result).Get("attributes")

	// Step 2: Process nested conversions in source
	if sourceObj := attrs.Get("source"); sourceObj.Exists() && sourceObj.IsObject() {
		result = m.convertListToObject(result, "attributes.source.config", sourceObj.Get("config"))

		// Refresh and rename field in source.config
		sourceObj = gjson.Parse(result).Get("attributes.source")
		if configObj := sourceObj.Get("config"); configObj.Exists() && configObj.IsObject() {
			if oldField := configObj.Get("production_deployment_enabled"); oldField.Exists() {
				result, _ = sjson.Set(result, "attributes.source.config.production_deployments_enabled", oldField.Value())
				result, _ = sjson.Delete(result, "attributes.source.config.production_deployment_enabled")
			}
		}
	}

	// Step 3: Process nested conversions in deployment_configs
	attrs = gjson.Parse(result).Get("attributes")
	if deploymentConfigsObj := attrs.Get("deployment_configs"); deploymentConfigsObj.Exists() && deploymentConfigsObj.IsObject() {
		// Convert preview and production
		result = m.convertListToObject(result, "attributes.deployment_configs.preview", deploymentConfigsObj.Get("preview"))
		result = m.convertListToObject(result, "attributes.deployment_configs.production", deploymentConfigsObj.Get("production"))

		// Refresh and process preview deployment config
		deploymentConfigsObj = gjson.Parse(result).Get("attributes.deployment_configs")
		if previewObj := deploymentConfigsObj.Get("preview"); previewObj.Exists() && previewObj.IsObject() {
			result = m.convertListToObject(result, "attributes.deployment_configs.preview.placement", previewObj.Get("placement"))
			result = m.processDeploymentConfigState(result, "attributes.deployment_configs.preview", previewObj)
		}

		// Process production deployment config
		deploymentConfigsObj = gjson.Parse(result).Get("attributes.deployment_configs")
		if productionObj := deploymentConfigsObj.Get("production"); productionObj.Exists() && productionObj.IsObject() {
			result = m.convertListToObject(result, "attributes.deployment_configs.production.placement", productionObj.Get("placement"))
			result = m.processDeploymentConfigState(result, "attributes.deployment_configs.production", productionObj)
		}
	}

	// ALWAYS set schema_version to 0
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// convertListToObject converts TypeList MaxItems:1 arrays to objects
// Example: [{"field": "value"}] → {"field": "value"}
func (m *V4ToV5Migrator) convertListToObject(result string, path string, field gjson.Result) string {
	if !field.Exists() {
		return result
	}

	if field.IsArray() {
		arr := field.Array()
		if len(arr) == 0 {
			// Empty array - delete it (v4 stores as nil, v5 must too)
			result, _ = sjson.Delete(result, path)
		} else if len(arr) == 1 {
			// Single item array - convert to object
			result, _ = sjson.Set(result, path, arr[0].Value())
		}
	}
	// If already an object or not an array, leave as-is

	return result
}

// processDeploymentConfigState handles complex transformations in deployment config (preview/production)
func (m *V4ToV5Migrator) processDeploymentConfigState(result string, basePath string, deploymentConfig gjson.Result) string {
	// Step 0a: Remove empty compatibility_flags arrays FIRST
	// The v5 provider removes empty arrays, so we should too to avoid plan diffs
	// Get a fresh copy of the deployment config to ensure we're checking the current state
	freshDeploymentConfig := gjson.Parse(result).Get(basePath)
	if compatFlags := freshDeploymentConfig.Get("compatibility_flags"); compatFlags.Exists() && compatFlags.IsArray() {
		arr := compatFlags.Array()
		if len(arr) == 0 {
			result, _ = sjson.Delete(result, basePath+".compatibility_flags")
		}
	}

	// Step 0b: Handle v4 default values that changed or were removed in v5
	// CRITICAL: usage_model default changed from "bundled" (v4) to "standard" (v5)
	// If missing in v4, explicitly set to "bundled" to preserve v4 behavior
	if !deploymentConfig.Get("usage_model").Exists() {
		result, _ = sjson.Set(result, basePath+".usage_model", "bundled")
	}

	// IMPORTANT: fail_open had default false in v4, no default in v5
	// If missing in v4, explicitly set to false to preserve v4 behavior
	if !deploymentConfig.Get("fail_open").Exists() {
		result, _ = sjson.Set(result, basePath+".fail_open", false)
	}

	// Step 1: Merge environment_variables + secrets → env_vars
	envVars := deploymentConfig.Get("environment_variables")
	secrets := deploymentConfig.Get("secrets")

	if envVars.Exists() || secrets.Exists() {
		mergedEnvVars := make(map[string]interface{})

		// Add environment_variables as plain_text
		if envVars.Exists() && envVars.IsObject() {
			envVars.ForEach(func(key, value gjson.Result) bool {
				mergedEnvVars[key.String()] = map[string]interface{}{
					"type":  "plain_text",
					"value": value.String(),
				}
				return true
			})
		}

		// Add secrets as secret_text
		if secrets.Exists() && secrets.IsObject() {
			secrets.ForEach(func(key, value gjson.Result) bool {
				mergedEnvVars[key.String()] = map[string]interface{}{
					"type":  "secret_text",
					"value": value.String(),
				}
				return true
			})
		}

		// Set merged env_vars if not empty, otherwise delete
		if len(mergedEnvVars) > 0 {
			result, _ = sjson.Set(result, basePath+".env_vars", mergedEnvVars)
		}

		// Remove old fields
		result, _ = sjson.Delete(result, basePath+".environment_variables")
		result, _ = sjson.Delete(result, basePath+".secrets")
	}

	// Step 2: Wrap TypeMap string values in objects
	result = m.wrapMapValue(result, basePath+".kv_namespaces", deploymentConfig.Get("kv_namespaces"), "namespace_id")
	result = m.wrapMapValue(result, basePath+".d1_databases", deploymentConfig.Get("d1_databases"), "id")
	result = m.wrapMapValue(result, basePath+".durable_object_namespaces", deploymentConfig.Get("durable_object_namespaces"), "namespace_id")
	result = m.wrapMapValue(result, basePath+".r2_buckets", deploymentConfig.Get("r2_buckets"), "name")

	// Step 3: Convert service_binding array to services map
	serviceBinding := deploymentConfig.Get("service_binding")
	if serviceBinding.Exists() && serviceBinding.IsArray() {
		servicesMap := make(map[string]interface{})

		serviceBinding.ForEach(func(k, v gjson.Result) bool {
			name := v.Get("name").String()
			if name != "" {
				service := map[string]interface{}{
					"service":     v.Get("service").String(),
					"environment": v.Get("environment").String(),
				}
				// Add entrypoint if exists
				if entrypoint := v.Get("entrypoint"); entrypoint.Exists() && entrypoint.String() != "" {
					service["entrypoint"] = entrypoint.String()
				}
				servicesMap[name] = service
			}
			return true
		})

		if len(servicesMap) > 0 {
			result, _ = sjson.Set(result, basePath+".services", servicesMap)
		}

		// Remove old field
		result, _ = sjson.Delete(result, basePath+".service_binding")
	}

	return result
}

// wrapMapValue wraps string values in a TypeMap with an object containing the specified field
// Example: { "KEY": "value" } → { "KEY": { "namespace_id": "value" } }
func (m *V4ToV5Migrator) wrapMapValue(result string, path string, mapField gjson.Result, wrapFieldName string) string {
	if !mapField.Exists() || !mapField.IsObject() {
		return result
	}

	newMap := make(map[string]interface{})
	mapField.ForEach(func(key, value gjson.Result) bool {
		newMap[key.String()] = map[string]interface{}{
			wrapFieldName: value.String(),
		}
		return true
	})

	if len(newMap) > 0 {
		result, _ = sjson.Set(result, path, newMap)
	} else {
		// Empty map - delete it
		result, _ = sjson.Delete(result, path)
	}

	return result
}
