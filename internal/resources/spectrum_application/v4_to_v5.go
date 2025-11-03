package spectrum_application

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_spectrum_application", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_spectrum_application"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_spectrum_application"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Remove id attribute (V4 allowed optional, V5 is computed-only)
	body.RemoveAttribute("id")

	// 2. Handle origin_port_range block to origin_port string conversion
	// V4: origin_port_range { start = 80, end = 85 }
	// V5: origin_port = "80-85"
	for _, childBlock := range body.Blocks() {
		if childBlock.Type() == "origin_port_range" {
			blockBody := childBlock.Body()

			startAttr := blockBody.GetAttribute("start")
			endAttr := blockBody.GetAttribute("end")

			if startAttr != nil && endAttr != nil {
				startTokens := startAttr.Expr().BuildTokens(nil)
				endTokens := endAttr.Expr().BuildTokens(nil)

				if len(startTokens) >= 1 && len(endTokens) >= 1 {
					startValue := string(startTokens[0].Bytes)
					endValue := string(endTokens[0].Bytes)

					rangeValue := fmt.Sprintf(`"%s-%s"`, startValue, endValue)

					body.SetAttributeRaw("origin_port", []*hclwrite.Token{
						{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(rangeValue)},
					})
				}
			}

			body.RemoveBlock(childBlock)
			break
		}
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("type").Exists() && stateJSON.Get("instances").Exists() {
		return m.transformFullResource(result, stateJSON)
	}

	if !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	return m.transformSingleInstance(result, stateJSON), nil
}

func (m *V4ToV5Migrator) transformFullResource(result string, resource gjson.Result) (string, error) {
	resourceType := resource.Get("type").String()
	if resourceType != "cloudflare_spectrum_application" {
		return result, nil
	}

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

func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	if !instance.Get("attributes").Exists() {
		return result
	}

	// Remove id if present in state
	if instance.Get("attributes.id").Exists() {
		result, _ = sjson.Delete(result, "attributes.id")
	}

	return result
}
