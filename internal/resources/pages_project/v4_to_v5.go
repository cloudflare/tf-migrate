package pages_project

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

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

	// Note: usage_model is deprecated and the API now always returns "standard".
	// We no longer need to extract or set usage_model values during migration.
	// The provider will handle this field automatically.

	// Handle build_config - remove if only contains empty/default values
	// Note: build_config is Computed+Optional in v5, so the provider can populate it from the API.
	// If the user only specified empty/default values, we should remove the block entirely
	// to avoid plan diffs with unspecified fields like web_analytics_tag/web_analytics_token.
	buildConfigBlock := tfhcl.FindBlockByType(body, "build_config")
	if buildConfigBlock != nil {
		buildConfigBody := buildConfigBlock.Body()
		// Check if build_config only has empty/falsey values
		hasNonEmptyValues := false
		for _, attr := range buildConfigBody.Attributes() {
			tokens := attr.Expr().BuildTokens(nil)
			value := string(tokens.Bytes())
			// Check if value is non-empty (not "", not "false", not empty quotes)
			trimmed := strings.TrimSpace(value)
			if trimmed != `""` && trimmed != "false" && trimmed != "" {
				hasNonEmptyValues = true
				break
			}
		}
		if !hasNonEmptyValues {
			// Remove the entire build_config block - let provider compute from API
			body.RemoveBlock(buildConfigBlock)
		}
	}

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
			// Note: usage_model is deprecated - don't add it to config, let provider handle it
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
			// Note: usage_model is deprecated - don't add it to config, let provider handle it
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

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState is a no-op - state transformation is handled by the provider's state upgrader.
// The v5 provider implements UpgradeState which handles all state transformations during
// terraform plan/apply when upgrading from v4 to v5.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath string, resourceName string) (string, error) {
	return instance.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
