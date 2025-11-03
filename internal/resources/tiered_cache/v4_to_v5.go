package tiered_cache

import (
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// V4ToV5Migrator handles migration of Tiered Cache resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_tiered_cache", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_tiered_cache"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_tiered_cache"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Transform cache_type to value and handle value transformations
	// Pattern to match cache_type = "smart" in tiered_cache resources
	cacheTypeSmartPattern := regexp.MustCompile(`(resource\s+"cloudflare_tiered_cache"[^{]+\{[^}]*\n\s*)cache_type(\s*=\s*)"smart"`)
	content = cacheTypeSmartPattern.ReplaceAllString(content, `${1}value${2}"on"`)

	// Transform cache_type = "generic" to value = "generic" (will be handled by config transform)
	cacheTypeGenericPattern := regexp.MustCompile(`(resource\s+"cloudflare_tiered_cache"[^{]+\{[^}]*\n\s*)cache_type(\s*=\s*)"generic"`)
	content = cacheTypeGenericPattern.ReplaceAllString(content, `${1}value${2}"generic"`)

	// Transform cache_type = "off"
	cacheTypeOffPattern := regexp.MustCompile(`(resource\s+"cloudflare_tiered_cache"[^{]+\{[^}]*\n\s*)cache_type(\s*=\s*)"off"`)
	content = cacheTypeOffPattern.ReplaceAllString(content, `${1}value${2}"off"`)

	// Handle any remaining cache_type (like variables)
	cacheTypeVarPattern := regexp.MustCompile(`(resource\s+"cloudflare_tiered_cache"[^{]+\{[^}]*\n\s*)cache_type(\s*=\s*)`)
	content = cacheTypeVarPattern.ReplaceAllString(content, `${1}value${2}`)

	// Also handle value = "smart" that might already be there
	valueSmartPattern := regexp.MustCompile(`(resource\s+"cloudflare_tiered_cache"[^{]+\{[^}]*\n\s*value\s*=\s*)"smart"`)
	content = valueSmartPattern.ReplaceAllString(content, `${1}"on"`)

	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Check if we have value = "generic"
	valueAttr := body.GetAttribute("value")
	if valueAttr == nil {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// Check if the value is "generic"
	tokens := valueAttr.Expr().BuildTokens(nil)
	isGeneric := false
	for _, token := range tokens {
		tokenStr := strings.Trim(string(token.Bytes), `"`)
		if tokenStr == "generic" {
			isGeneric = true
			break
		}
	}

	if !isGeneric {
		return &transform.TransformResult{
			Blocks:         []*hclwrite.Block{block},
			RemoveOriginal: false,
		}, nil
	}

	// Create new argo_tiered_caching resource
	resourceName := block.Labels()[1]
	newResource := hclwrite.NewBlock("resource", []string{"cloudflare_argo_tiered_caching", resourceName})
	newBody := newResource.Body()

	// Set zone_id first to maintain consistent attribute ordering
	if zoneIDAttr := body.GetAttribute("zone_id"); zoneIDAttr != nil {
		tokens := zoneIDAttr.Expr().BuildTokens(nil)
		newBody.SetAttributeRaw("zone_id", tokens)
	}

	// Set value to "on" for argo_tiered_caching
	newBody.SetAttributeValue("value", cty.StringVal("on"))

	// Copy any remaining attributes (excluding zone_id and value which we already set)
	for name, attr := range body.Attributes() {
		if name != "zone_id" && name != "value" {
			tokens := attr.Expr().BuildTokens(nil)
			newBody.SetAttributeRaw(name, tokens)
		}
	}

	// Copy all nested blocks (like lifecycle)
	for _, nestedBlock := range body.Blocks() {
		m.copyBlock(newBody, nestedBlock)
	}

	// Create moved block
	movedBlock := m.createMovedBlock(
		"cloudflare_tiered_cache."+resourceName,
		"cloudflare_argo_tiered_caching."+resourceName,
	)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{newResource, movedBlock},
		RemoveOriginal: true,
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
	if resourceType != "cloudflare_tiered_cache" {
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

// transformSingleInstance transforms a single instance's attributes
func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	cacheType := instance.Get("attributes.cache_type")
	if !cacheType.Exists() {
		return result
	}

	// Remove cache_type attribute
	result, _ = sjson.Delete(result, "attributes.cache_type")

	// Add value attribute based on cache_type
	value := "on" // default for both "smart" and "generic"
	if cacheType.String() == "off" {
		value = "off"
	}
	result, _ = sjson.Set(result, "attributes.value", value)

	return result
}

// copyBlock recursively copies a block to a target body
func (m *V4ToV5Migrator) copyBlock(targetBody *hclwrite.Body, sourceBlock *hclwrite.Block) {
	newBlock := targetBody.AppendNewBlock(sourceBlock.Type(), sourceBlock.Labels())
	newBlockBody := newBlock.Body()

	// Copy all attributes
	for name, attr := range sourceBlock.Body().Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		newBlockBody.SetAttributeRaw(name, tokens)
	}

	// Recursively copy nested blocks
	for _, nestedBlock := range sourceBlock.Body().Blocks() {
		m.copyBlock(newBlockBody, nestedBlock)
	}
}

// createMovedBlock creates a moved block for state migration
func (m *V4ToV5Migrator) createMovedBlock(from, to string) *hclwrite.Block {
	movedBlock := hclwrite.NewBlock("moved", nil)
	movedBody := movedBlock.Body()

	// Create traversal for 'from' and 'to'
	movedBody.SetAttributeTraversal("from", m.parseTraversal(from))
	movedBody.SetAttributeTraversal("to", m.parseTraversal(to))

	return movedBlock
}

// parseTraversal parses a string like "cloudflare_tiered_cache.example" into a traversal
func (m *V4ToV5Migrator) parseTraversal(path string) hcl.Traversal {
	parts := strings.Split(path, ".")
	traversal := hcl.Traversal{}

	if len(parts) > 0 {
		traversal = append(traversal, hcl.TraverseRoot{
			Name: parts[0],
		})
	}

	for i := 1; i < len(parts); i++ {
		traversal = append(traversal, hcl.TraverseAttr{
			Name: parts[i],
		})
	}

	return traversal
}
