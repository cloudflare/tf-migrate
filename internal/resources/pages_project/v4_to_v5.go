package pages_project

import (
	"strings"

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

	// CRITICAL: If deployment_configs exists in state but not in config, add it with fail_open = false
	// This prevents v5 from trying to "fix" the state to match its new default of fail_open = true
	if ctx.StateJSON != "" && tfhcl.FindBlockByType(body, "deployment_configs") == nil {
		// Parse state to check if this resource has deployment_configs
		stateData := gjson.Parse(ctx.StateJSON)

		// Find any pages_project resource in the state with deployment_configs
		resources := stateData.Get("resources").Array()
		for _, resource := range resources {
			if resource.Get("type").String() == "cloudflare_pages_project" {
				instances := resource.Get("instances").Array()
				for _, instance := range instances {
					// Check if this instance has deployment_configs with fail_open
					// In v4 state, deployment_configs is an array: [{preview: [...], production: [...]}]
					deploymentConfigs := instance.Get("attributes.deployment_configs")
					if deploymentConfigs.Exists() && deploymentConfigs.IsArray() && len(deploymentConfigs.Array()) > 0 {
						// Get the first element of the deployment_configs array
						configObj := deploymentConfigs.Array()[0]

						// Check if preview or production has fail_open = false
						hasFailOpenFalse := false
						for _, env := range []string{"preview", "production"} {
							// Each environment is also an array in v4: preview: [{fail_open: false, ...}]
							envArray := configObj.Get(env)
							if envArray.Exists() && envArray.IsArray() && len(envArray.Array()) > 0 {
								envObj := envArray.Array()[0]
								if failOpen := envObj.Get("fail_open"); failOpen.Exists() && !failOpen.Bool() {
									hasFailOpenFalse = true
									break
								}
							}
						}

						if hasFailOpenFalse {
							// Add minimal deployment_configs block with fail_open = false
							// Also preserve other fields from v4 state to prevent drift
							deploymentConfigsBlock := body.AppendNewBlock("deployment_configs", nil)
							deploymentConfigsBody := deploymentConfigsBlock.Body()

							// Helper to add environment block with all fields from v4 state
							// IMPORTANT: We must copy ALL fields from v4 state to prevent v5 from detecting drift
							addEnvBlock := func(envName string, envData gjson.Result) {
								envBlock := deploymentConfigsBody.AppendNewBlock(envName, nil)
								envBody := envBlock.Body()

								// Always set fail_open to false (v4 default, v5 default is true)
								tfhcl.SetAttribute(envBody, "fail_open", false)

								// Copy all v4 fields that exist in state
								if usageModel := envData.Get("usage_model"); usageModel.Exists() && usageModel.String() != "" {
									tfhcl.SetAttribute(envBody, "usage_model", usageModel.String())
								}
								if compatDate := envData.Get("compatibility_date"); compatDate.Exists() && compatDate.String() != "" {
									tfhcl.SetAttribute(envBody, "compatibility_date", compatDate.String())
								}
								// Note: compatibility_flags and always_use_latest_compatibility_date are optional
								// and will be handled by the provider if not specified
							}

							// Add production and preview blocks
							for _, env := range []string{"preview", "production"} {
								envArray := configObj.Get(env)
								if envArray.Exists() && envArray.IsArray() && len(envArray.Array()) > 0 {
									addEnvBlock(env, envArray.Array()[0])
								}
							}
						}
					}
				}
			}
		}
	}

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
			// IMPORTANT: fail_open default changed from false (v4) to true (v5)
			// Always explicitly set to false to preserve v4 behavior and prevent drift
			tfhcl.SetAttribute(previewBody, "fail_open", false)
			// Convert placement block to attribute
			tfhcl.ConvertSingleBlockToAttribute(previewBody, "placement", "placement")
		}

		// Process production deployment config (deepest first)
		if productionBlock := tfhcl.FindBlockByType(deploymentConfigsBody, "production"); productionBlock != nil {
			productionBody := productionBlock.Body()
			// Add v4 defaults if missing (CRITICAL for behavior preservation)
			tfhcl.EnsureAttribute(productionBody, "usage_model", "bundled")
			// IMPORTANT: fail_open default changed from false (v4) to true (v5)
			// Always explicitly set to false to preserve v4 behavior and prevent drift
			tfhcl.SetAttribute(productionBody, "fail_open", false)
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

	// Step 2: Process build_config - populate all v5 fields
	if !attrs.Get("canonical_deployment").Exists() {
		result, _ = sjson.Set(result, "attributes.canonical_deployment", nil)
	}
	if !attrs.Get("framework").Exists() {
		result, _ = sjson.Set(result, "attributes.framework", "")
	}
	if !attrs.Get("framework_version").Exists() {
		result, _ = sjson.Set(result, "attributes.framework_version", "")
	}
	if !attrs.Get("latest_deployment").Exists() {
		result, _ = sjson.Set(result, "attributes.latest_deployment", nil)
	}
	if !attrs.Get("uses_functions").Exists() {
		result, _ = sjson.Set(result, "attributes.uses_functions", nil)
	}
	if attrs.Get("domains").Exists() {
		result, _ = sjson.Delete(result, "attributes.domains")
	}

	if buildConfigObj := attrs.Get("build_config"); buildConfigObj.Exists() && buildConfigObj.IsObject() {
		result = m.populateBuildConfigV5Fields(result, "attributes.build_config", buildConfigObj)
	} else {
		// No build_config - set it to an object with all null fields to match v5
		result, _ = sjson.Set(result, "attributes.build_config", map[string]interface{}{
			"build_caching":       nil,
			"build_command":       nil,
			"destination_dir":     nil,
			"root_dir":            nil,
			"web_analytics_tag":   nil,
			"web_analytics_token": nil,
		})
	}

	// Refresh attrs
	attrs = gjson.Parse(result).Get("attributes")

	// Step 3: Process nested conversions in source
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
	} else {
		// No source - set to null
		result, _ = sjson.Set(result, "attributes.source", nil)
	}

	// Step 4: Process nested conversions in deployment_configs
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

	// Note: build_config and deployment_configs are ComputedOptional in both v4 and v5.
	// This means when not specified in config, Terraform accepts whatever the API returns.
	// We should NOT remove these from state even if empty, as this would cause the v5 provider
	// to attempt to delete them from the API, which the API doesn't support.
	// Instead, we leave them in state as-is, matching the API's auto-generated values.

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
		if len(arr) == 0 && !strings.Contains(path, "compatibility_flags") {
			// Empty array - delete it (v4 stores as nil, v5 must too)
			// This includes compatibility_flags: [] which the v5 provider removes when not in config
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
	// Step 1: Merge environment_variables + secrets → env_vars
	envVars := deploymentConfig.Get("environment_variables")
	secrets := deploymentConfig.Get("secrets")

	hasEnvVars := false
	if envVars.Exists() && envVars.IsObject() {
		envVars.ForEach(func(key, value gjson.Result) bool {
			hasEnvVars = true
			return false // early exit
		})
	}
	if secrets.Exists() && secrets.IsObject() {
		secrets.ForEach(func(key, value gjson.Result) bool {
			hasEnvVars = true
			return false // early exit
		})
	}

	if hasEnvVars {
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

		result, _ = sjson.Set(result, basePath+".env_vars", mergedEnvVars)
	}

	// Step 2: Transform TypeMap fields (wrap string values in objects if not empty)
	result = m.transformMapFieldToNull(result, basePath+".kv_namespaces", deploymentConfig.Get("kv_namespaces"), "namespace_id")
	result = m.transformMapFieldToNull(result, basePath+".d1_databases", deploymentConfig.Get("d1_databases"), "id")
	result = m.transformMapFieldToNull(result, basePath+".durable_object_namespaces", deploymentConfig.Get("durable_object_namespaces"), "namespace_id")
	result = m.transformMapFieldToNull(result, basePath+".r2_buckets", deploymentConfig.Get("r2_buckets"), "name")

	// Step 3: Convert service_binding array to services map
	serviceBinding := deploymentConfig.Get("service_binding")
	hasServices := false
	if serviceBinding.Exists() && serviceBinding.IsArray() {
		servicesMap := make(map[string]interface{})

		serviceBinding.ForEach(func(k, v gjson.Result) bool {
			name := v.Get("name").String()
			if name != "" {
				hasServices = true
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

		if hasServices {
			result, _ = sjson.Set(result, basePath+".services", servicesMap)
		}
	}

	// Step 4: Remove old v4 fields that don't exist in v5
	result, _ = sjson.Delete(result, basePath+".environment_variables")
	result, _ = sjson.Delete(result, basePath+".secrets")
	result, _ = sjson.Delete(result, basePath+".service_binding")
	// Note: placement exists in both v4 and v5, so we don't delete it here
	// It was already converted from array to object format earlier

	// Step 5: Populate all v5 fields with their default values to match v5 provider output
	// This prevents drift when the v5 provider reads the state

	// NOTE: compatibility_flags is removed earlier by convertListToObject if it's an empty array
	// The v5 provider removes empty compatibility_flags from state when not in config
	freshDeploymentConfig := gjson.Parse(result).Get(basePath)

	// Set fields that exist in v4 and v5, keeping their values or setting defaults
	if !freshDeploymentConfig.Get("always_use_latest_compatibility_date").Exists() {
		result, _ = sjson.Set(result, basePath+".always_use_latest_compatibility_date", false)
	}
	if !freshDeploymentConfig.Get("compatibility_date").Exists() {
		result, _ = sjson.Set(result, basePath+".compatibility_date", nil)
	}
	if !freshDeploymentConfig.Get("usage_model").Exists() {
		result, _ = sjson.Set(result, basePath+".usage_model", "standard")
	}
	// IMPORTANT: fail_open default changed from false (v4) to true (v5)
	// If doesn't exist set to false to match v4 default behavior
	if !freshDeploymentConfig.Get("fail_open").Exists() {
		result, _ = sjson.Set(result, basePath+".fail_open", false)
	}

	// Set new v5 fields that don't exist in v4 to null or their defaults
	result, _ = sjson.Set(result, basePath+".ai_bindings", nil)
	result, _ = sjson.Set(result, basePath+".analytics_engine_datasets", nil)
	result, _ = sjson.Set(result, basePath+".browsers", nil)
	result, _ = sjson.Set(result, basePath+".build_image_major_version", 3)
	result, _ = sjson.Set(result, basePath+".hyperdrive_bindings", nil)
	result, _ = sjson.Set(result, basePath+".limits", nil)
	result, _ = sjson.Set(result, basePath+".mtls_certificates", nil)
	result, _ = sjson.Set(result, basePath+".queue_producers", nil)
	result, _ = sjson.Set(result, basePath+".vectorize_bindings", nil)
	result, _ = sjson.Set(result, basePath+".wrangler_config_hash", nil)

	// Set fields to null if they weren't populated in previous steps
	freshDeploymentConfig = gjson.Parse(result).Get(basePath)
	if !freshDeploymentConfig.Get("env_vars").Exists() {
		result, _ = sjson.Set(result, basePath+".env_vars", nil)
	}
	if !freshDeploymentConfig.Get("kv_namespaces").Exists() {
		result, _ = sjson.Set(result, basePath+".kv_namespaces", nil)
	}
	if !freshDeploymentConfig.Get("d1_databases").Exists() {
		result, _ = sjson.Set(result, basePath+".d1_databases", nil)
	}
	if !freshDeploymentConfig.Get("durable_object_namespaces").Exists() {
		result, _ = sjson.Set(result, basePath+".durable_object_namespaces", nil)
	}
	if !freshDeploymentConfig.Get("r2_buckets").Exists() {
		result, _ = sjson.Set(result, basePath+".r2_buckets", nil)
	}
	if !freshDeploymentConfig.Get("services").Exists() {
		result, _ = sjson.Set(result, basePath+".services", nil)
	}
	if !freshDeploymentConfig.Get("placement").Exists() {
		result, _ = sjson.Set(result, basePath+".placement", nil)
	}

	return result
}

// populateBuildConfigV5Fields ensures all v5 build_config fields are present
func (m *V4ToV5Migrator) populateBuildConfigV5Fields(result string, basePath string, buildConfig gjson.Result) string {
	// Set all v5 fields, keeping existing values or setting to null
	fields := []string{"build_caching", "build_command", "destination_dir", "root_dir", "web_analytics_tag", "web_analytics_token"}

	for _, field := range fields {
		if !buildConfig.Get(field).Exists() {
			result, _ = sjson.Set(result, basePath+"."+field, nil)
		}
	}

	return result
}

// transformMapFieldToNull transforms a map field from v4 to v5 format
// If the map is empty or doesn't exist, it will be set to null later
// If it has values, wrap them in objects with the specified field name
func (m *V4ToV5Migrator) transformMapFieldToNull(result string, path string, mapField gjson.Result, wrapFieldName string) string {
	if !mapField.Exists() {
		return result
	}

	if !mapField.IsObject() {
		// Delete non-object fields
		result, _ = sjson.Delete(result, path)
		return result
	}

	// Check if map has any values
	hasValues := false
	mapField.ForEach(func(key, value gjson.Result) bool {
		hasValues = true
		return false // early exit
	})

	if !hasValues {
		// Empty map - will be set to null later
		result, _ = sjson.Delete(result, path)
		return result
	}

	// Map has values - wrap them in objects
	newMap := make(map[string]interface{})
	mapField.ForEach(func(key, value gjson.Result) bool {
		newMap[key.String()] = map[string]interface{}{
			wrapFieldName: value.String(),
		}
		return true
	})

	result, _ = sjson.Set(result, path, newMap)
	return result
}
