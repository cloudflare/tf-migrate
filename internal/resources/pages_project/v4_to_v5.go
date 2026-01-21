package pages_project

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

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

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_pages_project doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_pages_project", "cloudflare_pages_project"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Resource name doesn't change (cloudflare_pages_project in both v4 and v5)
	body := block.Body()

	// Parse state to extract usage_model values
	var previewUsageModel, productionUsageModel string
	if ctx.StateJSON != "" {
		labels := block.Labels()
		if len(labels) >= 2 {
			resourceName := labels[1]
			gjson.Parse(ctx.StateJSON).Get("resources").ForEach(func(key, resource gjson.Result) bool {
				if m.CanHandle(resource.Get("type").String()) && resource.Get("name").String() == resourceName {
					// Extract usage_model from state if it exists
					previewUsageModel = resource.Get("instances.0.attributes.deployment_configs.0.preview.0.usage_model").String()
					productionUsageModel = resource.Get("instances.0.attributes.deployment_configs.0.production.0.usage_model").String()
					return false // Stop iterating once we find our resource
				}
				return true
			})
		}
	}
	// Default to "bundled" if not found in state
	if previewUsageModel == "" {
		previewUsageModel = "bundled"
	}
	if productionUsageModel == "" {
		productionUsageModel = "bundled"
	}

	// Handle build_config - preserve existing fields and ensure web_analytics fields exist
	// Note: build_config is Computed+Optional in v5, so the provider can populate it from the API
	// even when not present in config. We only need to migrate it if it exists in v4 config.
	buildConfigBlock := tfhcl.FindBlockByType(body, "build_config")
	if buildConfigBlock != nil {
		// build_config exists as a block - ensure web_analytics fields are present
		buildConfigBody := buildConfigBlock.Body()
		if buildConfigBody.GetAttribute("web_analytics_tag") == nil {
			tfhcl.SetAttribute(buildConfigBody, "web_analytics_tag", nil)
		}
		if buildConfigBody.GetAttribute("web_analytics_token") == nil {
			tfhcl.SetAttribute(buildConfigBody, "web_analytics_token", nil)
		}
	}
	// If build_config exists as an attribute or is missing entirely, leave it as-is

	// Handle deployment_configs
	deploymentConfigsBlock := tfhcl.FindBlockByType(body, "deployment_configs")
	deploymentConfigsAttr := body.GetAttribute("deployment_configs")

	if deploymentConfigsBlock == nil && deploymentConfigsAttr == nil {
		// Add deployment_configs block if missing entirely (neither block nor attribute)
		deploymentConfigsBlock = body.AppendNewBlock("deployment_configs", nil)
		deploymentConfigsBody := deploymentConfigsBlock.Body()

		// Add preview and production blocks (without compatibility_flags - let provider handle default)
		deploymentConfigsBody.AppendNewBlock("preview", nil)
		deploymentConfigsBody.AppendNewBlock("production", nil)
	}
	// If deployment_configs exists as an attribute, leave it as-is
	// Don't add compatibility_flags - let provider handle defaults

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
		previewBlock := tfhcl.FindBlockByType(deploymentConfigsBody, "preview")
		if previewBlock != nil {
			previewBody := previewBlock.Body()
			// Don't add compatibility_flags if missing - let provider handle default (null)
			// Only preserve it if already present in config
			if previewBody.GetAttribute("usage_model") == nil {
				previewBody.SetAttributeRaw("usage_model", hclwrite.TokensForValue(cty.StringVal(previewUsageModel)))
			}
			// Ensure fail_open is set to false (preserve if already present)
			if previewBody.GetAttribute("fail_open") == nil {
				previewBody.SetAttributeRaw("fail_open", hclwrite.TokensForValue(cty.False))
			}
			// Convert placement block to attribute
			tfhcl.ConvertSingleBlockToAttribute(previewBody, "placement", "placement")

			// Transform bindings from v4 string format to v5 object format
			// v4: kv_namespaces = { MY_KV = "id" }
			// v5: kv_namespaces = { MY_KV = { namespace_id = "id" } }
			tfhcl.WrapMapValuesInObjects(previewBody, "kv_namespaces", "namespace_id")
			tfhcl.WrapMapValuesInObjects(previewBody, "d1_databases", "id")
			tfhcl.WrapMapValuesInObjects(previewBody, "r2_buckets", "name")
			tfhcl.WrapMapValuesInObjects(previewBody, "durable_object_namespaces", "namespace_id")

			// Transform service_binding blocks to services map
			// v4: service_binding { name = "MY_SERVICE" service = "worker-1" }
			// v5: services = { MY_SERVICE = { service = "worker-1" } }
			tfhcl.ConvertServiceBindingBlocksToServicesMap(previewBody)
		}

		// Process production deployment config (deepest first)
		productionBlock := tfhcl.FindBlockByType(deploymentConfigsBody, "production")
		if productionBlock != nil {
			productionBody := productionBlock.Body()
			// Don't add compatibility_flags if missing - let provider handle default (null)
			// Only preserve it if already present in config
			// Ensure fail_open is set to false (preserve if already present)
			if productionBody.GetAttribute("fail_open") == nil {
				productionBody.SetAttributeRaw("fail_open", hclwrite.TokensForValue(cty.False))
			}
			if productionBody.GetAttribute("usage_model") == nil {
				productionBody.SetAttributeRaw("usage_model", hclwrite.TokensForValue(cty.StringVal(productionUsageModel)))
			}
			// Convert placement block to attribute
			tfhcl.ConvertSingleBlockToAttribute(productionBody, "placement", "placement")

			// Transform bindings from v4 string format to v5 object format
			tfhcl.WrapMapValuesInObjects(productionBody, "kv_namespaces", "namespace_id")
			tfhcl.WrapMapValuesInObjects(productionBody, "d1_databases", "id")
			tfhcl.WrapMapValuesInObjects(productionBody, "r2_buckets", "name")
			tfhcl.WrapMapValuesInObjects(productionBody, "durable_object_namespaces", "namespace_id")

			// Transform service_binding blocks to services map
			// v4: service_binding { name = "MY_SERVICE" service = "worker-1" }
			// v5: services = { MY_SERVICE = { service = "worker-1" } }
			tfhcl.ConvertServiceBindingBlocksToServicesMap(productionBody)
		}

		// Convert preview and production blocks to attributes
		tfhcl.ConvertSingleBlockToAttribute(deploymentConfigsBody, "preview", "preview")
		tfhcl.ConvertSingleBlockToAttribute(deploymentConfigsBody, "production", "production")
	}

	// Step 3: Convert top-level blocks to attributes (after processing nested structure)
	tfhcl.ConvertSingleBlockToAttribute(body, "build_config", "build_config")
	tfhcl.ConvertSingleBlockToAttribute(body, "source", "source")
	tfhcl.ConvertSingleBlockToAttribute(body, "deployment_configs", "deployment_configs")

	// Note: Some complex transformations that require API data are left as-is
	// and will be handled in state transformation:
	// 1. Merging environment_variables + secrets → env_vars (with type/value structure)
	//    - This requires API data to determine which vars are secrets vs plain text
	//
	// All other transformations are now handled in the config transformation above:
	// - service_binding blocks → services map (extract name as key)
	// - TypeMap bindings → MapNestedAttribute (kv_namespaces, d1_databases, r2_buckets, durable_object_namespaces)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string, resourceName string) (string, error) {
	result := instance.String()

	// Get attributes
	attrs := instance.Get("attributes")
	if !attrs.Exists() {
		return result, nil
	}

	// Parse config to determine which build_config fields were explicitly set
	buildConfigFieldsInConfig := make(map[string]bool)
	if ctx.CFGFile != nil {
		// Find the resource block in the HCL config
		for _, block := range ctx.CFGFile.Body().Blocks() {
			if block.Type() == "resource" {
				labels := block.Labels()
				if len(labels) >= 2 && labels[0] == "cloudflare_pages_project" && labels[1] == resourceName {
					// Found our resource - check for build_config
					buildConfigBlock := tfhcl.FindBlockByType(block.Body(), "build_config")
					buildConfigAttr := block.Body().GetAttribute("build_config")

					if buildConfigBlock != nil {
						// build_config is a block - get all attributes
						for name := range buildConfigBlock.Body().Attributes() {
							buildConfigFieldsInConfig[name] = true
						}
					} else if buildConfigAttr != nil {
						// build_config is an attribute - parse the value by checking if field names appear in the string
						attrStr := string(buildConfigAttr.Expr().BuildTokens(nil).Bytes())
						// Simple string-based check for field names in HCL
						// This works because HCL syntax is "field = value"
						fieldsToCheck := []string{"build_caching", "build_command", "destination_dir", "root_dir", "web_analytics_tag", "web_analytics_token"}
						for _, field := range fieldsToCheck {
							// Check if the field name appears in the attribute string
							// Use a simple string search - if it's there, it was in the config
							if len(attrStr) > 0 {
								// Check for "field =" or "field=" patterns
								fieldPattern := field + " ="
								fieldPatternNoSpace := field + "="
								if contains(attrStr, fieldPattern) || contains(attrStr, fieldPatternNoSpace) {
									buildConfigFieldsInConfig[field] = true
								}
							}
						}
					}
					break
				}
			}
		}
	}


	// Step 1: Convert TypeList MaxItems:1 arrays to objects (deepest first)
	// Special handling for build_config: keep empty array as empty object for later processing
	buildConfigInState := attrs.Get("build_config")
	if buildConfigInState.Exists() && buildConfigInState.IsArray() {
		arr := buildConfigInState.Array()
		if len(arr) == 0 {
			// Empty array - convert to empty object (will be kept/removed later based on config)
			result, _ = sjson.Set(result, "attributes.build_config", map[string]interface{}{})
		} else if len(arr) == 1 {
			// Single item array - convert to object
			result, _ = sjson.Set(result, "attributes.build_config", arr[0].Value())
		}
	}
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

	// Check if build_config exists in the state (as array or object)
	buildConfigObj := attrs.Get("build_config")

	// Check if build_config has actual data in the state
	hasBuildConfigData := false
	if buildConfigObj.Exists() && buildConfigObj.IsObject() {
		buildConfigObj.ForEach(func(key, value gjson.Result) bool {
			if value.Exists() && value.Type != gjson.Null {
				// Check for truthy values: not false, not empty string
				if value.Type == gjson.False {
					return true // continue - false is falsy
				}
				if value.Type == gjson.String && value.String() == "" {
					return true // continue - empty string is falsy
				}
				// Has a truthy value
				hasBuildConfigData = true
				return false // early exit
			}
			return true
		})
	}

	// Decide what to do with build_config based on config and state
	// Since build_config is Computed+Optional in v5, we always add it to state (even if empty)
	// to prevent drift when the provider populates it from the API
	if buildConfigObj.Exists() && buildConfigObj.IsObject() && hasBuildConfigData {
		// build_config has actual data - preserve and populate v5 fields
		result = m.populateBuildConfigV5Fields(result, "attributes.build_config", buildConfigObj, buildConfigFieldsInConfig)
	} else {
		// build_config is empty or missing - always set to empty object
		// This matches what the v5 provider will do and prevents drift
		result, _ = sjson.Set(result, "attributes.build_config", map[string]interface{}{})
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
		if len(arr) == 0 {
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
					"service": v.Get("service").String(),
				}
				// Add environment if exists and not empty
				if environment := v.Get("environment"); environment.Exists() && environment.String() != "" {
					service["environment"] = environment.String()
				}
				// Add entrypoint if exists and not empty
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

	freshDeploymentConfig := gjson.Parse(result).Get(basePath)

	// Set fields that exist in v4 and v5, keeping their values or setting defaults
	if !freshDeploymentConfig.Get("always_use_latest_compatibility_date").Exists() {
		result, _ = sjson.Set(result, basePath+".always_use_latest_compatibility_date", false)
	}
	if !freshDeploymentConfig.Get("compatibility_date").Exists() {
		result, _ = sjson.Set(result, basePath+".compatibility_date", nil)
	}

	// Leave compatibility_flags as null if not present (matches provider behavior)
	// The provider returns null for this field when not explicitly set
	// Also convert empty arrays to null
	compatFlags := freshDeploymentConfig.Get("compatibility_flags")
	if !compatFlags.Exists() || compatFlags.Type == gjson.Null {
		result, _ = sjson.Set(result, basePath+".compatibility_flags", nil)
	} else if compatFlags.IsArray() && len(compatFlags.Array()) == 0 {
		// Empty array - convert to null to match provider behavior
		result, _ = sjson.Set(result, basePath+".compatibility_flags", nil)
	}

	deploymentConf := freshDeploymentConfig.Get("usage_model")
	if !deploymentConf.Exists() || deploymentConf.Type == gjson.Null {
		result, _ = sjson.Set(result, basePath+".usage_model", "bundled")
	}
	// IMPORTANT: fail_open default was false in v4, preserve it to avoid breaking changes
	// Even though v5 default may differ, we preserve v4 behavior for existing resources
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
	} else {
		// Check if placement exists but all its fields are empty/null
		placementObj := freshDeploymentConfig.Get("placement")
		if placementObj.IsObject() {
			hasValues := false
			placementObj.ForEach(func(key, value gjson.Result) bool {
				if value.Exists() && value.Type != gjson.Null && value.String() != "" {
					hasValues = true
					return false // early exit
				}
				return true
			})

			if !hasValues {
				// All placement fields are empty/null - set placement to null
				result, _ = sjson.Set(result, basePath+".placement", nil)
			} else {
				// Placement has values - clean up empty mode field to be null
				if mode := placementObj.Get("mode"); mode.Exists() && mode.String() == "" {
					result, _ = sjson.Set(result, basePath+".placement.mode", nil)
				}
			}
		}
	}

	return result
}

// populateBuildConfigV5Fields ensures all v5 build_config fields are present
// Only include fields that were explicitly set in the config OR have truthy values
func (m *V4ToV5Migrator) populateBuildConfigV5Fields(result string, basePath string, buildConfig gjson.Result, fieldsInConfig map[string]bool) string {
	// Rebuild build_config with only fields that:
	// 1. Were explicitly set in the config, OR
	// 2. Have truthy values (not null, not false, not empty string)
	newBuildConfig := make(map[string]interface{})
	fieldsToCheck := []string{"build_caching", "build_command", "destination_dir", "root_dir", "web_analytics_tag", "web_analytics_token"}

	for _, field := range fieldsToCheck {
		fieldValue := buildConfig.Get(field)
		if !fieldValue.Exists() {
			// Field doesn't exist - don't add it
			continue
		}

		// Include field if:
		// - It was explicitly in the config, OR
		// - It has a truthy value (not null, not false, not empty string)
		inConfig := fieldsInConfig[field] || fieldsInConfig["*"] // "*" means build_config exists as attribute
		hasTruthyValue := fieldValue.Type != gjson.Null &&
			fieldValue.Type != gjson.False &&
			!(fieldValue.Type == gjson.String && fieldValue.String() == "")

		if inConfig || hasTruthyValue {
			newBuildConfig[field] = fieldValue.Value()
		}
	}

	// Replace build_config with the new object
	result, _ = sjson.Set(result, basePath, newBuildConfig)
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

// addCompatibilityFlagsToDeploymentConfigsAttr adds compatibility_flags to preview/production
// in a deployment_configs attribute value, preserving all existing fields
func (m *V4ToV5Migrator) addCompatibilityFlagsToDeploymentConfigsAttr(attr *hclwrite.Attribute) hclwrite.Tokens {
	// Check if compatibility_flags already exists - if so, don't modify
	if tfhcl.AttributeValueContainsKey(attr, "compatibility_flags") {
		return nil // Return nil to signal no changes needed
	}

	tokens := attr.Expr().BuildTokens(nil)

	var result hclwrite.Tokens
	inPreviewOrProduction := false
	depth := 0
	addedCompatFlags := false
	hasCompatFlagsInCurrentObject := false

	compatFlagsTokens := hclwrite.TokensForValue(cty.ListValEmpty(cty.String))

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		result = append(result, token)

		// Track depth to know when we're at the right level
		if token.Type == hclsyntax.TokenOBrace {
			depth++
			// If we just entered preview or production object, add compatibility_flags
			if depth == 2 && inPreviewOrProduction && !addedCompatFlags && !hasCompatFlagsInCurrentObject {
				// Add newline and compatibility_flags right after opening brace
				result = append(result, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
				result = append(result, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("      compatibility_flags")})
				result = append(result, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")})
				result = append(result, compatFlagsTokens...)
				addedCompatFlags = true
			}
			continue
		}

		if token.Type == hclsyntax.TokenCBrace {
			depth--
			if depth == 1 {
				inPreviewOrProduction = false
				addedCompatFlags = false
				hasCompatFlagsInCurrentObject = false
			}
			continue
		}

		// Check if we're inside preview/production and see compatibility_flags
		if depth == 2 && inPreviewOrProduction && token.Type == hclsyntax.TokenIdent {
			if string(token.Bytes) == "compatibility_flags" {
				hasCompatFlagsInCurrentObject = true
			}
		}

		// Check if we're at the top level (depth 1) and found "preview" or "production" key
		if depth == 1 && token.Type == hclsyntax.TokenIdent {
			keyName := string(token.Bytes)
			if keyName == "preview" || keyName == "production" {
				// Look ahead to see if next non-whitespace token is '='
				for j := i + 1; j < len(tokens); j++ {
					nextToken := tokens[j]
					if nextToken.Type == hclsyntax.TokenNewline || nextToken.Type == hclsyntax.TokenComment {
						continue
					}
					if nextToken.Type == hclsyntax.TokenEqual {
						inPreviewOrProduction = true
					}
					break
				}
			}
		}
	}

	return result
}
