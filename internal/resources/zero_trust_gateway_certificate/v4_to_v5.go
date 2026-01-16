package zero_trust_gateway_certificate

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
	internal.RegisterMigrator("cloudflare_zero_trust_gateway_certificate", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_gateway_certificate"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zero_trust_gateway_certificate"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Remove v4-only fields that don't exist in v5 or changed behavior:
	tfhcl.RemoveAttributes(body, "custom", "gateway_managed", "id", "qs_pack_id")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("resources").Exists() {
		return m.transformFullState(ctx, result, stateJSON)
	}

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	result = m.transformSingleInstance(ctx, result, stateJSON, resourceName)

	return result, nil
}

func (m *V4ToV5Migrator) transformFullState(ctx *transform.Context, result string, stateJSON gjson.Result) (string, error) {
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		if !m.CanHandle(resourceType) {
			return true
		}

		resourceName := resource.Get("name").String()
		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			attrs := instance.Get("attributes")
			if attrs.Exists() {
				instJSON := instance.String()
				transformedInst := m.transformSingleInstance(ctx, instJSON, instance, resourceName)
				transformedInstParsed := gjson.Parse(transformedInst)
				result, _ = sjson.SetRaw(result, instPath, transformedInstParsed.Raw)
			}
			return true
		})

		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleInstance(ctx *transform.Context, result string, instance gjson.Result, resourceName string) string {
	attrs := instance.Get("attributes")

	// Remove v4-only fields that don't exist in v5 or changed behavior:
	result = state.RemoveFields(result, "attributes", attrs, "custom", "gateway_managed", "qs_pack_id")

	// Handle validity_period_days:
	// - If it was explicitly set in v4 config: keep it in state (convert to Int64)
	// - If it was just the v4 default: remove it from state
	validityPeriodDays := instance.Get("attributes.validity_period_days")
	if validityPeriodDays.Exists() {
		wasExplicitlySet := false
		if ctx.CFGFiles != nil {
			for _, cfgFile := range ctx.CFGFiles {
				body := cfgFile.Body()
				for _, block := range body.Blocks() {
					if block.Type() == "resource" && len(block.Labels()) >= 2 {
						if block.Labels()[0] == "cloudflare_zero_trust_gateway_certificate" && block.Labels()[1] == resourceName {
							if tfhcl.HasAttribute(block.Body(), "validity_period_days") {
								wasExplicitlySet = true
								break
							}
						}
					}
				}
				if wasExplicitlySet {
					break
				}
			}
		}

		if wasExplicitlySet {
			result, _ = sjson.Set(result, "attributes.validity_period_days", state.ConvertToFloat64(validityPeriodDays))
		} else {
			result = state.RemoveFields(result, "attributes", attrs, "validity_period_days")
		}
	}

	return result
}
